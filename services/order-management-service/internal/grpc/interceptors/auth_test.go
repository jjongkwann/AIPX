package interceptors

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	testSecret   = "test-secret-key-32-bytes-long!!!"
	testIssuer   = "test-issuer"
	testAudience = "test-audience"
	testUserID   = "user-123"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func createTestToken(claims jwt.MapClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func createValidClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub": testUserID,
		"iss": testIssuer,
		"aud": testAudience,
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"jti": "token-123",
	}
}

func TestNewAuthInterceptor(t *testing.T) {
	_, client := setupTestRedis(t)
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)
	assert.NotNil(t, interceptor)
	assert.Equal(t, []byte(testSecret), interceptor.jwtSecret)
	assert.Equal(t, testIssuer, interceptor.expectedIss)
	assert.Equal(t, testAudience, interceptor.expectedAud)
	assert.NotNil(t, interceptor.redisClient)
	assert.NotEmpty(t, interceptor.publicMethods)
}

func TestValidateToken_ValidToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)

	token, err := createTestToken(createValidClaims(), testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	userID, err := interceptor.validateToken(ctx)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, userID)
}

func TestValidateToken_MissingMetadata(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	_, err := interceptor.validateToken(context.Background())
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestValidateToken_MissingAuthHeader(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(nil))

	_, err := interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing authorization header")
}

func TestValidateToken_InvalidAuthFormat(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	testCases := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "token123"},
		{"Wrong prefix", "Basic token123"},
		{"Empty token", "Bearer "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(),
				metadata.Pairs("authorization", tc.header))

			_, err := interceptor.validateToken(ctx)
			assert.Error(t, err)

			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.Unauthenticated, st.Code())
		})
	}
}

func TestValidateToken_InvalidSigningMethod(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	// Create token with RS256 instead of HS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, createValidClaims())
	tokenString, _ := token.SignedString([]byte(testSecret)) // This will fail but produce a malformed token

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+tokenString))

	_, err := interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	claims := createValidClaims()
	claims["exp"] = time.Now().Add(-time.Hour).Unix() // Expired 1 hour ago

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid token")
	assert.Contains(t, st.Message(), "expired")
}

func TestValidateToken_NotYetValid(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	claims := createValidClaims()
	claims["nbf"] = time.Now().Add(time.Hour).Unix() // Valid in 1 hour

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid token")
	assert.Contains(t, st.Message(), "not valid yet")
}

func TestValidateToken_FutureIssuedToken(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	claims := createValidClaims()
	claims["iat"] = time.Now().Add(10 * time.Minute).Unix() // Issued 10 minutes in the future

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "token issued in the future")
}

func TestValidateToken_InvalidIssuer(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	claims := createValidClaims()
	claims["iss"] = "wrong-issuer"

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid issuer")
}

func TestValidateToken_InvalidAudience(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	claims := createValidClaims()
	claims["aud"] = "wrong-audience"

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "invalid audience")
}

func TestValidateToken_RevokedToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)

	claims := createValidClaims()
	jti := "revoked-token-123"
	claims["jti"] = jti

	// Add token to revoked list
	err := client.SAdd(context.Background(), "revoked_tokens", jti).Err()
	require.NoError(t, err)

	token, err := createTestToken(claims, testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "token has been revoked")
}

func TestValidateToken_BlockedUser(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)

	// Block the user
	err := client.SAdd(context.Background(), "blocked_users", testUserID).Err()
	require.NoError(t, err)

	token, err := createTestToken(createValidClaims(), testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	_, err = interceptor.validateToken(ctx)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Contains(t, st.Message(), "user account is blocked")
}

func TestUnaryInterceptor_PublicMethod(t *testing.T) {
	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, nil)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	unaryInterceptor := interceptor.Unary()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	resp, err := unaryInterceptor(context.Background(), nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

func TestUnaryInterceptor_AuthRequired(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		userID, ok := GetUserIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, testUserID, userID)
		return "response", nil
	}

	token, err := createTestToken(createValidClaims(), testSecret)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	unaryInterceptor := interceptor.Unary()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/api.OrderService/CreateOrder",
	}

	resp, err := unaryInterceptor(ctx, nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestRevokeToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	interceptor := NewAuthInterceptor(testSecret, testIssuer, testAudience, client)

	jti := "token-to-revoke"
	err := interceptor.RevokeToken(context.Background(), jti, time.Hour)
	assert.NoError(t, err)

	// Check token is in revoked set
	isMember, err := client.SIsMember(context.Background(), "revoked_tokens", jti).Result()
	assert.NoError(t, err)
	assert.True(t, isMember)

	// Check individual revoked key exists
	exists, err := client.Exists(context.Background(), "revoked_token:"+jti).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestGetUserIDFromContext(t *testing.T) {
	// Test with user ID in context
	ctx := context.WithValue(context.Background(), userIDKey, "test-user")
	userID, ok := GetUserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "test-user", userID)

	// Test without user ID in context
	userID, ok = GetUserIDFromContext(context.Background())
	assert.False(t, ok)
	assert.Empty(t, userID)
}
