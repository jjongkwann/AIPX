package models

import "time"

// APIResponse represents a standard API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	Details   []string  `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// AuthResponse represents authentication response with tokens
type AuthResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"` // seconds
	User         *UserResponse `json:"user"`
}

// TokenRefreshResponse represents a token refresh response
type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // seconds
}

// NewSuccessResponse creates a new success API response
func NewSuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(errorMsg string, details ...string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     errorMsg,
		Details:   details,
		Timestamp: time.Now().UTC(),
	}
}
