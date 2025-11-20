package testutil

import (
	"time"

	"user-service/internal/models"
)

// NewTestUser creates a test user with default values
func NewTestUser() *models.User {
	return &models.User{
		ID:            "test-user-id-123",
		Email:         "test@example.com",
		PasswordHash:  "$argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$somehash",
		Name:          "Test User",
		IsActive:      true,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// NewTestUserWithEmail creates a test user with specified email
func NewTestUserWithEmail(email string) *models.User {
	user := NewTestUser()
	user.Email = email
	return user
}

// NewTestSignupRequest creates a valid signup request
func NewTestSignupRequest() *models.SignupRequest {
	return &models.SignupRequest{
		Email:    "newuser@example.com",
		Password: "StrongPass123",
		Name:     "New User",
	}
}

// NewTestLoginRequest creates a valid login request
func NewTestLoginRequest() *models.LoginRequest {
	return &models.LoginRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
	}
}

// NewTestRefreshToken creates a test refresh token
func NewTestRefreshToken(userID string) *models.RefreshToken {
	return &models.RefreshToken{
		ID:        "test-token-id-123",
		UserID:    userID,
		TokenHash: "test-token-hash",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		RevokedAt: nil,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}
}

// NewTestAPIKey creates a test API key
func NewTestAPIKey(userID string) *models.APIKey {
	return &models.APIKey{
		ID:              "test-apikey-id-123",
		UserID:          userID,
		Broker:          models.BrokerKIS,
		KeyEncrypted:    "encrypted-key",
		SecretEncrypted: "encrypted-secret",
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastUsedAt:      nil,
	}
}

// ValidPasswords returns a list of valid test passwords
func ValidPasswords() []string {
	return []string{
		"StrongPass123",
		"SecureP@ssw0rd",
		"MyP@ssword1",
		"Test1234Pass",
		"Abcd1234Efgh",
	}
}

// WeakPasswords returns a list of weak passwords for testing validation
func WeakPasswords() []string {
	return []string{
		"short",           // Too short
		"alllowercase",    // No uppercase
		"ALLUPPERCASE",    // No lowercase
		"NoDigitsHere",    // No digits
		"12345678",        // Only digits
		"",                // Empty
	}
}

// SQLInjectionPayloads returns common SQL injection attack strings
func SQLInjectionPayloads() []string {
	return []string{
		"admin'--",
		"admin' OR '1'='1",
		"admin'; DROP TABLE users--",
		"' OR 1=1--",
		"1' UNION SELECT NULL, username, password FROM users--",
		"admin'/*",
		"' OR '1'='1' /*",
	}
}

// XSSPayloads returns common XSS attack strings
func XSSPayloads() []string {
	return []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg onload=alert('XSS')>",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
		"<body onload=alert('XSS')>",
	}
}

// LongPassword returns a password exceeding maximum length
func LongPassword() string {
	password := ""
	for i := 0; i < 1100; i++ {
		password += "a"
	}
	return password
}

// UnicodePassword returns a password with Unicode characters
func UnicodePassword() string {
	return "P@ssw0rd你好世界"
}
