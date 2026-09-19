package relay

import (
	"strings"
	"testing"
	"time"
)

func TestPairingManagerLifecycle(t *testing.T) {
	pm := NewPairingManager()

	agentID := "agent-pc-01"
	agentName := "Gautam's Laptop"

	// 1. Generate code
	code, err := pm.GeneratePairingCode(agentID, agentName)
	if err != nil {
		t.Fatalf("GeneratePairingCode error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}

	// 2. Reject wrong code
	_, err = pm.CompletePairing("WRONG1", "phone-dev-01", "Pixel 9")
	if err == nil {
		t.Fatal("expected error on invalid code")
	}

	// 3. Complete pairing with valid code
	paired, err := pm.CompletePairing(strings.ToLower(code), "phone-dev-01", "Pixel 9")
	if err != nil {
		t.Fatalf("CompletePairing error: %v", err)
	}
	if paired.AgentID != agentID || paired.DeviceID != "phone-dev-01" {
		t.Fatalf("unexpected paired device info: %+v", paired)
	}
	if paired.SessionID == "" || paired.SessionToken == "" || len(paired.SharedKeyHex) != 64 {
		t.Fatalf("session tokens not generated correctly: %+v", paired)
	}

	// 4. Code cannot be reused (one-time use)
	_, err = pm.CompletePairing(code, "phone-dev-02", "iPhone 16")
	if err == nil {
		t.Fatal("expected single-use code to be rejected on reuse")
	}

	// 5. Validate session
	validated, err := pm.ValidateSession(paired.SessionID, paired.SessionToken)
	if err != nil {
		t.Fatalf("ValidateSession error: %v", err)
	}
	if validated.DeviceID != "phone-dev-01" {
		t.Fatalf("validated device mismatch: %+v", validated)
	}

	// 6. Reject bad session token
	_, err = pm.ValidateSession(paired.SessionID, "bad-token")
	if err == nil {
		t.Fatal("expected error on bad session token")
	}

	// 7. List devices
	devices := pm.ListDevicesForAgent(agentID)
	if len(devices) != 1 || devices[0].DeviceID != "phone-dev-01" {
		t.Fatalf("ListDevicesForAgent returned unexpected: %v", devices)
	}

	// 8. Revoke device
	if err := pm.RevokeDevice(agentID, "phone-dev-01"); err != nil {
		t.Fatalf("RevokeDevice error: %v", err)
	}
	if len(pm.ListDevicesForAgent(agentID)) != 0 {
		t.Fatal("expected 0 devices after revocation")
	}

	// 9. Revoked session is no longer valid
	_, err = pm.ValidateSession(paired.SessionID, paired.SessionToken)
	if err == nil {
		t.Fatal("expected revoked session to fail validation")
	}
}

func TestPairingExpiration(t *testing.T) {
	pm := NewPairingManager()

	code, err := pm.GeneratePairingCode("agent-01", "PC")
	if err != nil {
		t.Fatal(err)
	}

	// Manually expire
	pm.mu.Lock()
	pm.pendingCodes[code].ExpiresAt = time.Now().Add(-1 * time.Second)
	pm.mu.Unlock()

	_, err = pm.CompletePairing(code, "device-01", "Phone")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired code error, got: %v", err)
	}
}
