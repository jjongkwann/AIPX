package testutil

import (
	"context"
	"sync"
	"time"

	"user-service/internal/models"
	"user-service/internal/repository"
)

// MockTokenRepository is a mock implementation of RefreshTokenRepository for testing
type MockTokenRepository struct {
	mu sync.RWMutex
	tokens map[string]*models.RefreshToken // token ID -> token
	hashIndex map[string]string // token hash -> token ID
}

// NewMockTokenRepository creates a new mock token repository
func NewMockTokenRepository() *MockTokenRepository {
	return &MockTokenRepository{
		tokens: make(map[string]*models.RefreshToken),
		hashIndex: make(map[string]string),
	}
}

// CreateRefreshToken creates a new refresh token
func (m *MockTokenRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID if not set
	if token.ID == "" {
		token.ID = "mock-token-" + token.TokenHash
	}

	m.tokens[token.ID] = token
	m.hashIndex[token.TokenHash] = token.ID
	return nil
}

// GetRefreshTokenByHash retrieves a token by its hash
func (m *MockTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenID, exists := m.hashIndex[tokenHash]
	if !exists {
		return nil, repository.ErrRefreshTokenNotFound
	}

	token, exists := m.tokens[tokenID]
	if !exists {
		return nil, repository.ErrRefreshTokenNotFound
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, repository.ErrRefreshTokenExpired
	}

	// Check if token is revoked
	if token.IsRevoked() {
		return nil, repository.ErrRefreshTokenRevoked
	}

	return token, nil
}

// GetRefreshTokensByUserID retrieves all tokens for a user
func (m *MockTokenRepository) GetRefreshTokensByUserID(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var userTokens []*models.RefreshToken
	for _, token := range m.tokens {
		if token.UserID == userID {
			userTokens = append(userTokens, token)
		}
	}

	return userTokens, nil
}

// GetRefreshToken retrieves a token by ID
func (m *MockTokenRepository) GetRefreshToken(ctx context.Context, id string) (*models.RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	token, exists := m.tokens[id]
	if !exists {
		return nil, repository.ErrRefreshTokenNotFound
	}

	return token, nil
}

// GetUserTokens retrieves all active tokens for a user
func (m *MockTokenRepository) GetUserTokens(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	return m.GetRefreshTokensByUserID(ctx, userID)
}

// RevokeToken revokes a token
func (m *MockTokenRepository) RevokeToken(ctx context.Context, tokenID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, exists := m.tokens[tokenID]
	if !exists {
		return repository.ErrRefreshTokenNotFound
	}

	now := time.Now()
	token.RevokedAt = &now
	m.tokens[tokenID] = token
	return nil
}

// RevokeAllUserTokens revokes all tokens for a user
func (m *MockTokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return m.RevokeUserTokens(ctx, userID)
}

// RevokeUserTokens revokes all tokens for a user
func (m *MockTokenRepository) RevokeUserTokens(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, token := range m.tokens {
		if token.UserID == userID && !token.IsRevoked() {
			token.RevokedAt = &now
			m.tokens[token.ID] = token
		}
	}

	return nil
}

// DeleteExpiredTokens deletes expired tokens
func (m *MockTokenRepository) DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for id, token := range m.tokens {
		if token.ExpiresAt.Before(cutoff) {
			delete(m.tokens, id)
			delete(m.hashIndex, token.TokenHash)
		}
	}

	return nil
}

// AddTestToken adds a test token to the repository (test helper)
func (m *MockTokenRepository) AddTestToken(token *models.RefreshToken) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens[token.ID] = token
	m.hashIndex[token.TokenHash] = token.ID
}

// Reset clears all data (test helper)
func (m *MockTokenRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = make(map[string]*models.RefreshToken)
	m.hashIndex = make(map[string]string)
}
