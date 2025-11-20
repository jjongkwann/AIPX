package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"user-service/internal/auth"
	"user-service/internal/handlers"
	"user-service/internal/middleware"
	"user-service/internal/repository"

	"github.com/go-chi/chi/v5"
)

// TestServer represents a test HTTP server with all dependencies
type TestServer struct {
	Router        *chi.Mux
	Server        *httptest.Server
	UserRepo      repository.UserRepository
	TokenRepo     repository.RefreshTokenRepository
	JWTManager    *auth.JWTManager
	PasswordHasher auth.PasswordHasher
}

// NewTestServer creates a new test server with mock repositories
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create mock repositories
	userRepo := NewMockUserRepository()
	tokenRepo := NewMockTokenRepository()

	// Create JWT manager with test secrets
	jwtManager, err := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Create password hasher
	passwordHasher := auth.NewArgon2Hasher(nil)

	// Create handlers
	authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, passwordHasher, jwtManager)
	userHandler := handlers.NewUserHandler(userRepo)

	// Setup router
	r := chi.NewRouter()

	// Public routes
	r.Post("/api/v1/auth/signup", authHandler.Signup)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/refresh", authHandler.Refresh)
	r.Post("/api/v1/auth/logout", authHandler.Logout)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtManager))
		r.Get("/api/v1/users/me", userHandler.GetMe)
		r.Put("/api/v1/users/me", userHandler.UpdateMe)
		r.Delete("/api/v1/users/me", userHandler.DeleteMe)
	})

	// Create test server
	server := httptest.NewServer(r)

	return &TestServer{
		Router:        r,
		Server:        server,
		UserRepo:      userRepo,
		TokenRepo:     tokenRepo,
		JWTManager:    jwtManager,
		PasswordHasher: passwordHasher,
	}
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// MakeRequest makes an HTTP request to the test server
func (ts *TestServer) MakeRequest(method, path string, body interface{}, authToken string) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	return http.DefaultClient.Do(req)
}

// GetAuthToken generates a valid auth token for testing
func (ts *TestServer) GetAuthToken(userID, email string) (string, error) {
	return ts.JWTManager.GenerateAccessToken(userID, email)
}

// DecodeResponse decodes a JSON response into the provided interface
func DecodeResponse(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}
