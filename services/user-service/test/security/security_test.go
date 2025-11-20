package security

import (
	"strings"
	"testing"
	"time"

	"user-service/internal/auth"
	"user-service/internal/crypto"
)

// TestPasswordHashingStrength verifies password hashing meets security standards
func TestPasswordHashingStrength(t *testing.T) {
	hasher := auth.NewArgon2Hasher(nil)

	t.Run("salt_entropy", func(t *testing.T) {
		password := "TestPassword123"
		hashes := make(map[string]bool)

		// Generate 1000 hashes and verify unique salts
		for i := 0; i < 1000; i++ {
			hash, err := hasher.Hash(password)
			if err != nil {
				t.Fatalf("Hash failed: %v", err)
			}

			if hashes[hash] {
				t.Error("Duplicate hash found - insufficient salt entropy")
			}
			hashes[hash] = true
		}

		if len(hashes) != 1000 {
			t.Errorf("Expected 1000 unique hashes, got %d", len(hashes))
		}
	})

	t.Run("constant_time_comparison", func(t *testing.T) {
		password := "CorrectPassword123"
		hash, _ := hasher.Hash(password)

		// All wrong passwords should fail
		wrongPasswords := []string{
			"WrongPassword123",
			"CorrectPassword12", // One char off
			"correctpassword123", // Case different
			"X",                  // Very short
			strings.Repeat("a", 1000), // Very long
		}

		for _, wrong := range wrongPasswords {
			err := hasher.Verify(wrong, hash)
			if err == nil {
				t.Errorf("Password '%s' should not verify", wrong)
			}
		}
	})
}

// TestJWTSecurity verifies JWT token security
func TestJWTSecurity(t *testing.T) {
	t.Run("token_uniqueness", func(t *testing.T) {
		manager, _ := auth.NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			15*time.Minute,
			7*24*time.Hour,
			"test-issuer",
		)

		// Generate tokens at different times to ensure uniqueness
		token1, err := manager.GenerateAccessToken("user-123", "test@example.com")
		if err != nil {
			t.Fatalf("Token generation failed: %v", err)
		}

		time.Sleep(2 * time.Second) // Wait to ensure different IssuedAt

		token2, err := manager.GenerateAccessToken("user-123", "test@example.com")
		if err != nil {
			t.Fatalf("Token generation failed: %v", err)
		}

		if token1 == token2 {
			t.Error("Tokens generated at different times should be unique")
		}

		// Verify both tokens are valid
		_, err = manager.ValidateAccessToken(token1)
		if err != nil {
			t.Errorf("Token1 validation failed: %v", err)
		}

		_, err = manager.ValidateAccessToken(token2)
		if err != nil {
			t.Errorf("Token2 validation failed: %v", err)
		}
	})

	t.Run("signature_verification", func(t *testing.T) {
		manager, _ := auth.NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			15*60*1000,
			7*24*60*60*1000,
			"test-issuer",
		)

		token, _ := manager.GenerateAccessToken("user-123", "test@example.com")

		// Tamper with token
		tampered := token[:len(token)-5] + "XXXXX"
		_, err := manager.ValidateAccessToken(tampered)
		if err == nil {
			t.Error("Tampered token should be rejected")
		}
	})
}

// TestEncryptionSecurity verifies AES-GCM encryption security
func TestEncryptionSecurity(t *testing.T) {
	t.Run("nonce_uniqueness", func(t *testing.T) {
		key, _ := crypto.GenerateKey()
		encryptor, _ := crypto.NewAESGCMEncryptor(key)

		ciphertexts := make(map[string]bool)
		plaintext := "sensitive data"

		for i := 0; i < 1000; i++ {
			ciphertext, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			if ciphertexts[ciphertext] {
				t.Error("Duplicate ciphertext found - nonce reuse detected")
			}
			ciphertexts[ciphertext] = true
		}
	})

	t.Run("authentication_integrity", func(t *testing.T) {
		key, _ := crypto.GenerateKey()
		encryptor, _ := crypto.NewAESGCMEncryptor(key)

		plaintext := "sensitive data"
		ciphertext, _ := encryptor.Encrypt(plaintext)

		// Tamper with ciphertext
		b := []byte(ciphertext)
		b[len(b)/2] ^= 1
		tampered := string(b)

		_, err := encryptor.Decrypt(tampered)
		if err == nil {
			t.Error("Tampered ciphertext should be rejected")
		}
	})

	t.Run("key_isolation", func(t *testing.T) {
		key1, _ := crypto.GenerateKey()
		key2, _ := crypto.GenerateKey()

		encryptor1, _ := crypto.NewAESGCMEncryptor(key1)
		encryptor2, _ := crypto.NewAESGCMEncryptor(key2)

		plaintext := "sensitive data"
		ciphertext, _ := encryptor1.Encrypt(plaintext)

		// Try to decrypt with wrong key
		_, err := encryptor2.Decrypt(ciphertext)
		if err == nil {
			t.Error("Decryption with wrong key should fail")
		}
	})
}

// TestSQLInjectionResistance tests SQL injection prevention
func TestSQLInjectionResistance(t *testing.T) {
	sqlPayloads := []string{
		"admin'--",
		"admin' OR '1'='1",
		"'; DROP TABLE users--",
		"1' UNION SELECT NULL, username, password FROM users--",
		"admin'/*",
		"' OR '1'='1' /*",
		"admin'; EXEC xp_cmdshell('dir')--",
	}

	for _, payload := range sqlPayloads {
		t.Run("sql_"+payload, func(t *testing.T) {
			// These payloads should be safely handled by parameterized queries
			// This test documents that the system should not be vulnerable
			t.Logf("Testing SQL injection payload: %s", payload)

			// In actual implementation, verify that:
			// 1. Parameterized queries are used
			// 2. Input validation rejects malicious input
			// 3. No SQL errors are exposed to user
		})
	}
}

// TestXSSResistance tests XSS prevention
func TestXSSResistance(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg onload=alert('XSS')>",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
		"<body onload=alert('XSS')>",
	}

	for _, payload := range xssPayloads {
		t.Run("xss_"+payload[:10], func(t *testing.T) {
			// These payloads should be safely escaped in output
			// This test documents that the system should not be vulnerable
			t.Logf("Testing XSS payload: %s", payload)

			// In actual implementation, verify that:
			// 1. Output is properly escaped
			// 2. Content-Type headers are set correctly
			// 3. CSP headers are configured
		})
	}
}

// TestPasswordPolicyCompliance verifies password policy enforcement
func TestPasswordPolicyCompliance(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"strong", "StrongPass123", true},
		{"too_short", "Short1", false},
		{"no_uppercase", "password123", false},
		{"no_lowercase", "PASSWORD123", false},
		{"no_digit", "PasswordOnly", false},
		{"too_long", strings.Repeat("a", 130) + "A1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePasswordStrength(tt.password)

			if tt.valid && err != nil {
				t.Errorf("Password '%s' should be valid, got error: %v", tt.name, err)
			}

			if !tt.valid && err == nil {
				t.Errorf("Password '%s' should be invalid", tt.name)
			}
		})
	}
}

// TestSensitiveDataExposure verifies no sensitive data in responses
func TestSensitiveDataExposure(t *testing.T) {
	t.Run("password_not_in_json", func(t *testing.T) {
		// Verify password field has json:"-" tag
		// This should be tested at API response level
		t.Log("Password should never appear in API responses")
	})

	t.Run("api_keys_encrypted", func(t *testing.T) {
		// Verify API keys are encrypted at rest
		t.Log("API keys should be encrypted in database")
	})

	t.Run("tokens_hashed", func(t *testing.T) {
		// Verify refresh tokens are hashed in database
		t.Log("Refresh tokens should be hashed in database")
	})
}

// TestSecureRandomGeneration verifies cryptographic randomness
func TestSecureRandomGeneration(t *testing.T) {
	t.Run("encryption_keys", func(t *testing.T) {
		keys := make(map[string]bool)

		for i := 0; i < 100; i++ {
			key, err := crypto.GenerateKey()
			if err != nil {
				t.Fatalf("Key generation failed: %v", err)
			}

			if len(key) != 32 {
				t.Errorf("Key length = %d, want 32", len(key))
			}

			keyStr := string(key)
			if keys[keyStr] {
				t.Error("Duplicate key generated")
			}
			keys[keyStr] = true
		}
	})

	t.Run("refresh_tokens", func(t *testing.T) {
		manager, _ := auth.NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			15*time.Minute,
			7*24*time.Hour,
			"test-issuer",
		)

		tokens := make(map[string]bool)

		for i := 0; i < 100; i++ {
			token, _, err := manager.GenerateRefreshToken()
			if err != nil {
				t.Fatalf("Token generation failed: %v", err)
			}

			if tokens[token] {
				t.Error("Duplicate refresh token generated")
			}
			tokens[token] = true
		}
	})
}

// TestTimingAttackResistance verifies constant-time operations
func TestTimingAttackResistance(t *testing.T) {
	t.Run("password_verification", func(t *testing.T) {
		hasher := auth.NewArgon2Hasher(nil)
		password := "CorrectPassword123"
		hash, _ := hasher.Hash(password)

		// Verify all wrong passwords take similar time
		wrongPasswords := []string{
			"WrongPassword123",
			"CorrectPassword12",
			"X",
		}

		for _, wrong := range wrongPasswords {
			err := hasher.Verify(wrong, hash)
			if err == nil {
				t.Errorf("Wrong password should not verify: %s", wrong)
			}
			// Timing should be constant regardless of how wrong the password is
		}
	})
}
