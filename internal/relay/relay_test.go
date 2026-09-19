package relay

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pushly/internal/agent"
	"pushly/internal/config"
	"pushly/internal/git"
	"pushly/internal/protocol"
)

type mockRunner struct {
	repoRoot string
	status   string
}

func (m *mockRunner) Run(_ context.Context, _ string, args []string) (string, error) {
	if len(args) > 0 && args[0] == "rev-parse" {
		return m.repoRoot, nil
	}
	if len(args) > 0 && args[0] == "status" {
		return m.status, nil
	}
	if len(args) > 0 && args[0] == "diff" {
		return "diff --git a/a.txt b/a.txt\n+hello world\n", nil
	}
	return "ok", nil
}

func TestEndToEndRelayPairingAndEncryptedCommand(t *testing.T) {
	// 1. Start Relay Server
	server := NewRelayServer()
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// 2. Setup Agent with approved directory
	tmpDir := t.TempDir()
	approvedDir := filepath.Join(tmpDir, "Projects")
	repoDir := filepath.Join(approvedDir, "App")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := &config.FolderStore{}
	if err := store.AddFolder(approvedDir); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		repoRoot: repoDir,
		status:   "# branch.head feature\x00",
	}
	gitClient := git.Client{Runner: runner, Timeout: 5 * time.Second}
	ag := agent.NewAgentWithStore(store, gitClient, nil)

	// 3. Start Agent Relay Client
	agentID := "agent-test-123"
	rc := agent.NewRelayClient(ag, ts.URL, agentID, "Test Computer")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = rc.Run(ctx)
	}()

	// Wait briefly for SSE stream to connect
	time.Sleep(100 * time.Millisecond)

	// 4. Request pairing code from Relay
	pairingCode, err := rc.RequestPairingCode(ctx)
	if err != nil {
		t.Fatalf("RequestPairingCode error: %v", err)
	}
	if len(pairingCode) != 6 {
		t.Fatalf("unexpected pairing code: %s", pairingCode)
	}

	// 5. Client pairs with Relay using code
	pairReqBody, _ := json.Marshal(protocol.PairRequestPayload{
		PairingCode: pairingCode,
		DeviceID:    "mobile-client-01",
		DeviceName:  "Test Phone",
	})
	pairResp, err := http.Post(ts.URL+"/api/v1/client/pair", "application/json", bytes.NewReader(pairReqBody))
	if err != nil {
		t.Fatalf("client pair request error: %v", err)
	}
	defer pairResp.Body.Close()

	if pairResp.StatusCode != http.StatusOK {
		t.Fatalf("client pair returned status %d", pairResp.StatusCode)
	}

	var pairResult protocol.PairResponsePayload
	if err := json.NewDecoder(pairResp.Body).Decode(&pairResult); err != nil {
		t.Fatalf("decode pair response error: %v", err)
	}

	sharedKey, err := hex.DecodeString(pairResult.SharedKeyHex)
	if err != nil {
		t.Fatalf("decode shared key error: %v", err)
	}

	// Agent registers session key
	rc.RegisterSessionKey(pairResult.SessionID, sharedKey)

	// 6. Client sends Encrypted "get_diff" command
	cmdPayload := protocol.CommandPayload{
		Action:         protocol.ActionGetDiff,
		RepositoryPath: repoDir,
	}
	rawCmd, _ := json.Marshal(cmdPayload)

	encryptedCmd, err := protocol.EncryptPayload(rawCmd, sharedKey)
	if err != nil {
		t.Fatalf("encrypt command error: %v", err)
	}

	msgID, _ := protocol.GenerateRandomHex(8)
	encryptedPayloadJSON, _ := json.Marshal(encryptedCmd)

	msg := protocol.Message{
		ID:        msgID,
		Type:      protocol.TypeCommandRequest,
		SessionID: pairResult.SessionID,
		Timestamp: time.Now().Unix(),
		Encrypted: true,
		Payload:   encryptedPayloadJSON,
	}

	msgBody, _ := json.Marshal(msg)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/client/command", bytes.NewReader(msgBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", pairResult.SessionToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client command error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("client command returned status %d", resp.StatusCode)
	}

	var responseMsg protocol.Message
	if err := json.NewDecoder(resp.Body).Decode(&responseMsg); err != nil {
		t.Fatalf("decode command response error: %v", err)
	}

	if responseMsg.Error != "" {
		t.Fatalf("response message returned error: %s", responseMsg.Error)
	}
	if !responseMsg.Encrypted {
		t.Fatal("response message should be encrypted")
	}

	// 7. Client decrypts response
	var responseCiphertext string
	if err := json.Unmarshal(responseMsg.Payload, &responseCiphertext); err != nil {
		t.Fatalf("unmarshal ciphertext error: %v", err)
	}

	decryptedResponse, err := protocol.DecryptPayload(responseCiphertext, sharedKey)
	if err != nil {
		t.Fatalf("decrypt response error: %v", err)
	}

	var cmdResp protocol.CommandResponsePayload
	if err := json.Unmarshal(decryptedResponse, &cmdResp); err != nil {
		t.Fatalf("unmarshal decrypted response error: %v", err)
	}

	if !cmdResp.Success {
		t.Fatalf("command failed: %s", cmdResp.Error)
	}

	var diffData map[string]string
	if err := json.Unmarshal(cmdResp.Data, &diffData); err != nil {
		t.Fatalf("unmarshal diff data error: %v", err)
	}

	if diffData["diff"] != "diff --git a/a.txt b/a.txt\n+hello world\n" {
		t.Fatalf("unexpected diff: %q", diffData["diff"])
	}
}
