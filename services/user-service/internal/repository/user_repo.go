package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"user-service/internal/models"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when trying to create a user with existing email
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// CreateUser creates a new user in the database
	CreateUser(ctx context.Context, user *models.User) error
	// GetUserByID retrieves a user by their ID
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// UpdateUser updates an existing user's information
	UpdateUser(ctx context.Context, user *models.User) error
	// DeleteUser soft deletes a user by setting is_active to false
	DeleteUser(ctx context.Context, id string) error
	// UpdateEmailVerified marks a user's email as verified
	UpdateEmailVerified(ctx context.Context, id string, verified bool) error
	// UpdatePassword updates a user's password hash
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
}

// PostgresUserRepository implements UserRepository using PostgreSQL
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		pool: pool,
	}
}

// CreateUser creates a new user in the database
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, is_active, email_verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.IsActive,
		user.EmailVerified,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if err.Error() == "SQLSTATE 23505" {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID
func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, is_active, email_verified, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = true
	`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.IsActive,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, is_active, email_verified, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.IsActive,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user's information
func (r *PostgresUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, name = $2, email_verified = $3, updated_at = NOW()
		WHERE id = $4 AND is_active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.Name,
		user.EmailVerified,
		user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user by setting is_active to false
func (r *PostgresUserRepository) DeleteUser(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND is_active = true
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateEmailVerified marks a user's email as verified
func (r *PostgresUserRepository) UpdateEmailVerified(ctx context.Context, id string, verified bool) error {
	query := `
		UPDATE users
		SET email_verified = $1, updated_at = NOW()
		WHERE id = $2 AND is_active = true
	`

	result, err := r.pool.Exec(ctx, query, verified, id)
	if err != nil {
		return fmt.Errorf("failed to update email verification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates a user's password hash
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2 AND is_active = true
	`

	result, err := r.pool.Exec(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
