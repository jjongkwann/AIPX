package testutil

import (
	"context"
	"sync"

	"user-service/internal/models"
	"user-service/internal/repository"
)

// MockUserRepository is a mock implementation of UserRepository for testing
type MockUserRepository struct {
	mu sync.RWMutex
	users map[string]*models.User
	emailIndex map[string]string // email -> user ID
}

// NewMockUserRepository creates a new mock user repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*models.User),
		emailIndex: make(map[string]string),
	}
}

// CreateUser creates a new user
func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if email already exists
	if _, exists := m.emailIndex[user.Email]; exists {
		return repository.ErrUserAlreadyExists
	}

	// Generate ID if not set
	if user.ID == "" {
		user.ID = "mock-user-" + user.Email
	}

	m.users[user.ID] = user
	m.emailIndex[user.Email] = user.ID
	return nil
}

// GetUserByID retrieves a user by ID
func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists || !user.IsActive {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userID, exists := m.emailIndex[email]
	if !exists {
		return nil, repository.ErrUserNotFound
	}

	user, exists := m.users[userID]
	if !exists {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

// UpdateUser updates a user
func (m *MockUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingUser, exists := m.users[user.ID]
	if !exists || !existingUser.IsActive {
		return repository.ErrUserNotFound
	}

	// Update email index if email changed
	if existingUser.Email != user.Email {
		delete(m.emailIndex, existingUser.Email)
		m.emailIndex[user.Email] = user.ID
	}

	m.users[user.ID] = user
	return nil
}

// DeleteUser soft deletes a user
func (m *MockUserRepository) DeleteUser(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists || !user.IsActive {
		return repository.ErrUserNotFound
	}

	user.IsActive = false
	m.users[id] = user
	return nil
}

// UpdateEmailVerified updates email verification status
func (m *MockUserRepository) UpdateEmailVerified(ctx context.Context, id string, verified bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists || !user.IsActive {
		return repository.ErrUserNotFound
	}

	user.EmailVerified = verified
	m.users[id] = user
	return nil
}

// UpdatePassword updates user password
func (m *MockUserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists || !user.IsActive {
		return repository.ErrUserNotFound
	}

	user.PasswordHash = passwordHash
	m.users[id] = user
	return nil
}

// AddTestUser adds a test user to the repository (test helper)
func (m *MockUserRepository) AddTestUser(user *models.User) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[user.ID] = user
	m.emailIndex[user.Email] = user.ID
}

// Reset clears all data (test helper)
func (m *MockUserRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users = make(map[string]*models.User)
	m.emailIndex = make(map[string]string)
}
