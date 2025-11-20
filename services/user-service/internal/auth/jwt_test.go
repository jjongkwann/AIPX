package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name          string
		accessSecret  string
		refreshSecret string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "valid secrets",
			accessSecret:  "test-access-secret-key-minimum-32-characters",
			refreshSecret: "test-refresh-secret-key-minimum-32-characters",
			wantErr:       false,
		},
		{
			name:          "empty access secret",
			accessSecret:  "",
			refreshSecret: "test-refresh-secret-key-minimum-32-characters",
			wantErr:       true,
			errContains:   "access secret cannot be empty",
		},
		{
			name:          "empty refresh secret",
			accessSecret:  "test-access-secret-key-minimum-32-characters",
			refreshSecret: "",
			wantErr:       true,
			errContains:   "refresh secret cannot be empty",
		},
		{
			name:          "access secret too short",
			accessSecret:  "short",
			refreshSecret: "test-refresh-secret-key-minimum-32-characters",
			wantErr:       true,
			errContains:   "access secret must be at least 32 characters",
		},
		{
			name:          "refresh secret too short",
			accessSecret:  "test-access-secret-key-minimum-32-characters",
			refreshSecret: "short",
			wantErr:       true,
			errContains:   "refresh secret must be at least 32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(
				tt.accessSecret,
				tt.refreshSecret,
				15*time.Minute,
				7*24*time.Hour,
				"test-issuer",
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewJWTManager() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewJWTManager() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewJWTManager() unexpected error = %v", err)
				return
			}

			if manager == nil {
				t.Error("NewJWTManager() returned nil manager")
			}
		})
	}
}

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := newTestJWTManager(t)
	userID := "test-user-123"
	email := "test@example.com"

	token, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Verify token can be parsed
	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}

	if claims.Email != email {
		t.Errorf("Email = %v, want %v", claims.Email, email)
	}

	if claims.Type != string(AccessToken) {
		t.Errorf("Type = %v, want %v", claims.Type, AccessToken)
	}

	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer = %v, want test-issuer", claims.Issuer)
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := newTestJWTManager(t)

	token1, hash1, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token1 == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	if hash1 == "" {
		t.Error("GenerateRefreshToken() returned empty hash")
	}

	// Generate another token and verify uniqueness
	token2, hash2, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() second call error = %v", err)
	}

	if token1 == token2 {
		t.Error("GenerateRefreshToken() generated duplicate tokens")
	}

	if hash1 == hash2 {
		t.Error("GenerateRefreshToken() generated duplicate hashes")
	}

	// Verify hash matches token
	computedHash := manager.HashRefreshToken(token1)
	if computedHash != hash1 {
		t.Error("HashRefreshToken() doesn't match generated hash")
	}
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	manager := newTestJWTManager(t)
	userID := "test-user-123"
	email := "test@example.com"

	t.Run("valid token", func(t *testing.T) {
		token, err := manager.GenerateAccessToken(userID, email)
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := manager.ValidateAccessToken(token)
		if err != nil {
			t.Errorf("ValidateAccessToken() error = %v", err)
		}

		if claims.UserID != userID {
			t.Errorf("UserID = %v, want %v", claims.UserID, userID)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create manager with very short TTL
		shortManager, _ := NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			1*time.Nanosecond, // Expire immediately
			7*24*time.Hour,
			"test-issuer",
		)

		token, err := shortManager.GenerateAccessToken(userID, email)
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		_, err = shortManager.ValidateAccessToken(token)
		if err == nil {
			t.Error("ValidateAccessToken() should reject expired token")
		}

		if err != ErrTokenExpired {
			t.Errorf("ValidateAccessToken() error = %v, want %v", err, ErrTokenExpired)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		token, err := manager.GenerateAccessToken(userID, email)
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Tamper with token by replacing last character
		tamperedToken := token[:len(token)-1] + "X"

		_, err = manager.ValidateAccessToken(tamperedToken)
		if err == nil {
			t.Error("ValidateAccessToken() should reject tampered token")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		malformedTokens := []string{
			"",
			"not.a.token",
			"invalid",
			"header.payload", // Missing signature
		}

		for _, token := range malformedTokens {
			_, err := manager.ValidateAccessToken(token)
			if err == nil {
				t.Errorf("ValidateAccessToken(%q) should reject malformed token", token)
			}
		}
	})

	t.Run("token from different issuer", func(t *testing.T) {
		otherManager, _ := NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			15*time.Minute,
			7*24*time.Hour,
			"other-issuer",
		)

		token, err := otherManager.GenerateAccessToken(userID, email)
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Try to validate with different manager (same secret but different issuer check)
		manager2, _ := NewJWTManager(
			"test-access-secret-key-minimum-32-characters",
			"test-refresh-secret-key-minimum-32-characters",
			15*time.Minute,
			7*24*time.Hour,
			"test-issuer", // Different issuer
		)

		_, err = manager2.ValidateAccessToken(token)
		if err == nil {
			t.Error("ValidateAccessToken() should reject token from different issuer")
		}
	})

	t.Run("token with wrong signing method", func(t *testing.T) {
		// Create token with RS256 instead of HS256
		claims := &Claims{
			UserID: userID,
			Email:  email,
			Type:   string(AccessToken),
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "test-issuer",
				Subject:   userID,
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

		_, err := manager.ValidateAccessToken(tokenString)
		if err == nil {
			t.Error("ValidateAccessToken() should reject token with wrong signing method")
		}
	})
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	manager, err := NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		accessTTL,
		refreshTTL,
		"test-issuer",
	)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	t.Run("access token TTL", func(t *testing.T) {
		if manager.GetAccessTTL() != accessTTL {
			t.Errorf("GetAccessTTL() = %v, want %v", manager.GetAccessTTL(), accessTTL)
		}

		if manager.GetAccessTTLSeconds() != int64(accessTTL.Seconds()) {
			t.Errorf("GetAccessTTLSeconds() = %v, want %v", manager.GetAccessTTLSeconds(), int64(accessTTL.Seconds()))
		}
	})

	t.Run("refresh token TTL", func(t *testing.T) {
		if manager.GetRefreshTTL() != refreshTTL {
			t.Errorf("GetRefreshTTL() = %v, want %v", manager.GetRefreshTTL(), refreshTTL)
		}

		if manager.GetRefreshTTLSeconds() != int64(refreshTTL.Seconds()) {
			t.Errorf("GetRefreshTTLSeconds() = %v, want %v", manager.GetRefreshTTLSeconds(), int64(refreshTTL.Seconds()))
		}
	})

	t.Run("access token expires after specified duration", func(t *testing.T) {
		token, err := manager.GenerateAccessToken("user-123", "test@example.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := manager.ValidateAccessToken(token)
		if err != nil {
			t.Fatalf("ValidateAccessToken() error = %v", err)
		}

		expectedExpiry := time.Now().Add(accessTTL)
		actualExpiry := claims.ExpiresAt.Time

		// Allow 1 second tolerance
		diff := actualExpiry.Sub(expectedExpiry)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Token expiry = %v, want within 1s of %v", actualExpiry, expectedExpiry)
		}
	})
}

func TestJWTManager_ClaimsExtraction(t *testing.T) {
	manager := newTestJWTManager(t)
	userID := "user-123"
	email := "test@example.com"

	token, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"UserID", claims.UserID, userID},
		{"Email", claims.Email, email},
		{"Type", claims.Type, string(AccessToken)},
		{"Issuer", claims.Issuer, "test-issuer"},
		{"Subject", claims.Subject, userID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Verify timestamps
	if claims.IssuedAt == nil {
		t.Error("IssuedAt should not be nil")
	}

	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}

	if claims.NotBefore == nil {
		t.Error("NotBefore should not be nil")
	}

	// Verify token is not valid before NotBefore time
	if time.Now().Before(claims.NotBefore.Time) {
		t.Error("Token should be valid after NotBefore time")
	}
}

func TestJWTManager_HashRefreshToken(t *testing.T) {
	manager := newTestJWTManager(t)

	token1 := "test-token-1"
	token2 := "test-token-2"

	hash1 := manager.HashRefreshToken(token1)
	hash2 := manager.HashRefreshToken(token2)

	if hash1 == "" {
		t.Error("HashRefreshToken() returned empty hash")
	}

	if hash2 == "" {
		t.Error("HashRefreshToken() returned empty hash")
	}

	if hash1 == hash2 {
		t.Error("HashRefreshToken() produced same hash for different tokens")
	}

	// Verify hashing is deterministic
	hash1Again := manager.HashRefreshToken(token1)
	if hash1 != hash1Again {
		t.Error("HashRefreshToken() should be deterministic")
	}
}

func TestJWTManager_ReplayAttackPrevention(t *testing.T) {
	manager := newTestJWTManager(t)
	userID := "test-user-123"
	email := "test@example.com"

	// Generate first token
	token1, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(2 * time.Second)

	// Generate second token
	token2, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Tokens should be different (preventing replay attacks)
	if token1 == token2 {
		t.Error("GenerateAccessToken() generated identical tokens")
	}

	// Both should be valid
	_, err = manager.ValidateAccessToken(token1)
	if err != nil {
		t.Errorf("ValidateAccessToken(token1) error = %v", err)
	}

	_, err = manager.ValidateAccessToken(token2)
	if err != nil {
		t.Errorf("ValidateAccessToken(token2) error = %v", err)
	}
}

// Test helper to create a JWT manager for testing
func newTestJWTManager(t *testing.T) *JWTManager {
	t.Helper()

	manager, err := NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)
	if err != nil {
		t.Fatalf("Failed to create test JWT manager: %v", err)
	}

	return manager
}
