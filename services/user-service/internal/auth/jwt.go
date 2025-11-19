package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when the token has expired
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidSigningMethod is returned when the signing method is invalid
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	// AccessToken is a short-lived token for API access
	AccessToken TokenType = "access"
	// RefreshToken is a long-lived token for refreshing access tokens
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims for access tokens
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer string) (*JWTManager, error) {
	if accessSecret == "" {
		return nil, errors.New("access secret cannot be empty")
	}
	if refreshSecret == "" {
		return nil, errors.New("refresh secret cannot be empty")
	}
	if len(accessSecret) < 32 {
		return nil, errors.New("access secret must be at least 32 characters")
	}
	if len(refreshSecret) < 32 {
		return nil, errors.New("refresh secret must be at least 32 characters")
	}

	return &JWTManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}, nil
}

// GenerateAccessToken generates a new access token for a user
func (m *JWTManager) GenerateAccessToken(userID, email string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   string(AccessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token (opaque token)
// Returns the token string and its hash for database storage
func (m *JWTManager) GenerateRefreshToken() (token string, tokenHash string, err error) {
	// Generate 32 bytes of random data
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64 for transmission
	token = base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(token))
	tokenHash = base64.URLEncoding.EncodeToString(hash[:])

	return token, tokenHash, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
		}
		return m.accessSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify token type
	if claims.Type != string(AccessToken) {
		return nil, errors.New("invalid token type")
	}

	// Verify issuer
	if claims.Issuer != m.issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}

// HashRefreshToken creates a hash of a refresh token for comparison with stored hash
func (m *JWTManager) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// GetAccessTTL returns the access token TTL
func (m *JWTManager) GetAccessTTL() time.Duration {
	return m.accessTTL
}

// GetRefreshTTL returns the refresh token TTL
func (m *JWTManager) GetRefreshTTL() time.Duration {
	return m.refreshTTL
}

// GetAccessTTLSeconds returns the access token TTL in seconds
func (m *JWTManager) GetAccessTTLSeconds() int64 {
	return int64(m.accessTTL.Seconds())
}

// GetRefreshTTLSeconds returns the refresh token TTL in seconds
func (m *JWTManager) GetRefreshTTLSeconds() int64 {
	return int64(m.refreshTTL.Seconds())
}
