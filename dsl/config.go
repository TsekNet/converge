package dsl

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
)

const encPrefix = "ENC[AES256:"
const encSuffix = "]"

var (
	configKey   []byte
	configKeyMu sync.RWMutex
)

// SetConfigKey sets the AES-256 key used to decrypt ENC[AES256:...] values.
// The key is hashed with SHA-256 to ensure it is exactly 32 bytes.
// Call this once at startup, before any Secret calls.
func SetConfigKey(key string) {
	h := sha256.Sum256([]byte(key))
	configKeyMu.Lock()
	configKey = h[:]
	configKeyMu.Unlock()
}

// configStore holds all registered config maps. Blueprints register
// their config via RegisterConfig, and Secret walks them.
var (
	configStore   []map[string]any
	configStoreMu sync.RWMutex
)

// RegisterConfig adds a config map to the global store. Call from init()
// functions in config Go files. Maps registered later take precedence
// (last-writer-wins for duplicate keys).
func RegisterConfig(m map[string]any) {
	configStoreMu.Lock()
	configStore = append(configStore, m)
	configStoreMu.Unlock()
}

// Secret retrieves a config value by dotted key (e.g., "fleet.enroll_secret").
// If the value is an ENC[AES256:...] string, it is decrypted transparently.
// Returns empty string if the key is not found or decryption fails.
// The Run receiver is unused: Secret reads from the global config store
// so all blueprints share a single config namespace.
func (r *Run) Secret(key string) string {
	configStoreMu.RLock()
	defer configStoreMu.RUnlock()
	return secretFromStores(configStore, key)
}

// secretFromStores walks stores in reverse order (last registered wins)
// and decrypts ENC[AES256:...] values. Used by both Run.Secret and tests.
func secretFromStores(stores []map[string]any, key string) string {
	for i := len(stores) - 1; i >= 0; i-- {
		if v, ok := resolve(stores[i], key); ok {
			return maybeDecrypt(v)
		}
	}
	return ""
}

// resolve walks a nested map using a dotted key path.
func resolve(m map[string]any, key string) (string, bool) {
	parts := strings.Split(key, ".")
	current := any(m)

	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		val, exists := cm[part]
		if !exists {
			return "", false
		}
		if i == len(parts)-1 {
			return fmt.Sprintf("%v", val), true
		}
		current = val
	}
	return "", false
}

// maybeDecrypt checks if a value is encrypted and decrypts it.
// Returns empty string if decryption fails (fail-closed).
func maybeDecrypt(value string) string {
	if !strings.HasPrefix(value, encPrefix) || !strings.HasSuffix(value, encSuffix) {
		return value
	}

	configKeyMu.RLock()
	key := configKey
	configKeyMu.RUnlock()

	if len(key) == 0 {
		return "" // No key configured: fail-closed.
	}

	encoded := value[len(encPrefix) : len(value)-len(encSuffix)]
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "" // Malformed: fail-closed.
	}

	plaintext, err := decryptAESGCM(key, ciphertext)
	if err != nil {
		return "" // Wrong key or corrupted: fail-closed.
	}

	return string(plaintext)
}

// Encrypt encrypts a plaintext value with AES-256-GCM using the configured key.
// Returns the ENC[AES256:...] wrapped string. Uses a random nonce for each
// call, so encrypting the same value twice produces different ciphertexts.
func Encrypt(plaintext string) (string, error) {
	configKeyMu.RLock()
	key := configKey
	configKeyMu.RUnlock()

	if len(key) == 0 {
		return "", fmt.Errorf("no config key set: call SetConfigKey first")
	}

	ciphertext, err := encryptAESGCM(key, []byte(plaintext))
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encPrefix + encoded + encSuffix, nil
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
