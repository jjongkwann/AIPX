package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"user-service/internal/models"
)

// AuditLogRepository defines the interface for audit log operations
type AuditLogRepository interface {
	// CreateAuditLog creates a new audit log entry
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	// GetUserAuditLogs retrieves audit logs for a specific user
	GetUserAuditLogs(ctx context.Context, userID string, limit int) ([]*models.AuditLog, error)
	// GetAuditLogsByEventType retrieves audit logs by event type
	GetAuditLogsByEventType(ctx context.Context, eventType models.AuditEventType, limit int) ([]*models.AuditLog, error)
}

// PostgresAuditLogRepository implements AuditLogRepository using PostgreSQL
type PostgresAuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditLogRepository creates a new PostgreSQL audit log repository
func NewPostgresAuditLogRepository(pool *pgxpool.Pool) *PostgresAuditLogRepository {
	return &PostgresAuditLogRepository{
		pool: pool,
	}
}

// CreateAuditLog creates a new audit log entry
func (r *PostgresAuditLogRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, event_type, event_data, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	// Convert event_data map to JSONB
	var eventDataJSON []byte
	var err error
	if log.EventData != nil {
		eventDataJSON, err = json.Marshal(log.EventData)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}
	}

	err = r.pool.QueryRow(
		ctx,
		query,
		log.UserID,
		log.EventType,
		eventDataJSON,
		log.IPAddress,
		log.UserAgent,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetUserAuditLogs retrieves audit logs for a specific user
func (r *PostgresAuditLogRepository) GetUserAuditLogs(ctx context.Context, userID string, limit int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, user_id, event_type, event_data, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		var eventDataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.EventType,
			&eventDataJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Unmarshal event_data JSON
		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &log.EventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}

// GetAuditLogsByEventType retrieves audit logs by event type
func (r *PostgresAuditLogRepository) GetAuditLogsByEventType(
	ctx context.Context,
	eventType models.AuditEventType,
	limit int,
) ([]*models.AuditLog, error) {
	query := `
		SELECT id, user_id, event_type, event_data, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE event_type = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by event type: %w", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		var eventDataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.EventType,
			&eventDataJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Unmarshal event_data JSON
		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &log.EventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}
