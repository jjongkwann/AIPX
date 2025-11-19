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
