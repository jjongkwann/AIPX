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
			plaintext: "sk_live_1234567890abcdef",
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
