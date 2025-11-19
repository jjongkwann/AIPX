package models

import (
	"time"
)

// User represents a user account in the system
type User struct {
	ID            string    `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"` // Never expose in JSON
	Name          string    `json:"name" db:"name"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// UserCreateRequest represents a request to create a new user
type UserCreateRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Name     string `json:"name" binding:"required,min=2,max=100"`
}

// UserUpdateRequest represents a request to update user information
type UserUpdateRequest struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Email *string `json:"email,omitempty" binding:"omitempty,email"`
}

// UserResponse represents the public user information returned in API responses
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	IsActive      bool      `json:"is_active"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToResponse converts a User model to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		IsActive:      u.IsActive,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// BrokerType represents supported broker types
type BrokerType string

const (
	BrokerKIS   BrokerType = "KIS"
	BrokerEBest BrokerType = "eBest"
	BrokerNH    BrokerType = "NH"
	BrokerKB    BrokerType = "KB"
)

// IsValid checks if the broker type is valid
func (b BrokerType) IsValid() bool {
	switch b {
	case BrokerKIS, BrokerEBest, BrokerNH, BrokerKB:
		return true
	}
	return false
}

// APIKey represents encrypted broker API credentials
type APIKey struct {
	ID              string     `json:"id" db:"id"`
	UserID          string     `json:"user_id" db:"user_id"`
	Broker          BrokerType `json:"broker" db:"broker"`
	KeyEncrypted    string     `json:"-" db:"key_encrypted"`    // Never expose in JSON
	SecretEncrypted string     `json:"-" db:"secret_encrypted"` // Never expose in JSON
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// APIKeyCreateRequest represents a request to create new API credentials
type APIKeyCreateRequest struct {
	Broker    BrokerType `json:"broker" binding:"required"`
	APIKey    string     `json:"api_key" binding:"required,min=10"`
	APISecret string     `json:"api_secret" binding:"required,min=10"`
}

// APIKeyResponse represents the public API key information (without secrets)
type APIKeyResponse struct {
	ID         string     `json:"id"`
	Broker     BrokerType `json:"broker"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ToResponse converts an APIKey model to APIKeyResponse
func (a *APIKey) ToResponse() *APIKeyResponse {
	return &APIKeyResponse{
		ID:         a.ID,
		Broker:     a.Broker,
		IsActive:   a.IsActive,
		CreatedAt:  a.CreatedAt,
		LastUsedAt: a.LastUsedAt,
	}
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never expose in JSON
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	IPAddress string     `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string     `json:"user_agent,omitempty" db:"user_agent"`
}

// IsExpired checks if the refresh token has expired
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsRevoked checks if the refresh token has been revoked
func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}

// IsValid checks if the refresh token is valid (not expired and not revoked)
func (r *RefreshToken) IsValid() bool {
	return !r.IsExpired() && !r.IsRevoked()
}

// AuditEventType represents the type of security audit event
type AuditEventType string

const (
	EventLogin           AuditEventType = "login"
	EventLogout          AuditEventType = "logout"
	EventRegister        AuditEventType = "register"
	EventPasswordChange  AuditEventType = "password_change"
	EventPasswordReset   AuditEventType = "password_reset"
	EventAPIKeyCreated   AuditEventType = "api_key_created"
	EventAPIKeyDeleted   AuditEventType = "api_key_deleted"
	EventEmailVerified   AuditEventType = "email_verified"
	EventAccountLocked   AuditEventType = "account_locked"
	EventAccountUnlocked AuditEventType = "account_unlocked"
	EventFailedLogin     AuditEventType = "failed_login"
)

// AuditLog represents a security audit log entry
type AuditLog struct {
	ID        string         `json:"id" db:"id"`
	UserID    *string        `json:"user_id,omitempty" db:"user_id"`
	EventType AuditEventType `json:"event_type" db:"event_type"`
	EventData map[string]any `json:"event_data,omitempty" db:"event_data"`
	IPAddress string         `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string         `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"` // seconds
	User         *UserResponse `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents a token refresh response
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // seconds
}

// PasswordChangeRequest represents a password change request
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

// EmailVerificationRequest represents an email verification request
type EmailVerificationRequest struct {
	Token string `json:"token" binding:"required"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirmRequest represents a password reset confirmation
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}
