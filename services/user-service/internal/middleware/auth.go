package middleware

import (
	"context"
	"net/http"
	"strings"

	"user-service/internal/auth"
	"user-service/internal/models"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UserEmailKey is the context key for user email
	UserEmailKey contextKey = "user_email"
	// ClaimsKey is the context key for JWT claims
	ClaimsKey contextKey = "claims"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			// Check for Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			token := parts[1]
			if token == "" {
				respondWithError(w, http.StatusUnauthorized, "missing token")
				return
			}

			// Validate token
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				switch err {
				case auth.ErrTokenExpired:
					respondWithError(w, http.StatusUnauthorized, "token expired")
				case auth.ErrInvalidToken:
					respondWithError(w, http.StatusUnauthorized, "invalid token")
				default:
					respondWithError(w, http.StatusUnauthorized, "token validation failed")
				}
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetUserEmail extracts the user email from the request context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// GetClaims extracts the JWT claims from the request context
func GetClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*auth.Claims)
	return claims, ok
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := models.NewErrorResponse(message)
	// Using simple JSON encoding inline to avoid circular dependencies
	w.Write([]byte(`{"success":false,"error":"` + message + `","timestamp":"` + response.Timestamp.Format("2006-01-02T15:04:05Z07:00") + `"}`))
}
