package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"user-service/internal/auth"
)

func setupTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()

	manager, err := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	return manager
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	// Generate valid token
	userID := "test-user-123"
	email := "test@example.com"
	token, err := jwtManager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context values were set
		gotUserID, ok := GetUserID(r.Context())
		if !ok {
			t.Error("User ID not found in context")
		}
		if gotUserID != userID {
			t.Errorf("User ID = %v, want %v", gotUserID, userID)
		}

		gotEmail, ok := GetUserEmail(r.Context())
		if !ok {
			t.Error("Email not found in context")
		}
		if gotEmail != email {
			t.Errorf("Email = %v, want %v", gotEmail, email)
		}

		claims, ok := GetClaims(r.Context())
		if !ok {
			t.Error("Claims not found in context")
		}
		if claims.UserID != userID {
			t.Errorf("Claims.UserID = %v, want %v", claims.UserID, userID)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with auth middleware
	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	// Create request with valid token
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without token")
	})

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid format")
	})

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	invalidHeaders := []string{
		"InvalidFormat",
		"Bearer",
		"Bearer ",
		"Basic token123",
		"token123",
	}

	for _, header := range invalidHeaders {
		t.Run(header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Authorization", header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d for '%s', got %d", http.StatusUnauthorized, header, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	shortTTLManager, _ := auth.NewJWTManager(
		"test-access-secret-key-minimum-32-characters",
		"test-refresh-secret-key-minimum-32-characters",
		1*time.Nanosecond, // Expire immediately
		7*24*time.Hour,
		"test-issuer",
	)

	token, err := shortTTLManager.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with expired token")
	})

	middleware := AuthMiddleware(shortTTLManager)
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_TamperedToken(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	token, err := jwtManager.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Tamper with token
	tamperedToken := token[:len(token)-5] + "XXXXX"

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with tampered token")
	})

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tamperedToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_UserContextInjection(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	userID := "user-12345"
	email := "user@example.com"
	token, _ := jwtManager.GenerateAccessToken(userID, email)

	var capturedUserID string
	var capturedEmail string
	var capturedClaims *auth.Claims

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, _ = GetUserID(r.Context())
		capturedEmail, _ = GetUserEmail(r.Context())
		capturedClaims, _ = GetClaims(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if capturedUserID != userID {
		t.Errorf("UserID = %v, want %v", capturedUserID, userID)
	}

	if capturedEmail != email {
		t.Errorf("Email = %v, want %v", capturedEmail, email)
	}

	if capturedClaims == nil {
		t.Fatal("Claims should not be nil")
	}

	if capturedClaims.UserID != userID {
		t.Errorf("Claims.UserID = %v, want %v", capturedClaims.UserID, userID)
	}
}

func TestAuthMiddleware_MalformedToken(t *testing.T) {
	jwtManager := setupTestJWTManager(t)

	malformedTokens := []string{
		"",
		"not.a.token",
		"invalid",
		"header.payload", // Missing signature
		"a.b.c.d",        // Too many parts
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with malformed token")
	})

	middleware := AuthMiddleware(jwtManager)
	handler := middleware(testHandler)

	for _, token := range malformedTokens {
		t.Run("malformed_"+token, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d for malformed token, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	ctx := context.Background()

	// Test with no user ID
	_, ok := GetUserID(ctx)
	if ok {
		t.Error("Expected false for context without user ID")
	}

	// Test with user ID
	userID := "test-user-123"
	ctx = context.WithValue(ctx, UserIDKey, userID)

	gotUserID, ok := GetUserID(ctx)
	if !ok {
		t.Error("Expected true for context with user ID")
	}

	if gotUserID != userID {
		t.Errorf("UserID = %v, want %v", gotUserID, userID)
	}

	// Test with wrong type
	ctx = context.WithValue(ctx, UserIDKey, 12345)
	_, ok = GetUserID(ctx)
	if ok {
		t.Error("Expected false for context with wrong type")
	}
}

func TestGetUserEmail(t *testing.T) {
	ctx := context.Background()

	// Test with no email
	_, ok := GetUserEmail(ctx)
	if ok {
		t.Error("Expected false for context without email")
	}

	// Test with email
	email := "test@example.com"
	ctx = context.WithValue(ctx, UserEmailKey, email)

	gotEmail, ok := GetUserEmail(ctx)
	if !ok {
		t.Error("Expected true for context with email")
	}

	if gotEmail != email {
		t.Errorf("Email = %v, want %v", gotEmail, email)
	}
}

func TestGetClaims(t *testing.T) {
	ctx := context.Background()

	// Test with no claims
	_, ok := GetClaims(ctx)
	if ok {
		t.Error("Expected false for context without claims")
	}

	// Test with claims
	claims := &auth.Claims{
		UserID: "test-user",
		Email:  "test@example.com",
		Type:   string(auth.AccessToken),
	}
	ctx = context.WithValue(ctx, ClaimsKey, claims)

	gotClaims, ok := GetClaims(ctx)
	if !ok {
		t.Error("Expected true for context with claims")
	}

	if gotClaims.UserID != claims.UserID {
		t.Errorf("Claims.UserID = %v, want %v", gotClaims.UserID, claims.UserID)
	}
}
