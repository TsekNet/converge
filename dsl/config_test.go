package dsl

import (
	"strings"
	"testing"
)

func TestSecret_FlatKey(t *testing.T) {
	t.Parallel()

	store := []map[string]any{{
		"server_url": "https://fleet.example.com",
		"debug":      true,
	}}

	tests := []struct {
		key  string
		want string
	}{
		{"server_url", "https://fleet.example.com"},
		{"debug", "true"},
		{"missing", ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := secretFromStores(store, tt.key)
			if got != tt.want {
				t.Errorf("secretFromStores(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestSecret_NestedKey(t *testing.T) {
	t.Parallel()

	store := []map[string]any{{
		"fleet": map[string]any{
			"server_url":    "https://fleet.example.com",
			"enroll_secret": "secret123",
		},
		"datadog": map[string]any{
			"api_key": "dd-key-456",
		},
	}}

	tests := []struct {
		key  string
		want string
	}{
		{"fleet.server_url", "https://fleet.example.com"},
		{"fleet.enroll_secret", "secret123"},
		{"datadog.api_key", "dd-key-456"},
		{"fleet.missing", ""},
		{"nonexistent.key", ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := secretFromStores(store, tt.key)
			if got != tt.want {
				t.Errorf("secretFromStores(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestSecret_LastRegisteredWins(t *testing.T) {
	t.Parallel()

	stores := []map[string]any{
		{"key": "first"},
		{"key": "second"},
	}

	got := secretFromStores(stores, "key")
	if got != "second" {
		t.Errorf("secretFromStores(\"key\") = %q, want %q (last registered should win)", got, "second")
	}
}

func TestSecret_FallbackToPreviousStore(t *testing.T) {
	t.Parallel()

	stores := []map[string]any{
		{"base_key": "base_value"},
		{"override_key": "override_value"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"base_key", "base_value"},
		{"override_key", "override_value"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := secretFromStores(stores, tt.key)
			if got != tt.want {
				t.Errorf("secretFromStores(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	// Not parallel: mutates global configKey.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	SetConfigKey("test-encryption-key")

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple string", "hello world"},
		{"empty string", ""},
		{"special chars", "p@$$w0rd!#%&*"},
		{"long string", strings.Repeat("a", 1000)},
		{"unicode", "hëllo wörld"},
		{"json", `{"api_key": "secret-123"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encrypted, err := Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if !strings.HasPrefix(encrypted, encPrefix) {
				t.Errorf("encrypted should start with %q, got %q", encPrefix, encrypted)
			}
			if !strings.HasSuffix(encrypted, encSuffix) {
				t.Errorf("encrypted should end with %q, got %q", encSuffix, encrypted)
			}

			got := maybeDecrypt(encrypted)
			if got != tt.plaintext {
				t.Errorf("roundtrip: got %q, want %q", got, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_RandomNonce(t *testing.T) {
	// Not parallel: mutates global configKey.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	SetConfigKey("random-nonce-key")

	enc1, err := Encrypt("test-value")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := Encrypt("test-value")
	if err != nil {
		t.Fatal(err)
	}
	if enc1 == enc2 {
		t.Error("Encrypt should produce different ciphertexts (random nonce)")
	}
	// Both should still decrypt to the same value.
	got1 := maybeDecrypt(enc1)
	got2 := maybeDecrypt(enc2)
	if got1 != "test-value" || got2 != "test-value" {
		t.Errorf("both ciphertexts should decrypt: got %q and %q", got1, got2)
	}
}

func TestEncrypt_NoKey(t *testing.T) {
	// Not parallel: mutates global configKey.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	configKeyMu.Lock()
	configKey = nil
	configKeyMu.Unlock()

	_, err := Encrypt("test")
	if err == nil {
		t.Error("Encrypt() should fail without a key")
	}

	// Decrypting a valid-looking ENC value with no key should fail-closed.
	got := maybeDecrypt("ENC[AES256:dGVzdA==]")
	if got != "" {
		t.Errorf("maybeDecrypt with no key should return empty, got %q", got)
	}
}

func TestMaybeDecrypt_PlainValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"plain string", "just a string"},
		{"url", "https://fleet.example.com"},
		{"number", "42"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := maybeDecrypt(tt.value)
			if got != tt.value {
				t.Errorf("maybeDecrypt(%q) = %q, want unchanged", tt.value, got)
			}
		})
	}
}

func TestMaybeDecrypt_MalformedEnc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"bad prefix", "ENC[AES128:baddata]", "ENC[AES128:baddata]"}, // Doesn't match ENC[AES256: prefix, returned as-is.
		{"no suffix", "ENC[AES256:baddata", "ENC[AES256:baddata"},    // Doesn't match suffix, returned as-is.
		{"bad base64", "ENC[AES256:not-valid-base64!@#]", ""},        // Matches prefix+suffix, fails decode: fail-closed.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := maybeDecrypt(tt.value)
			if got != tt.want {
				t.Errorf("maybeDecrypt(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestSecret_WithEncryptedValue(t *testing.T) {
	// Not parallel: mutates global configKey.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	SetConfigKey("lookup-test-key")

	encrypted, err := Encrypt("my-secret-value")
	if err != nil {
		t.Fatal(err)
	}

	stores := []map[string]any{{
		"fleet": map[string]any{
			"enroll_secret": encrypted,
			"server_url":    "https://fleet.example.com",
		},
	}}

	tests := []struct {
		key  string
		want string
	}{
		{"fleet.enroll_secret", "my-secret-value"},
		{"fleet.server_url", "https://fleet.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := secretFromStores(stores, tt.key)
			if got != tt.want {
				t.Errorf("secretFromStores(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestRun_Secret(t *testing.T) {
	// Not parallel: touches global configStore.
	origStore := configStore
	t.Cleanup(func() {
		configStoreMu.Lock()
		configStore = origStore
		configStoreMu.Unlock()
	})

	configStoreMu.Lock()
	configStore = nil
	configStoreMu.Unlock()

	RegisterConfig(map[string]any{"test_key": "test_value"})

	run := newRun(New())
	got := run.Secret("test_key")
	if got != "test_value" {
		t.Errorf("Secret(%q) = %q, want %q", "test_key", got, "test_value")
	}
}

func TestSetConfigKey_NormalizesLength(t *testing.T) {
	// Not parallel: mutates global configKey.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	// Short key should be SHA-256 hashed to 32 bytes.
	SetConfigKey("short")
	configKeyMu.RLock()
	keyLen := len(configKey)
	configKeyMu.RUnlock()

	if keyLen != 32 {
		t.Errorf("key length = %d, want 32", keyLen)
	}

	// Long key should also be 32 bytes.
	SetConfigKey(strings.Repeat("x", 1000))
	configKeyMu.RLock()
	keyLen = len(configKey)
	configKeyMu.RUnlock()

	if keyLen != 32 {
		t.Errorf("key length = %d, want 32", keyLen)
	}
}

func TestEncrypt_DifferentKeys(t *testing.T) {
	// Not parallel: mutates global configKey sequentially.
	origKey := configKey
	t.Cleanup(func() {
		configKeyMu.Lock()
		configKey = origKey
		configKeyMu.Unlock()
	})

	SetConfigKey("key-alpha")
	enc1, _ := Encrypt("secret")

	SetConfigKey("key-beta")
	enc2, _ := Encrypt("secret")

	if enc1 == enc2 {
		t.Error("different keys should produce different ciphertexts")
	}

	// Decrypting enc1 with key-beta should fail-closed (empty string).
	got := maybeDecrypt(enc1)
	if got != "" {
		t.Errorf("decrypting with wrong key should return empty, got %q", got)
	}
}
