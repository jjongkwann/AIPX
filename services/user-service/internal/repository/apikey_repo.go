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
	// ErrAPIKeyNotFound is returned when an API key is not found
	ErrAPIKeyNotFound = errors.New("api key not found")
	// ErrAPIKeyAlreadyExists is returned when trying to create a duplicate API key
	ErrAPIKeyAlreadyExists = errors.New("api key for this broker already exists")
)

// APIKeyRepository defines the interface for API key data operations
type APIKeyRepository interface {
	// CreateAPIKey creates a new encrypted API key in the database
	CreateAPIKey(ctx context.Context, key *models.APIKey) error
	// GetAPIKey retrieves an API key by its ID
	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
	// GetAPIKeysByUserID retrieves all API keys for a user
	GetAPIKeysByUserID(ctx context.Context, userID string) ([]*models.APIKey, error)
	// GetAPIKeyByUserAndBroker retrieves a specific broker's API key for a user
	GetAPIKeyByUserAndBroker(ctx context.Context, userID string, broker models.BrokerType) (*models.APIKey, error)
	// UpdateAPIKey updates an existing API key
	UpdateAPIKey(ctx context.Context, key *models.APIKey) error
	// DeleteAPIKey deletes an API key by its ID
	DeleteAPIKey(ctx context.Context, id string) error
	// UpdateLastUsed updates the last_used_at timestamp for an API key
	UpdateLastUsed(ctx context.Context, id string) error
	// SetActive sets the active status of an API key
	SetActive(ctx context.Context, id string, active bool) error
}

// PostgresAPIKeyRepository implements APIKeyRepository using PostgreSQL
type PostgresAPIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAPIKeyRepository creates a new PostgreSQL API key repository
func NewPostgresAPIKeyRepository(pool *pgxpool.Pool) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{
		pool: pool,
	}
}

// CreateAPIKey creates a new encrypted API key in the database
func (r *PostgresAPIKeyRepository) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	query := `
		INSERT INTO api_keys (user_id, broker, key_encrypted, secret_encrypted, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		key.UserID,
		key.Broker,
		key.KeyEncrypted,
		key.SecretEncrypted,
		key.IsActive,
	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate user_id + broker)
		if err.Error() == "SQLSTATE 23505" {
			return ErrAPIKeyAlreadyExists
		}
		return fmt.Errorf("failed to create api key: %w", err)
	}

	return nil
}

// GetAPIKey retrieves an API key by its ID
func (r *PostgresAPIKeyRepository) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, broker, key_encrypted, secret_encrypted,
		       is_active, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE id = $1
	`

	key := &models.APIKey{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&key.ID,
		&key.UserID,
		&key.Broker,
		&key.KeyEncrypted,
		&key.SecretEncrypted,
		&key.IsActive,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.LastUsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return key, nil
}

// GetAPIKeysByUserID retrieves all API keys for a user
func (r *PostgresAPIKeyRepository) GetAPIKeysByUserID(ctx context.Context, userID string) ([]*models.APIKey, error) {
	query := `
		SELECT id, user_id, broker, key_encrypted, secret_encrypted,
		       is_active, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get api keys for user: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key := &models.APIKey{}
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Broker,
			&key.KeyEncrypted,
			&key.SecretEncrypted,
			&key.IsActive,
			&key.CreatedAt,
			&key.UpdatedAt,
			&key.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating api keys: %w", err)
	}

	return keys, nil
}

// GetAPIKeyByUserAndBroker retrieves a specific broker's API key for a user
func (r *PostgresAPIKeyRepository) GetAPIKeyByUserAndBroker(
	ctx context.Context,
	userID string,
	broker models.BrokerType,
) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, broker, key_encrypted, secret_encrypted,
		       is_active, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE user_id = $1 AND broker = $2
	`

	key := &models.APIKey{}
	err := r.pool.QueryRow(ctx, query, userID, broker).Scan(
		&key.ID,
		&key.UserID,
		&key.Broker,
		&key.KeyEncrypted,
		&key.SecretEncrypted,
		&key.IsActive,
		&key.CreatedAt,
		&key.UpdatedAt,
		&key.LastUsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by user and broker: %w", err)
	}

	return key, nil
}

// UpdateAPIKey updates an existing API key
func (r *PostgresAPIKeyRepository) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	query := `
		UPDATE api_keys
		SET key_encrypted = $1, secret_encrypted = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		key.KeyEncrypted,
		key.SecretEncrypted,
		key.IsActive,
		key.ID,
	).Scan(&key.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("failed to update api key: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key by its ID
func (r *PostgresAPIKeyRepository) DeleteAPIKey(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for an API key
func (r *PostgresAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// SetActive sets the active status of an API key
func (r *PostgresAPIKeyRepository) SetActive(ctx context.Context, id string, active bool) error {
	query := `
		UPDATE api_keys
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, active, id)
	if err != nil {
		return fmt.Errorf("failed to set api key active status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}
