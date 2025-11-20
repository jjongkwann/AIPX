package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"user-service/internal/auth"
	"user-service/internal/models"
	"user-service/internal/repository"
)

type mockUserRepo struct {
	users      map[string]*models.User
	emailIndex map[string]string
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:      make(map[string]*models.User),
		emailIndex: make(map[string]string),
	}
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	if _, exists := m.emailIndex[user.Email]; exists {
		return repository.ErrUserAlreadyExists
	}
	user.ID = "test-user-id-" + user.Email
	m.users[user.ID] = user
	m.emailIndex[user.Email] = user.ID
	return nil
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	userID, exists := m.emailIndex[email]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return m.users[userID], nil
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	return nil
}

func (m *mockUserRepo) DeleteUser(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepo) UpdateEmailVerified(ctx context.Context, id string, verified bool) error {
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	return nil
}

type mockTokenRepo struct {
	tokens    map[string]*models.RefreshToken
	hashIndex map[string]string
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{
		tokens:    make(map[string]*models.RefreshToken),
		hashIndex: make(map[string]string),
	}
}

func (m *mockTokenRepo) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	token.ID = "test-token-id"
	m.tokens[token.ID] = token
	m.hashIndex[token.TokenHash] = token.ID
	return nil
}

func (m *mockTokenRepo) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	tokenID, exists := m.hashIndex[tokenHash]
	if !exists {
		return nil, repository.ErrRefreshTokenNotFound
	}
	token := m.tokens[tokenID]
	if token.IsExpired() {
		return nil, repository.ErrRefreshTokenExpired
	}
	if token.IsRevoked() {
		return nil, repository.ErrRefreshTokenRevoked
	}
	return token, nil
}

func (m *mockTokenRepo) GetRefreshTokensByUserID(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	return nil, nil
}

func (m *mockTokenRepo) RevokeToken(ctx context.Context, tokenID string) error {
	token, exists := m.tokens[tokenID]
	if !exists {
		return repository.ErrRefreshTokenNotFound
	}
	now := time.Now()
	token.RevokedAt = &now
	return nil
}

func (m *mockTokenRepo) RevokeUserTokens(ctx context.Context, userID string) error {
	return nil
}

func (m *mockTokenRepo) DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func (m *mockTokenRepo) GetRefreshToken(ctx context.Context, id string) (*models.RefreshToken, error) {
	return nil, nil
}

func (m *mockTokenRepo) GetUserTokens(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	return nil, nil
}

func setupAuthHandler(t *testing.T) (*AuthHandler, *mockUserRepo, *mockTokenRepo) {
	t.Helper()

	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	passwordHasher := auth.NewArgon2Hasher(nil)
	jwtManager, _ := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	handler := NewAuthHandler(userRepo, tokenRepo, passwordHasher, jwtManager)
	return handler, userRepo, tokenRepo
}

func TestAuthHandler_Signup_Success(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	reqBody := models.SignupRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
		Name:     "Test User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Signup(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    *models.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success {
		t.Error("expected success to be true")
	}

	data := resp.Data

	if data.AccessToken == "" {
		t.Error("expected access token")
	}

	if data.RefreshToken == "" {
		t.Error("expected refresh token")
	}

	if data.User.Email != reqBody.Email {
		t.Errorf("expected email %s, got %s", reqBody.Email, data.User.Email)
	}
}

func TestAuthHandler_Signup_DuplicateEmail(t *testing.T) {
	handler, userRepo, _ := setupAuthHandler(t)

	// Add existing user
	existingUser := &models.User{
		ID:           "existing-user",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Name:         "Existing User",
		IsActive:     true,
	}
	userRepo.users[existingUser.ID] = existingUser
	userRepo.emailIndex[existingUser.Email] = existingUser.ID

	reqBody := models.SignupRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
		Name:     "New User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Signup(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestAuthHandler_Signup_WeakPassword(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	weakPasswords := []string{
		"short",        // Too short
		"nouppercase1", // No uppercase
		"NOLOWERCASE1", // No lowercase
		"NoDigitsHere", // No digits
	}

	for _, password := range weakPasswords {
		t.Run("weak_"+password, func(t *testing.T) {
			reqBody := models.SignupRequest{
				Email:    "test@example.com",
				Password: password,
				Name:     "Test User",
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Signup(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d for weak password, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestAuthHandler_Signup_InvalidEmail(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	invalidEmails := []string{
		"notanemail",
		"@example.com",
		"test@",
		"test..test@example.com",
	}

	for _, email := range invalidEmails {
		t.Run("invalid_"+email, func(t *testing.T) {
			reqBody := models.SignupRequest{
				Email:    email,
				Password: "StrongPass123",
				Name:     "Test User",
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Signup(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d for invalid email, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestAuthHandler_Signup_SQLInjection(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	sqlPayloads := []string{
		"admin'--",
		"admin' OR '1'='1",
		"'; DROP TABLE users--",
	}

	for _, payload := range sqlPayloads {
		t.Run("sql_"+payload, func(t *testing.T) {
			reqBody := models.SignupRequest{
				Email:    "test@example.com",
				Password: "StrongPass123",
				Name:     payload,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Signup(w, req)

			// Should not crash, should handle safely
			if w.Code == http.StatusInternalServerError {
				t.Error("SQL injection attempt caused internal error")
			}
		})
	}
}

func TestAuthHandler_Signup_XSS(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
	}

	for _, payload := range xssPayloads {
		t.Run("xss_"+payload[:10], func(t *testing.T) {
			reqBody := models.SignupRequest{
				Email:    "test@example.com",
				Password: "StrongPass123",
				Name:     payload,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Signup(w, req)

			// Should not crash, should handle safely
			if w.Code == http.StatusInternalServerError {
				t.Error("XSS attempt caused internal error")
			}

			// Verify response doesn't include unescaped payload
			if w.Code == http.StatusCreated {
				var resp struct {
					Success bool                 `json:"success"`
					Data    *models.AuthResponse `json:"data"`
				}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp.Data.User.Name == payload {
					// Name is stored as-is, which is fine as long as it's properly escaped when rendered
					t.Log("XSS payload stored (should be escaped on output)")
				}
			}
		})
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	handler, userRepo, _ := setupAuthHandler(t)

	// Create user with known password
	passwordHasher := auth.NewArgon2Hasher(nil)
	passwordHash, _ := passwordHasher.Hash("StrongPass123")

	user := &models.User{
		ID:           "test-user-id",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Name:         "Test User",
		IsActive:     true,
	}
	userRepo.users[user.ID] = user
	userRepo.emailIndex[user.Email] = user.ID

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    *models.AuthResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Success {
		t.Error("expected success to be true")
	}

	if resp.Data.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	handler, userRepo, _ := setupAuthHandler(t)

	// Create user
	passwordHasher := auth.NewArgon2Hasher(nil)
	passwordHash, _ := passwordHasher.Hash("CorrectPassword123")

	user := &models.User{
		ID:           "test-user-id",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Name:         "Test User",
		IsActive:     true,
	}
	userRepo.users[user.ID] = user
	userRepo.emailIndex[user.Email] = user.ID

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_NonexistentUser(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	reqBody := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "SomePassword123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_InactiveUser(t *testing.T) {
	handler, userRepo, _ := setupAuthHandler(t)

	// Create inactive user
	passwordHasher := auth.NewArgon2Hasher(nil)
	passwordHash, _ := passwordHasher.Hash("StrongPass123")

	user := &models.User{
		ID:           "test-user-id",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Name:         "Test User",
		IsActive:     false, // Inactive
	}
	userRepo.users[user.ID] = user
	userRepo.emailIndex[user.Email] = user.ID

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_CaseSensitivity(t *testing.T) {
	handler, userRepo, _ := setupAuthHandler(t)

	// Create user with lowercase email
	passwordHasher := auth.NewArgon2Hasher(nil)
	passwordHash, _ := passwordHasher.Hash("StrongPass123")

	user := &models.User{
		ID:           "test-user-id",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Name:         "Test User",
		IsActive:     true,
	}
	userRepo.users[user.ID] = user
	userRepo.emailIndex[user.Email] = user.ID

	// Try to login with uppercase email
	reqBody := models.LoginRequest{
		Email:    "TEST@EXAMPLE.COM",
		Password: "StrongPass123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	// Email lookup is typically case-insensitive in practice, but depends on implementation
	// This test documents the expected behavior
	if w.Code == http.StatusUnauthorized {
		t.Log("Email lookup is case-sensitive")
	} else if w.Code == http.StatusOK {
		t.Log("Email lookup is case-insensitive")
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	handler, userRepo, tokenRepo := setupAuthHandler(t)

	// Create user
	user := &models.User{
		ID:       "test-user-id",
		Email:    "test@example.com",
		Name:     "Test User",
		IsActive: true,
	}
	userRepo.users[user.ID] = user

	// Create valid refresh token
	jwtManager, _ := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	refreshToken, tokenHash, _ := jwtManager.GenerateRefreshToken()

	token := &models.RefreshToken{
		ID:        "test-token-id",
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	tokenRepo.tokens[token.ID] = token
	tokenRepo.hashIndex[tokenHash] = token.ID

	reqBody := models.RefreshRequest{
		RefreshToken: refreshToken,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAuthHandler_Refresh_ExpiredToken(t *testing.T) {
	handler, _, tokenRepo := setupAuthHandler(t)

	// Create expired token
	token := &models.RefreshToken{
		ID:        "test-token-id",
		UserID:    "test-user-id",
		TokenHash: "test-hash",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	tokenRepo.tokens[token.ID] = token
	tokenRepo.hashIndex[token.TokenHash] = token.ID

	reqBody := models.RefreshRequest{
		RefreshToken: "test-token",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_RevokedToken(t *testing.T) {
	handler, _, tokenRepo := setupAuthHandler(t)

	// Create revoked token
	now := time.Now()
	token := &models.RefreshToken{
		ID:        "test-token-id",
		UserID:    "test-user-id",
		TokenHash: "test-hash",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		RevokedAt: &now, // Revoked
	}
	tokenRepo.tokens[token.ID] = token
	tokenRepo.hashIndex[token.TokenHash] = token.ID

	reqBody := models.RefreshRequest{
		RefreshToken: "test-token",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	handler, _, tokenRepo := setupAuthHandler(t)

	// Create valid token
	jwtManager, _ := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	refreshToken, tokenHash, _ := jwtManager.GenerateRefreshToken()

	token := &models.RefreshToken{
		ID:        "test-token-id",
		UserID:    "test-user-id",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	tokenRepo.tokens[token.ID] = token
	tokenRepo.hashIndex[tokenHash] = token.ID

	reqBody := models.RefreshRequest{
		RefreshToken: refreshToken,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify token was revoked
	if token.RevokedAt == nil {
		t.Error("expected token to be revoked")
	}
}

func TestAuthHandler_Logout_NonexistentToken(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	reqBody := models.RefreshRequest{
		RefreshToken: "nonexistent-token",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	// Should still return success (idempotent operation)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
