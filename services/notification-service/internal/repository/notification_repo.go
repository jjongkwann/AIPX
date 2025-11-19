package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRepository defines the interface for notification data access
type NotificationRepository interface {
	// Preferences
	GetUserPreferences(ctx context.Context, userID string) ([]*Preference, error)
	GetPreference(ctx context.Context, userID, channel string) (*Preference, error)
	CreatePreference(ctx context.Context, pref *Preference) error
	UpdatePreference(ctx context.Context, pref *Preference) error
	DeletePreference(ctx context.Context, userID, channel string) error

	// History
	SaveHistory(ctx context.Context, history *History) error
	GetHistory(ctx context.Context, userID string, limit int) ([]*History, error)
	GetHistoryByStatus(ctx context.Context, status string, limit int) ([]*History, error)
	UpdateHistoryStatus(ctx context.Context, id, status, errorMsg string) error
}

// Preference represents a user's notification channel preference
type Preference struct {
	ID        string
	UserID    string
	Channel   string
	Enabled   bool
	Config    map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

// History represents a notification history record
type History struct {
	ID           string
	UserID       string
	Channel      string
	EventType    string
	Title        string
	Message      string
	Payload      map[string]interface{}
	Status       string
	ErrorMessage string
	RetryCount   int
	SentAt       *time.Time
	CreatedAt    time.Time
}

// PostgresRepository implements NotificationRepository using PostgreSQL
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool, logger *logger.Logger) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}
}

// GetUserPreferences gets all notification preferences for a user
func (r *PostgresRepository) GetUserPreferences(ctx context.Context, userID string) ([]*Preference, error) {
	query := `
		SELECT id, user_id, channel, enabled, config, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
		ORDER BY channel
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer rows.Close()

	var preferences []*Preference
	for rows.Next() {
		pref, err := r.scanPreference(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}
		preferences = append(preferences, pref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return preferences, nil
}

// GetPreference gets a specific notification preference
func (r *PostgresRepository) GetPreference(ctx context.Context, userID, channel string) (*Preference, error) {
	query := `
		SELECT id, user_id, channel, enabled, config, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND channel = $2
	`

	row := r.pool.QueryRow(ctx, query, userID, channel)
	pref, err := r.scanPreference(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}

	return pref, nil
}

// CreatePreference creates a new notification preference
func (r *PostgresRepository) CreatePreference(ctx context.Context, pref *Preference) error {
	configJSON, err := json.Marshal(pref.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO notification_preferences (user_id, channel, enabled, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err = r.pool.QueryRow(ctx, query, pref.UserID, pref.Channel, pref.Enabled, configJSON).
		Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create preference: %w", err)
	}

	return nil
}

// UpdatePreference updates a notification preference
func (r *PostgresRepository) UpdatePreference(ctx context.Context, pref *Preference) error {
	configJSON, err := json.Marshal(pref.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE notification_preferences
		SET enabled = $1, config = $2, updated_at = NOW()
		WHERE user_id = $3 AND channel = $4
		RETURNING updated_at
	`

	err = r.pool.QueryRow(ctx, query, pref.Enabled, configJSON, pref.UserID, pref.Channel).
		Scan(&pref.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update preference: %w", err)
	}

	return nil
}

// DeletePreference deletes a notification preference
func (r *PostgresRepository) DeletePreference(ctx context.Context, userID, channel string) error {
	query := `
		DELETE FROM notification_preferences
		WHERE user_id = $1 AND channel = $2
	`

	result, err := r.pool.Exec(ctx, query, userID, channel)
	if err != nil {
		return fmt.Errorf("failed to delete preference: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("preference not found")
	}

	return nil
}

// SaveHistory saves a notification history record
func (r *PostgresRepository) SaveHistory(ctx context.Context, history *History) error {
	payloadJSON, err := json.Marshal(history.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO notification_history
		(user_id, channel, event_type, title, message, payload, status, error_message, retry_count, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err = r.pool.QueryRow(ctx, query,
		history.UserID,
		history.Channel,
		history.EventType,
		history.Title,
		history.Message,
		payloadJSON,
		history.Status,
		history.ErrorMessage,
		history.RetryCount,
		history.SentAt,
	).Scan(&history.ID, &history.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// GetHistory gets notification history for a user
func (r *PostgresRepository) GetHistory(ctx context.Context, userID string, limit int) ([]*History, error) {
	query := `
		SELECT id, user_id, channel, event_type, title, message, payload,
		       status, error_message, retry_count, sent_at, created_at
		FROM notification_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []*History
	for rows.Next() {
		h, err := r.scanHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return history, nil
}

// GetHistoryByStatus gets notification history by status
func (r *PostgresRepository) GetHistoryByStatus(ctx context.Context, status string, limit int) ([]*History, error) {
	query := `
		SELECT id, user_id, channel, event_type, title, message, payload,
		       status, error_message, retry_count, sent_at, created_at
		FROM notification_history
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history by status: %w", err)
	}
	defer rows.Close()

	var history []*History
	for rows.Next() {
		h, err := r.scanHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return history, nil
}

// UpdateHistoryStatus updates the status of a notification history record
func (r *PostgresRepository) UpdateHistoryStatus(ctx context.Context, id, status, errorMsg string) error {
	query := `
		UPDATE notification_history
		SET status = $1, error_message = $2, sent_at = NOW()
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update history status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("history record not found")
	}

	return nil
}

// scanPreference scans a preference row
func (r *PostgresRepository) scanPreference(row pgx.Row) (*Preference, error) {
	var pref Preference
	var configJSON []byte

	err := row.Scan(
		&pref.ID,
		&pref.UserID,
		&pref.Channel,
		&pref.Enabled,
		&configJSON,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &pref.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return &pref, nil
}

// scanHistory scans a history row
func (r *PostgresRepository) scanHistory(row pgx.Row) (*History, error) {
	var history History
	var payloadJSON []byte

	err := row.Scan(
		&history.ID,
		&history.UserID,
		&history.Channel,
		&history.EventType,
		&history.Title,
		&history.Message,
		&payloadJSON,
		&history.Status,
		&history.ErrorMessage,
		&history.RetryCount,
		&history.SentAt,
		&history.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &history.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	}

	return &history, nil
}
