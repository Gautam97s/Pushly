package protocol

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestEncryptDecryptPayloadRoundTrip(t *testing.T) {
	key, err := GenerateKey256()
	if err != nil {
		t.Fatalf("GenerateKey256 error: %v", err)
	}

	secretPayload := []byte("diff --git a/main.go b/main.go\n+ func Secret() {}")

	encrypted, err := EncryptPayload(secretPayload, key)
	if err != nil {
		t.Fatalf("EncryptPayload error: %v", err)
	}

	if encrypted == string(secretPayload) {
		t.Fatal("encrypted payload must not match plaintext")
	}

	decrypted, err := DecryptPayload(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptPayload error: %v", err)
	}

	if !bytes.Equal(decrypted, secretPayload) {
		t.Fatalf("decrypted = %q, want %q", string(decrypted), string(secretPayload))
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1, _ := GenerateKey256()
	key2, _ := GenerateKey256()

	payload := []byte("confidential source code")
	encrypted, err := EncryptPayload(payload, key1)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = DecryptPayload(encrypted, key2)
	if err == nil {
		t.Fatal("expected decryption with wrong key to fail")
	}
}

func TestReplayCache(t *testing.T) {
	cache := NewReplayCache(100 * time.Millisecond)

	msgID := "msg-12345"

	// First check should succeed
	if !cache.CheckAndRecord(msgID) {
		t.Fatal("first check should be accepted")
	}

	// Immediate replay should fail
	if cache.CheckAndRecord(msgID) {
		t.Fatal("duplicate check should be rejected as replay")
	}

	// Empty ID should fail
	if cache.CheckAndRecord("") {
		t.Fatal("empty ID should be rejected")
	}

	// After TTL expiration, ID can be re-recorded
	time.Sleep(120 * time.Millisecond)
	if !cache.CheckAndRecord(msgID) {
		t.Fatal("check after TTL expiry should be accepted")
	}
}

func TestMessageValidation(t *testing.T) {
	now := time.Now().Unix()

	validMsg := Message{
		ID:        "m1",
		Type:      TypeCommandRequest,
		Timestamp: now,
	}
	if err := validMsg.Validate(); err != nil {
		t.Fatalf("valid message failed validation: %v", err)
	}

	expiredMsg := Message{
		ID:        "m2",
		Type:      TypeCommandRequest,
		Timestamp: now - 400, // > 5 min old
	}
	if err := expiredMsg.Validate(); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got: %v", err)
	}

	futureMsg := Message{
		ID:        "m3",
		Type:      TypeCommandRequest,
		Timestamp: now + 120, // > 1 min in future
	}
	if err := futureMsg.Validate(); err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("expected future error, got: %v", err)
	}
}
