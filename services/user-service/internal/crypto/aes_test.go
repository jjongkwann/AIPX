package crypto

import (
	"strings"
	"testing"
)

func TestNewAESGCMEncryptor(t *testing.T) {
	tests := []struct {
		name        string
		keySize     int
		shouldError bool
	}{
		{
			name:        "valid 256-bit key",
			keySize:     32,
			shouldError: false,
		},
		{
			name:        "invalid 128-bit key",
			keySize:     16,
			shouldError: true,
		},
		{
			name:        "invalid 192-bit key",
			keySize:     24,
			shouldError: true,
		},
		{
			name:        "empty key",
			keySize:     0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewAESGCMEncryptor(key)

			if tt.shouldError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAESGCMEncryptor_EncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "api key",
			plaintext: "test_api_key_1234567890abcdef",
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("A", 1000),
		},
		{
			name:      "unicode text",
			plaintext: "こんにちは世界",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Ciphertext should be different from plaintext
			if ciphertext == tt.plaintext {
				t.Error("ciphertext should not match plaintext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// Verify decrypted matches original
			if decrypted != tt.plaintext {
				t.Errorf("decrypted text does not match original\nwant: %s\ngot:  %s",
					tt.plaintext, decrypted)
			}
		})
	}
}

func TestAESGCMEncryptor_EncryptUniqueness(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "Same plaintext"

	// Encrypt same plaintext multiple times
	ciphertext1, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first encryption failed: %v", err)
	}

	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second encryption failed: %v", err)
	}

	// Ciphertexts should be different due to random nonces
	if ciphertext1 == ciphertext2 {
		t.Error("encrypting same plaintext should produce different ciphertexts")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := encryptor.Decrypt(ciphertext1)
	decrypted2, _ := encryptor.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("decryption failed to recover original plaintext")
	}
}

func TestAESGCMEncryptor_InvalidInputs(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	t.Run("empty plaintext", func(t *testing.T) {
		_, err := encryptor.Encrypt("")
		if err == nil {
			t.Error("expected error for empty plaintext")
		}
	})

	t.Run("empty ciphertext", func(t *testing.T) {
		_, err := encryptor.Decrypt("")
		if err == nil {
			t.Error("expected error for empty ciphertext")
		}
	})

	t.Run("invalid ciphertext", func(t *testing.T) {
		_, err := encryptor.Decrypt("not-base64-encoded!")
		if err == nil {
			t.Error("expected error for invalid ciphertext")
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		ciphertext, _ := encryptor.Encrypt("valid plaintext")
		// Tamper with the ciphertext
		tampered := ciphertext[:len(ciphertext)-1] + "X"
		_, err := encryptor.Decrypt(tampered)
		if err == nil {
			t.Error("expected error for tampered ciphertext")
		}
	})
}

func TestRotatableEncryptor(t *testing.T) {
	currentKey, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate current key: %v", err)
	}

	previousKey, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate previous key: %v", err)
	}

	// Create encryptor with old key
	oldEncryptor, err := NewAESGCMEncryptor(previousKey)
	if err != nil {
		t.Fatalf("failed to create old encryptor: %v", err)
	}

	// Encrypt with old key
	plaintext := "sensitive data"
	oldCiphertext, err := oldEncryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt with old key: %v", err)
	}

	// Create rotatable encryptor with both keys
	rotatable, err := NewRotatableEncryptor(currentKey, previousKey)
	if err != nil {
		t.Fatalf("failed to create rotatable encryptor: %v", err)
	}

	t.Run("decrypt with previous key", func(t *testing.T) {
		decrypted, err := rotatable.Decrypt(oldCiphertext)
		if err != nil {
			t.Fatalf("failed to decrypt with previous key: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("decrypted text does not match original\nwant: %s\ngot:  %s",
				plaintext, decrypted)
		}
	})

	t.Run("encrypt with current key", func(t *testing.T) {
		newCiphertext, err := rotatable.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("failed to encrypt with current key: %v", err)
		}

		// Verify it's different from old ciphertext
		if newCiphertext == oldCiphertext {
			t.Error("new encryption should produce different ciphertext")
		}

		// Verify it can be decrypted
		decrypted, err := rotatable.Decrypt(newCiphertext)
		if err != nil {
			t.Fatalf("failed to decrypt new ciphertext: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("decrypted text does not match original")
		}
	})

	t.Run("re-encrypt", func(t *testing.T) {
		newCiphertext, err := rotatable.ReEncrypt(oldCiphertext)
		if err != nil {
			t.Fatalf("failed to re-encrypt: %v", err)
		}

		// Verify new ciphertext decrypts correctly
		decrypted, err := rotatable.Decrypt(newCiphertext)
		if err != nil {
			t.Fatalf("failed to decrypt re-encrypted data: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("re-encrypted data does not match original")
		}
	})
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate first key: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("expected key length 32, got %d", len(key1))
	}

	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate second key: %v", err)
	}

	// Keys should be different
	if string(key1) == string(key2) {
		t.Error("generated keys should be unique")
	}
}

func TestGenerateKeyString(t *testing.T) {
	keyStr, err := GenerateKeyString()
	if err != nil {
		t.Fatalf("failed to generate key string: %v", err)
	}

	// Verify it can be used to create an encryptor
	_, err = NewAESGCMEncryptorFromString(keyStr)
	if err != nil {
		t.Errorf("failed to create encryptor from generated key: %v", err)
	}
}

func BenchmarkAESGCMEncryptor_Encrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)
	plaintext := "Benchmark plaintext for encryption"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkAESGCMEncryptor_Decrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)
	plaintext := "Benchmark plaintext for decryption"
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

func TestAESGCMEncryptor_NonceUniqueness(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "Same plaintext"
	nonces := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encryption failed at iteration %d: %v", i, err)
		}

		// Nonce is first 12 bytes of ciphertext (after base64 decode)
		// For this test, we just verify ciphertext uniqueness
		if nonces[ciphertext] {
			t.Errorf("Duplicate ciphertext detected at iteration %d", i)
		}
		nonces[ciphertext] = true
	}

	if len(nonces) != iterations {
		t.Errorf("Generated %d unique ciphertexts, want %d", len(nonces), iterations)
	}
}

func TestAESGCMEncryptor_WrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	encryptor1, _ := NewAESGCMEncryptor(key1)
	encryptor2, _ := NewAESGCMEncryptor(key2)

	plaintext := "sensitive data"
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("decryption should fail with wrong key")
	}

	// Error should be wrapped ErrDecryption
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("expected decryption error, got %v", err)
	}
}

func TestAESGCMEncryptor_IntegrityCheck(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	plaintext := "important data"
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with different parts of the ciphertext
	tests := []struct {
		name   string
		tamper func(string) string
	}{
		{
			name: "flip first bit",
			tamper: func(s string) string {
				if len(s) == 0 {
					return s
				}
				b := []byte(s)
				b[0] ^= 1
				return string(b)
			},
		},
		{
			name: "flip last bit",
			tamper: func(s string) string {
				if len(s) == 0 {
					return s
				}
				b := []byte(s)
				b[len(b)-1] ^= 1
				return string(b)
			},
		},
		{
			name: "truncate",
			tamper: func(s string) string {
				if len(s) <= 1 {
					return s
				}
				return s[:len(s)-1]
			},
		},
		{
			name: "append",
			tamper: func(s string) string {
				return s + "A"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := tt.tamper(ciphertext)
			_, err := encryptor.Decrypt(tampered)
			if err == nil {
				t.Error("decryption should fail for tampered ciphertext")
			}
		})
	}
}

func TestAESGCMEncryptor_SensitiveData(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	sensitiveData := []string{
		"test_api_key_51HxLPqJR7zLfGCxWgVxoQeU8Lk2FsH6ky3G9Rf8H4e", // API key
		"test_public_key_51HxLPqJR7zLfGCxWgVxoQeU8Lk2FsH6ky3G9Rf8H4e", // Public key
		"test_slack_token_1234567890_1234567890123_AbCdEfGhIjKlMnOpQrStUvWx", // Slack token
		"test_github_token_1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", // GitHub token
	}

	for _, data := range sensitiveData {
		t.Run("encrypt_"+data[:10], func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt(data)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Verify ciphertext doesn't contain plaintext
			if strings.Contains(ciphertext, data) {
				t.Error("ciphertext contains plaintext data")
			}

			// Verify decryption
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			if decrypted != data {
				t.Error("decrypted data doesn't match original")
			}
		})
	}
}

func TestAESGCMEncryptor_NonceSize(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	plaintext := "test data"
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Decode base64 to check nonce size
	// GCM nonce should be 12 bytes
	// The ciphertext format is: nonce || ciphertext || tag
	// Total base64 encoded length should be reasonable
	if len(ciphertext) < 16 { // Minimum base64 encoded size
		t.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}
}

func TestAESGCMEncryptor_AuthenticationTag(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	plaintext := "authenticated data"
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// GCM automatically includes 16-byte authentication tag
	// Verify ciphertext is longer than plaintext + nonce
	// Base64 encoding expands by ~33%
	minExpectedLength := ((12 + len(plaintext) + 16) * 4 / 3)
	if len(ciphertext) < minExpectedLength {
		t.Errorf("ciphertext length %d too short, expected at least %d",
			len(ciphertext), minExpectedLength)
	}
}

func TestRotatableEncryptor_WithoutPreviousKey(t *testing.T) {
	currentKey, _ := GenerateKey()

	// Create rotatable encryptor without previous key
	rotatable, err := NewRotatableEncryptor(currentKey, nil)
	if err != nil {
		t.Fatalf("failed to create rotatable encryptor: %v", err)
	}

	plaintext := "test data"

	// Encrypt with current key
	ciphertext, err := rotatable.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Decrypt should work
	decrypted, err := rotatable.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("decrypted data doesn't match original")
	}
}

func TestAESGCMEncryptor_ConcurrentEncryption(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAESGCMEncryptor(key)

	plaintext := "concurrent test"
	iterations := 100

	// Test concurrent encryption
	done := make(chan bool, iterations)
	ciphertexts := make(chan string, iterations)

	for i := 0; i < iterations; i++ {
		go func() {
			ciphertext, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Errorf("concurrent encryption failed: %v", err)
			}
			ciphertexts <- ciphertext
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < iterations; i++ {
		<-done
	}
	close(ciphertexts)

	// Verify all ciphertexts are unique and decrypt correctly
	seen := make(map[string]bool)
	count := 0
	for ciphertext := range ciphertexts {
		if seen[ciphertext] {
			t.Error("duplicate ciphertext in concurrent encryption")
		}
		seen[ciphertext] = true

		decrypted, err := encryptor.Decrypt(ciphertext)
		if err != nil {
			t.Errorf("decryption failed: %v", err)
		}
		if decrypted != plaintext {
			t.Error("decrypted data doesn't match original")
		}
		count++
	}

	if count != iterations {
		t.Errorf("expected %d ciphertexts, got %d", iterations, count)
	}
}
