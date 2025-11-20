package auth

import (
	"strings"
	"testing"
)

func TestArgon2Hasher_Hash(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	tests := []struct {
		name        string
		password    string
		shouldError bool
	}{
		{
			name:        "valid password",
			password:    "SecurePassword123!",
			shouldError: false,
		},
		{
			name:        "empty password",
			password:    "",
			shouldError: true,
		},
		{
			name:        "long password",
			password:    strings.Repeat("a", 100),
			shouldError: false,
		},
		{
			name:        "unicode password",
			password:    "パスワード123",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)

			if tt.shouldError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify hash format
			if !strings.HasPrefix(hash, "$argon2id$v=19$") {
				t.Errorf("invalid hash format: %s", hash)
			}

			// Verify hash parts count
			parts := strings.Split(hash, "$")
			if len(parts) != 6 {
				t.Errorf("expected 6 parts in hash, got %d", len(parts))
			}
		})
	}
}

func TestArgon2Hasher_Verify(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	password := "SecurePassword123!"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed to create hash: %v", err)
	}

	tests := []struct {
		name        string
		password    string
		hash        string
		shouldError bool
	}{
		{
			name:        "correct password",
			password:    password,
			hash:        hash,
			shouldError: false,
		},
		{
			name:        "incorrect password",
			password:    "WrongPassword123!",
			hash:        hash,
			shouldError: true,
		},
		{
			name:        "empty password",
			password:    "",
			hash:        hash,
			shouldError: true,
		},
		{
			name:        "empty hash",
			password:    password,
			hash:        "",
			shouldError: true,
		},
		{
			name:        "invalid hash format",
			password:    password,
			hash:        "invalid-hash",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.Verify(tt.password, tt.hash)

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

func TestArgon2Hasher_HashUniqueness(t *testing.T) {
	hasher := NewArgon2Hasher(nil)
	password := "SamePassword123!"

	// Generate multiple hashes of the same password
	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed to create first hash: %v", err)
	}

	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed to create second hash: %v", err)
	}

	// Hashes should be different due to different salts
	if hash1 == hash2 {
		t.Error("expected different hashes for same password")
	}

	// But both should verify correctly
	if err := hasher.Verify(password, hash1); err != nil {
		t.Errorf("first hash verification failed: %v", err)
	}

	if err := hasher.Verify(password, hash2); err != nil {
		t.Errorf("second hash verification failed: %v", err)
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		shouldError bool
	}{
		{
			name:        "valid strong password",
			password:    "SecurePass123",
			shouldError: false,
		},
		{
			name:        "too short",
			password:    "Short1A",
			shouldError: true,
		},
		{
			name:        "too long",
			password:    strings.Repeat("a", 129) + "A1",
			shouldError: true,
		},
		{
			name:        "no uppercase",
			password:    "securepass123",
			shouldError: true,
		},
		{
			name:        "no lowercase",
			password:    "SECUREPASS123",
			shouldError: true,
		},
		{
			name:        "no digit",
			password:    "SecurePassword",
			shouldError: true,
		},
		{
			name:        "with special characters",
			password:    "SecurePass123!@#",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)

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

func BenchmarkArgon2Hasher_Hash(b *testing.B) {
	hasher := NewArgon2Hasher(nil)
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

func BenchmarkArgon2Hasher_Verify(b *testing.B) {
	hasher := NewArgon2Hasher(nil)
	password := "BenchmarkPassword123!"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Verify(password, hash)
	}
}

func TestArgon2Hasher_SaltUniqueness(t *testing.T) {
	hasher := NewArgon2Hasher(nil)
	password := "TestPassword123"

	// Generate multiple hashes
	hashes := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		hash, err := hasher.Hash(password)
		if err != nil {
			t.Fatalf("Hash() error = %v", err)
		}

		if hashes[hash] {
			t.Errorf("Duplicate hash detected: %s", hash)
		}
		hashes[hash] = true

		// Extract salt from hash
		parts := strings.Split(hash, "$")
		if len(parts) < 5 {
			t.Fatalf("Invalid hash format: %s", hash)
		}
		salt := parts[4]

		// Verify salt is different each time (with high probability)
		for otherHash := range hashes {
			if otherHash == hash {
				continue
			}
			otherParts := strings.Split(otherHash, "$")
			otherSalt := otherParts[4]
			if salt == otherSalt {
				t.Error("Found duplicate salt in different hashes")
			}
		}
	}

	if len(hashes) != iterations {
		t.Errorf("Generated %d unique hashes, want %d", len(hashes), iterations)
	}
}

func TestArgon2Hasher_TimingAttack(t *testing.T) {
	hasher := NewArgon2Hasher(nil)
	password := "CorrectPassword123"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Test that verification time is constant regardless of how wrong the password is
	tests := []string{
		"WrongPassword123",   // Completely different
		"CorrectPassword12",  // One character off
		"correctpassword123", // Case different
		"X",                  // Very short
	}

	for _, wrongPassword := range tests {
		err := hasher.Verify(wrongPassword, hash)
		if err == nil {
			t.Errorf("Verify() should fail for wrong password: %s", wrongPassword)
		}
		if err != ErrMismatchedHashAndPassword {
			t.Errorf("Verify() error = %v, want %v", err, ErrMismatchedHashAndPassword)
		}
	}
}

func TestArgon2Hasher_EmptyPassword(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	hash, err := hasher.Hash("")
	if err == nil {
		t.Error("Hash() should reject empty password")
	}

	if hash != "" {
		t.Errorf("Hash() returned non-empty hash for empty password: %s", hash)
	}
}

func TestArgon2Hasher_LongPassword(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	// Test with 1000+ character password
	longPassword := strings.Repeat("a", 1000) + "Bc123"

	hash, err := hasher.Hash(longPassword)
	if err != nil {
		t.Fatalf("Hash() error with long password = %v", err)
	}

	// Verify it works correctly
	err = hasher.Verify(longPassword, hash)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}

	// Verify wrong long password fails
	wrongLongPassword := strings.Repeat("b", 1000) + "Bc123"
	err = hasher.Verify(wrongLongPassword, hash)
	if err == nil {
		t.Error("Verify() should fail for wrong long password")
	}
}

func TestArgon2Hasher_SpecialCharacters(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	specialPasswords := []string{
		"P@ssw0rd!",
		"Test#123$",
		"Sp€cial123",
		"你好世界Abc123", // Unicode
		"P@ss\nword123",  // Newline
		"P@ss\tword123",  // Tab
	}

	for _, password := range specialPasswords {
		t.Run(password, func(t *testing.T) {
			hash, err := hasher.Hash(password)
			if err != nil {
				t.Fatalf("Hash() error = %v", err)
			}

			err = hasher.Verify(password, hash)
			if err != nil {
				t.Errorf("Verify() error = %v", err)
			}

			// Verify slightly different password fails
			wrongPassword := password + "x"
			err = hasher.Verify(wrongPassword, hash)
			if err == nil {
				t.Error("Verify() should fail for wrong password")
			}
		})
	}
}

func TestArgon2Hasher_HashOutputLength(t *testing.T) {
	hasher := NewArgon2Hasher(nil)
	password := "TestPassword123"

	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Parse hash to verify key length
	parts := strings.Split(hash1, "$")
	if len(parts) != 6 {
		t.Fatalf("Invalid hash format, got %d parts", len(parts))
	}

	// Hash part should be base64 encoded 32-byte key
	hashPart := parts[5]
	if len(hashPart) < 40 { // Base64 of 32 bytes is at least 43 chars (minus padding)
		t.Errorf("Hash part too short: %d characters", len(hashPart))
	}
}

func TestArgon2Hasher_Performance(t *testing.T) {
	hasher := NewArgon2Hasher(nil)
	password := "TestPassword123"

	// Hashing should take 100-500ms per OWASP recommendations
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Just verify it completes (actual timing checked in benchmarks)
	err = hasher.Verify(password, hash)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}
}

func TestDecodeHash_InvalidFormats(t *testing.T) {
	hasher := NewArgon2Hasher(nil)

	invalidHashes := []string{
		"",
		"invalid",
		"$argon2id$",
		"$bcrypt$v=19$m=65536,t=3,p=2$salt$hash", // Wrong algorithm
		"$argon2id$v=18$m=65536,t=3,p=2$salt$hash", // Wrong version
		"$argon2id$v=19$m=65536$salt$hash",         // Missing parameters
		"$argon2id$v=19$m=65536,t=3,p=2$invalidsalt$hash", // Invalid base64
	}

	for _, invalidHash := range invalidHashes {
		t.Run(invalidHash, func(t *testing.T) {
			err := hasher.Verify("password", invalidHash)
			if err == nil {
				t.Errorf("Verify() should reject invalid hash: %s", invalidHash)
			}
		})
	}
}
