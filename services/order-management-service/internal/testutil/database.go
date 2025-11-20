package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer represents a test PostgreSQL container
type PostgresContainer struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
}

// NewTestDB creates a test database using testcontainers
func NewTestDB(t *testing.T) *PostgresContainer {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	if err := initSchema(ctx, pool); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return &PostgresContainer{
		Container: container,
		Pool:      pool,
	}
}

// Close closes the test database and container
func (pc *PostgresContainer) Close(t *testing.T) {
	ctx := context.Background()
	if pc.Pool != nil {
		pc.Pool.Close()
	}
	if pc.Container != nil {
		if err := pc.Container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// initSchema initializes the database schema
func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
		CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			strategy_id VARCHAR(255),
			symbol VARCHAR(50) NOT NULL,
			side VARCHAR(10) NOT NULL,
			order_type VARCHAR(10) NOT NULL,
			price DECIMAL(20, 2),
			quantity INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL,
			broker_order_id VARCHAR(255),
			filled_price DECIMAL(20, 2),
			filled_quantity INTEGER,
			reject_reason TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
		CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
		CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

		CREATE TABLE IF NOT EXISTS order_audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			order_id UUID NOT NULL REFERENCES orders(id),
			status VARCHAR(20) NOT NULL,
			reason TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_audit_log_order_id ON order_audit_log(order_id);
	`

	_, err := pool.Exec(ctx, schema)
	return err
}
