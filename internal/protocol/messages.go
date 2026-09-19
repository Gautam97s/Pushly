package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Message types
const (
	TypePing            = "ping"
	TypePong            = "pong"
	TypePairRequest     = "pair_request"
	TypePairResponse    = "pair_response"
	TypeCommandRequest  = "command_request"
	TypeCommandResponse = "command_response"
	TypeError           = "error"
)

// Command actions
const (
	ActionPing          = "ping"
	ActionListFolders   = "list_folders"
	ActionScanProjects  = "scan_projects"
	ActionGetStatus     = "get_status"
	ActionGetDiff       = "get_diff"
	ActionReadFile      = "read_file"
	ActionCommit        = "commit"
	ActionPush          = "push"
)

// Message is the standard communication envelope between Agent, Relay, and Client.
type Message struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	AgentID   string          `json:"agent_id,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Encrypted bool            `json:"encrypted,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func (m *Message) Validate() error {
	if m.ID == "" {
		return errors.New("message id is required")
	}
	if m.Type == "" {
		return errors.New("message type is required")
	}
	if m.Timestamp <= 0 {
		return errors.New("valid timestamp is required")
	}

	// Reject messages with timestamps older than 5 minutes or in future by > 1 minute
	msgTime := time.Unix(m.Timestamp, 0)
	now := time.Now()
	if now.Sub(msgTime) > 5*time.Minute {
		return fmt.Errorf("message expired: timestamp %d is too old", m.Timestamp)
	}
	if msgTime.Sub(now) > 1*time.Minute {
		return fmt.Errorf("message timestamp is from the future")
	}

	return nil
}

// CommandPayload defines a command sent from the client to the agent.
type CommandPayload struct {
	Action             string   `json:"action"`
	RepositoryPath     string   `json:"repository_path,omitempty"`
	Files              []string `json:"files,omitempty"`
	CommitMessage      string   `json:"commit_message,omitempty"`
	ExpectedSnapshotID string   `json:"expected_snapshot_id,omitempty"`
	Remote             string   `json:"remote,omitempty"`
	Branch             string   `json:"branch,omitempty"`
	RelativePath       string   `json:"relative_path,omitempty"`
}

func (c *CommandPayload) Validate() error {
	switch c.Action {
	case ActionPing, ActionListFolders, ActionScanProjects:
		return nil
	case ActionGetStatus, ActionGetDiff:
		if strings.TrimSpace(c.RepositoryPath) == "" {
			return errors.New("repository_path is required")
		}
		return nil
	case ActionReadFile:
		if strings.TrimSpace(c.RepositoryPath) == "" {
			return errors.New("repository_path is required")
		}
		if strings.TrimSpace(c.RelativePath) == "" {
			return errors.New("relative_path is required")
		}
		return nil
	case ActionCommit:
		if strings.TrimSpace(c.RepositoryPath) == "" {
			return errors.New("repository_path is required")
		}
		if len(c.Files) == 0 {
			return errors.New("at least one file must be selected for commit")
		}
		if strings.TrimSpace(c.CommitMessage) == "" {
			return errors.New("commit_message cannot be empty")
		}
		return nil
	case ActionPush:
		if strings.TrimSpace(c.RepositoryPath) == "" {
			return errors.New("repository_path is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported command action: %q", c.Action)
	}
}

// CommandResponsePayload defines the structured response from the agent.
type CommandResponsePayload struct {
	Action    string          `json:"action"`
	Success   bool            `json:"success"`
	Error     string          `json:"error,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// PairRequestPayload is sent by a client to initiate pairing with an agent code.
type PairRequestPayload struct {
	PairingCode string `json:"pairing_code"`
	DeviceID    string `json:"device_id"`
	DeviceName  string `json:"device_name"`
}

// PairResponsePayload is returned to the client upon successful pairing.
type PairResponsePayload struct {
	AgentID      string `json:"agent_id"`
	AgentName    string `json:"agent_name"`
	SessionID    string `json:"session_id"`
	SessionToken string `json:"session_token"`
	SharedKeyHex string `json:"shared_key_hex"`
}
