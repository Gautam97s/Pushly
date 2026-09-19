package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// GenerateRandomBytes generates cryptographically secure random bytes of given length.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomHex generates a random hex-encoded string of n random bytes.
func GenerateRandomHex(n int) (string, error) {
	bytes, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateKey256 generates a 32-byte (256-bit) AES key.
func GenerateKey256() ([]byte, error) {
	return GenerateRandomBytes(32)
}

// EncryptPayload encrypts plaintext bytes with an AES-256-GCM key and returns base64 ciphertext (nonce + ciphertext).
func EncryptPayload(plaintext []byte, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("invalid key length: got %d bytes, want 32 bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPayload decrypts a base64 ciphertext string (nonce + ciphertext) using an AES-256-GCM key.
func DecryptPayload(ciphertextBase64 string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: got %d bytes, want 32 bytes", len(key))
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}

// ReplayCache tracks message IDs to prevent replay attacks within a sliding TTL window.
type ReplayCache struct {
	mu  sync.Mutex
	ttl time.Duration
	ids map[string]time.Time
}

// NewReplayCache creates a replay cache with the specified TTL.
func NewReplayCache(ttl time.Duration) *ReplayCache {
	return &ReplayCache{
		ttl: ttl,
		ids: make(map[string]time.Time),
	}
}

// CheckAndRecord returns true if the message ID is fresh (not seen before), and records it.
// Returns false if the message ID was already processed.
func (c *ReplayCache) CheckAndRecord(id string) bool {
	if id == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// Cleanup expired entries
	for k, exp := range c.ids {
		if now.After(exp) {
			delete(c.ids, k)
		}
	}

	if _, exists := c.ids[id]; exists {
		return false
	}

	c.ids[id] = now.Add(c.ttl)
	return true
}
