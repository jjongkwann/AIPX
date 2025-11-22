package interceptors

import (
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides JWT-based authentication for gRPC services
type AuthInterceptor struct {
	jwtSecret     []byte
	expectedIss   string
	expectedAud   string
	redisClient   *redis.Client
	publicMethods map[string]bool
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(secret, issuer, audience string, redis *redis.Client) *AuthInterceptor {
	return &AuthInterceptor{
		jwtSecret:   []byte(secret),
		expectedIss: issuer,
		expectedAud: audience,
		redisClient: redis,
		publicMethods: map[string]bool{
			"/grpc.health.v1.Health/Check":        true,
			"/grpc.health.v1.Health/Watch":        true,
			"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
		},
	}
}

// Unary returns a unary server interceptor for authentication
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for public methods
		if i.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract and validate token
		userID, err := i.validateToken(ctx)
		if err != nil {
			return nil, err
		}

		// Add user ID to context
		ctx = context.WithValue(ctx, userIDKey, userID)
		return handler(ctx, req)
	}
}

// Stream returns a stream server interceptor for authentication
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip auth for public methods
		if i.publicMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		// Extract and validate token
		userID, err := i.validateToken(ss.Context())
		if err != nil {
			return err
		}

		// Create wrapped stream with user context
		wrapped := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), userIDKey, userID),
		}

		return handler(srv, wrapped)
	}
}

// validateToken validates the JWT token from the request
func (i *AuthInterceptor) validateToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	// Extract Bearer token
	parts := strings.SplitN(authHeader[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}
	tokenString := parts[1]

	// Parse and validate JWT with strict validation
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method - only allow HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Errorf(codes.Unauthenticated, "invalid signing method: %v", token.Header["alg"])
		}
		// Ensure it's specifically HS256
		if token.Method.Alg() != "HS256" {
			return nil, status.Error(codes.Unauthenticated, "only HS256 signing method is allowed")
		}
		return i.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", status.Error(codes.Unauthenticated, "invalid token claims")
	}

	// Validate expiration (exp)
	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing expiration claim")
	}
	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return "", status.Error(codes.Unauthenticated, "token expired")
	}

	// Validate not before (nbf) if present
	if nbf, ok := claims["nbf"].(float64); ok {
		if time.Unix(int64(nbf), 0).After(time.Now()) {
			return "", status.Error(codes.Unauthenticated, "token not yet valid")
		}
	}

	// Validate issued at (iat) - reject tokens from the future
	if iat, ok := claims["iat"].(float64); ok {
		if time.Unix(int64(iat), 0).After(time.Now().Add(5 * time.Minute)) { // 5 min clock skew tolerance
			return "", status.Error(codes.Unauthenticated, "token issued in the future")
		}
	}

	// Validate issuer (iss)
	iss, ok := claims["iss"].(string)
	if !ok || iss != i.expectedIss {
		return "", status.Errorf(codes.Unauthenticated, "invalid issuer: expected %s, got %s", i.expectedIss, iss)
	}

	// Validate audience (aud)
	aud, ok := claims["aud"].(string)
	if !ok || aud != i.expectedAud {
		return "", status.Errorf(codes.Unauthenticated, "invalid audience: expected %s, got %s", i.expectedAud, aud)
	}

	// Extract user ID from subject (sub)
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", status.Error(codes.Unauthenticated, "missing subject claim")
	}

	// Check token ID (jti) against revocation list if present
	if jti, ok := claims["jti"].(string); ok && jti != "" {
		revoked, err := i.isTokenRevoked(ctx, jti)
		if err != nil {
			// Log error but don't fail authentication if Redis is down
			// In production, you might want to fail closed instead
			// return "", status.Error(codes.Internal, "failed to check token revocation")
		} else if revoked {
			return "", status.Error(codes.Unauthenticated, "token has been revoked")
		}
	}

	// Additional check: verify user is not blocked
	if i.redisClient != nil {
		blocked, err := i.isUserBlocked(ctx, userID)
		if err != nil {
			// Log error but continue (fail open)
			// In production, consider failing closed
		} else if blocked {
			return "", status.Error(codes.PermissionDenied, "user account is blocked")
		}
	}

	return userID, nil
}

// isTokenRevoked checks if a token has been revoked
func (i *AuthInterceptor) isTokenRevoked(ctx context.Context, jti string) (bool, error) {
	if i.redisClient == nil {
		return false, nil
	}

	// Check if JTI exists in revoked tokens set
	revoked, err := i.redisClient.SIsMember(ctx, "revoked_tokens", jti).Result()
	if err != nil {
		return false, err
	}

	return revoked, nil
}

// isUserBlocked checks if a user account is blocked
func (i *AuthInterceptor) isUserBlocked(ctx context.Context, userID string) (bool, error) {
	if i.redisClient == nil {
		return false, nil
	}

	// Check if user is in blocked users set
	blocked, err := i.redisClient.SIsMember(ctx, "blocked_users", userID).Result()
	if err != nil {
		return false, err
	}

	return blocked, nil
}

// RevokeToken adds a token to the revocation list
func (i *AuthInterceptor) RevokeToken(ctx context.Context, jti string, expiration time.Duration) error {
	if i.redisClient == nil {
		return nil
	}

	// Add to revoked tokens set
	err := i.redisClient.SAdd(ctx, "revoked_tokens", jti).Err()
	if err != nil {
		return err
	}

	// Set expiration for the revoked token entry
	// This ensures revoked tokens are automatically cleaned up after they expire
	revokedKey := "revoked_token:" + jti
	err = i.redisClient.Set(ctx, revokedKey, "1", expiration).Err()
	if err != nil {
		return err
	}

	// Periodically clean up the set (optional, could be done in a background job)
	// This removes expired JTIs from the set
	go i.cleanupExpiredTokens(context.Background())

	return nil
}

// cleanupExpiredTokens removes expired JTIs from the revocation set
func (i *AuthInterceptor) cleanupExpiredTokens(ctx context.Context) {
	if i.redisClient == nil {
		return
	}

	// Get all JTIs in the revoked set
	jtis, err := i.redisClient.SMembers(ctx, "revoked_tokens").Result()
	if err != nil {
		return
	}

	for _, jti := range jtis {
		// Check if the individual revoked token key still exists
		exists, err := i.redisClient.Exists(ctx, "revoked_token:"+jti).Result()
		if err != nil {
			continue
		}

		// If the key doesn't exist, the token has expired, remove from set
		if exists == 0 {
			i.redisClient.SRem(ctx, "revoked_tokens", jti)
		}
	}
}

// contextKey type for context values
type contextKey string

const (
	userIDKey contextKey = "user_id"
)

// authenticatedServerStream wraps grpc.ServerStream with authenticated context
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// GetUserIDFromContext extracts the user ID from the context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}