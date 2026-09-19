package relay

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"pushly/internal/protocol"
)

// Characters for human-friendly pairing codes (excluding 0, O, 1, I to prevent confusion)
const pairingCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
const pairingCodeLength = 6
const pairingCodeTTL = 5 * time.Minute

// PendingPairing represents a temporary code waiting for a client to claim.
type PendingPairing struct {
	Code      string
	AgentID   string
	AgentName string
	ExpiresAt time.Time
}

// PairedDevice represents an authorized client device.
type PairedDevice struct {
	DeviceID     string    `json:"device_id"`
	DeviceName   string    `json:"device_name"`
	AgentID      string    `json:"agent_id"`
	SessionID    string    `json:"session_id"`
	SessionToken string    `json:"session_token"`
	SharedKeyHex string    `json:"shared_key_hex"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// PairingManager coordinates temporary pairing codes and established device sessions.
type PairingManager struct {
	mu             sync.RWMutex
	pendingCodes   map[string]*PendingPairing // code -> PendingPairing
	sessions       map[string]*PairedDevice   // sessionID -> PairedDevice
	devicesByAgent map[string]map[string]*PairedDevice // agentID -> deviceID -> PairedDevice
}

func NewPairingManager() *PairingManager {
	return &PairingManager{
		pendingCodes:   make(map[string]*PendingPairing),
		sessions:       make(map[string]*PairedDevice),
		devicesByAgent: make(map[string]map[string]*PairedDevice),
	}
}

func (pm *PairingManager) GeneratePairingCode(agentID, agentName string) (string, error) {
	if strings.TrimSpace(agentID) == "" {
		return "", errors.New("agent_id is required")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clean expired codes
	now := time.Now()
	for code, p := range pm.pendingCodes {
		if now.After(p.ExpiresAt) {
			delete(pm.pendingCodes, code)
		}
	}

	// Generate a 6-character code
	code, err := generateRandomCode(pairingCodeLength)
	if err != nil {
		return "", err
	}

	pm.pendingCodes[code] = &PendingPairing{
		Code:      code,
		AgentID:   agentID,
		AgentName: agentName,
		ExpiresAt: now.Add(pairingCodeTTL),
	}

	return code, nil
}

func (pm *PairingManager) CompletePairing(code, deviceID, deviceName string) (*PairedDevice, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, errors.New("pairing code is required")
	}
	if strings.TrimSpace(deviceID) == "" {
		return nil, errors.New("device_id is required")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pending, exists := pm.pendingCodes[code]
	if !exists || time.Now().After(pending.ExpiresAt) {
		delete(pm.pendingCodes, code)
		return nil, errors.New("invalid or expired pairing code")
	}

	// Code is single-use: delete immediately
	delete(pm.pendingCodes, code)

	// Generate session ID, session token, and 256-bit shared encryption key
	sessionID, err := protocol.GenerateRandomHex(16)
	if err != nil {
		return nil, err
	}
	sessionToken, err := protocol.GenerateRandomHex(32)
	if err != nil {
		return nil, err
	}
	sharedKey, err := protocol.GenerateKey256()
	if err != nil {
		return nil, err
	}

	paired := &PairedDevice{
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		AgentID:      pending.AgentID,
		SessionID:    sessionID,
		SessionToken: sessionToken,
		SharedKeyHex: hex.EncodeToString(sharedKey),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

	pm.sessions[sessionID] = paired

	if _, ok := pm.devicesByAgent[pending.AgentID]; !ok {
		pm.devicesByAgent[pending.AgentID] = make(map[string]*PairedDevice)
	}
	pm.devicesByAgent[pending.AgentID][deviceID] = paired

	return paired, nil
}

func (pm *PairingManager) ValidateSession(sessionID, token string) (*PairedDevice, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	paired, exists := pm.sessions[sessionID]
	if !exists {
		return nil, errors.New("invalid session")
	}

	if paired.SessionToken != token {
		return nil, errors.New("unauthorized session token")
	}

	paired.LastActiveAt = time.Now()
	return paired, nil
}

func (pm *PairingManager) ListDevicesForAgent(agentID string) []*PairedDevice {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	devMap, ok := pm.devicesByAgent[agentID]
	if !ok {
		return nil
	}

	result := make([]*PairedDevice, 0, len(devMap))
	for _, dev := range devMap {
		result = append(result, dev)
	}
	return result
}

func (pm *PairingManager) RevokeDevice(agentID, deviceID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	devMap, ok := pm.devicesByAgent[agentID]
	if !ok {
		return fmt.Errorf("no devices registered for agent %q", agentID)
	}

	dev, ok := devMap[deviceID]
	if !ok {
		return fmt.Errorf("device %q not found for agent %q", deviceID, agentID)
	}

	delete(pm.sessions, dev.SessionID)
	delete(devMap, deviceID)
	return nil
}

func generateRandomCode(length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(pairingCharset)))
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("generate random code: %w", err)
		}
		result[i] = pairingCharset[num.Int64()]
	}
	return string(result), nil
}
