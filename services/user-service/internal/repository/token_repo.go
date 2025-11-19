package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"user-service/internal/models"
)

var (
	// ErrRefreshTokenNotFound is returned when a refresh token is not found
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// ErrRefreshTokenExpired is returned when a refresh token has expired
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	// ErrRefreshTokenRevoked is returned when a refresh token has been revoked
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

// RefreshTokenRepository defines the interface for refresh token operations
type RefreshTokenRepository interface {
	// CreateRefreshToken creates a new refresh token
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	// GetRefreshToken retrieves a refresh token by its ID
	GetRefreshToken(ctx context.Context, id string) (*models.RefreshToken, error)
	// GetRefreshTokenByHash retrieves a refresh token by its hash
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	// GetUserTokens retrieves all active tokens for a user
	GetUserTokens(ctx context.Context, userID string) ([]*models.RefreshToken, error)
	// RevokeToken revokes a specific refresh token
	RevokeToken(ctx context.Context, id string) error
	// RevokeUserTokens revokes all tokens for a user
	RevokeUserTokens(ctx context.Context, userID string) error
	// DeleteExpiredTokens deletes tokens that expired more than the specified duration ago
	DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) error
}

// PostgresRefreshTokenRepository implements RefreshTokenRepository using PostgreSQL
type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository creates a new PostgreSQL refresh token repository
func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{
		pool: pool,
	}
}

// CreateRefreshToken creates a new refresh token
func (r *PostgresRefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.IPAddress,
		token.UserAgent,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by its ID
func (r *PostgresRefreshTokenRepository) GetRefreshToken(ctx context.Context, id string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent
		FROM refresh_tokens
		WHERE id = $1
	`

	token := &models.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.IPAddress,
		&token.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, ErrRefreshTokenExpired
	}

	// Check if token is revoked
	if token.IsRevoked() {
		return nil, ErrRefreshTokenRevoked
	}

	return token, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *PostgresRefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &models.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.IPAddress,
		&token.UserAgent,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, ErrRefreshTokenExpired
	}

	// Check if token is revoked
	if token.IsRevoked() {
		return nil, ErrRefreshTokenRevoked
	}

	return token, nil
}

// GetUserTokens retrieves all active tokens for a user
func (r *PostgresRefreshTokenRepository) GetUserTokens(ctx context.Context, userID string) ([]*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.RefreshToken
	for rows.Next() {
		token := &models.RefreshToken{}
		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.ExpiresAt,
			&token.CreatedAt,
			&token.RevokedAt,
			&token.IPAddress,
			&token.UserAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating refresh tokens: %w", err)
	}

	return tokens, nil
}

// RevokeToken revokes a specific refresh token
func (r *PostgresRefreshTokenRepository) RevokeToken(ctx context.Context, id string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}

// RevokeUserTokens revokes all tokens for a user
func (r *PostgresRefreshTokenRepository) RevokeUserTokens(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// DeleteExpiredTokens deletes tokens that expired more than the specified duration ago
func (r *PostgresRefreshTokenRepository) DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1
	`

	cutoffTime := time.Now().Add(-olderThan)
	result, err := r.pool.Exec(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		// Log the number of deleted tokens (in production, use proper logging)
		_ = rowsAffected
	}

	return nil
}
