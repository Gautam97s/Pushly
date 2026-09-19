package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"pushly/internal/git"
	"pushly/internal/protocol"
	"pushly/internal/scanner"
)

type RelayClient struct {
	Agent       *Agent
	RelayURL    string
	AgentID     string
	AgentName   string
	httpClient  *http.Client
	sessionMu   sync.RWMutex
	sessionKeys map[string][]byte // sessionID -> AES key
}

func NewRelayClient(ag *Agent, relayURL, agentID, agentName string) *RelayClient {
	if agentID == "" {
		agentID = "agent-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if agentName == "" {
		agentName = "Pushly Agent"
	}
	relayURL = strings.TrimRight(relayURL, "/")

	return &RelayClient{
		Agent:       ag,
		RelayURL:    relayURL,
		AgentID:     agentID,
		AgentName:   agentName,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		sessionKeys: make(map[string][]byte),
	}
}

func (rc *RelayClient) RegisterSessionKey(sessionID string, key []byte) {
	rc.sessionMu.Lock()
	defer rc.sessionMu.Unlock()
	rc.sessionKeys[sessionID] = key
}

func (rc *RelayClient) RegisterSessionKeyHex(sessionID string, keyHex string) error {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("decode hex key: %w", err)
	}
	rc.RegisterSessionKey(sessionID, key)
	return nil
}

func (rc *RelayClient) getSessionKey(sessionID string) ([]byte, bool) {
	rc.sessionMu.RLock()
	defer rc.sessionMu.RUnlock()
	key, ok := rc.sessionKeys[sessionID]
	return key, ok
}

func (rc *RelayClient) RequestPairingCode(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v1/agent/pair/code", rc.RelayURL)
	body, _ := json.Marshal(map[string]string{
		"agent_id":   rc.AgentID,
		"agent_name": rc.AgentName,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request pairing code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("relay returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		PairingCode string `json:"pairing_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode pairing response: %w", err)
	}

	return result.PairingCode, nil
}

func (rc *RelayClient) Run(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := rc.connectAndListen(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		} else {
			backoff = 1 * time.Second
		}
	}
}

func (rc *RelayClient) connectAndListen(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/agent/events?agent_id=%s", rc.RelayURL, rc.AgentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to relay event stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay stream returned %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "{}" || data == "" {
				continue
			}

			var msg protocol.Message
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				continue
			}

			go rc.handleMessage(ctx, &msg)
		}
	}
}

func (rc *RelayClient) handleMessage(ctx context.Context, msg *protocol.Message) {
	resp := &protocol.Message{
		ID:        msg.ID,
		Type:      protocol.TypeCommandResponse,
		AgentID:   rc.AgentID,
		DeviceID:  msg.DeviceID,
		SessionID: msg.SessionID,
		Timestamp: time.Now().Unix(),
		Encrypted: msg.Encrypted,
	}

	payloadBytes := msg.Payload
	var sessionKey []byte
	var hasKey bool

	if msg.Encrypted && msg.SessionID != "" {
		sessionKey, hasKey = rc.getSessionKey(msg.SessionID)
		if hasKey {
			var ciphertext string
			if err := json.Unmarshal(payloadBytes, &ciphertext); err == nil {
				decrypted, err := protocol.DecryptPayload(ciphertext, sessionKey)
				if err == nil {
					payloadBytes = decrypted
				}
			}
		}
	}

	var cmd protocol.CommandPayload
	if err := json.Unmarshal(payloadBytes, &cmd); err != nil {
		resp.Error = "invalid command payload: " + err.Error()
		rc.sendResponse(ctx, resp)
		return
	}

	cmdResp := rc.executeCommand(ctx, cmd)
	rawResult, _ := json.Marshal(cmdResp)

	if msg.Encrypted && hasKey {
		ciphertext, err := protocol.EncryptPayload(rawResult, sessionKey)
		if err != nil {
			resp.Error = "encrypt response error: " + err.Error()
		} else {
			resp.Payload, _ = json.Marshal(ciphertext)
		}
	} else {
		resp.Payload = rawResult
	}

	rc.sendResponse(ctx, resp)
}

func (rc *RelayClient) executeCommand(ctx context.Context, cmd protocol.CommandPayload) protocol.CommandResponsePayload {
	res := protocol.CommandResponsePayload{
		Action:    cmd.Action,
		Timestamp: time.Now().Unix(),
	}

	switch cmd.Action {
	case protocol.ActionPing:
		res.Success = true
		res.Data, _ = json.Marshal(map[string]string{"pong": "true"})

	case protocol.ActionListFolders:
		folders := rc.Agent.ListFolders()
		res.Success = true
		res.Data, _ = json.Marshal(folders)

	case protocol.ActionScanProjects:
		scanResults, err := rc.Agent.ScanApprovedFolders(scanner.Options{MaxDepth: 8, MaxRepositories: 1000})
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
			res.Data, _ = json.Marshal(scanResults)
		}

	case protocol.ActionGetStatus:
		status, err := rc.Agent.GetStatus(ctx, cmd.RepositoryPath)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
			res.Data, _ = json.Marshal(status)
		}

	case protocol.ActionGetDiff:
		diff, err := rc.Agent.GetDiff(ctx, cmd.RepositoryPath)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
			res.Data, _ = json.Marshal(map[string]string{"diff": diff})
		}

	case protocol.ActionReadFile:
		content, err := rc.Agent.ReadFile(ctx, cmd.RepositoryPath, cmd.RelativePath)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
			res.Data, _ = json.Marshal(map[string]string{"content": string(content)})
		}

	case protocol.ActionCommit:
		commitRes, err := rc.Agent.Commit(ctx, cmd.RepositoryPath, git.CommitOptions{
			SelectedFiles:      cmd.Files,
			Message:            cmd.CommitMessage,
			ExpectedSnapshotID: cmd.ExpectedSnapshotID,
		})
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = commitRes.Success
			res.Data, _ = json.Marshal(commitRes)
		}

	case protocol.ActionPush:
		pushRes, err := rc.Agent.Push(ctx, cmd.RepositoryPath, git.PushOptions{
			Remote: cmd.Remote,
			Branch: cmd.Branch,
		})
		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = pushRes.Success
			res.Data, _ = json.Marshal(pushRes)
		}

	default:
		res.Success = false
		res.Error = fmt.Sprintf("unsupported action: %q", cmd.Action)
	}

	return res
}

func (rc *RelayClient) sendResponse(ctx context.Context, msg *protocol.Message) {
	url := fmt.Sprintf("%s/api/v1/agent/response", rc.RelayURL)
	data, _ := json.Marshal(msg)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rc.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
