package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/buffer"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/pkg/pb"
)

// TimescaleWriter writes market data to TimescaleDB
type TimescaleWriter struct {
	pool   *pgxpool.Pool
	config *TimescaleConfig

	// Metrics
	totalInserts atomic.Int64
	totalErrors  atomic.Int64
	lastInsertAt atomic.Int64

	// Control
	mu     sync.RWMutex
	closed bool
}

// TimescaleConfig holds TimescaleDB configuration
type TimescaleConfig struct {
	DSN              string
	MaxConns         int
	MinConns         int
	MaxConnLifetime  time.Duration
	MaxConnIdleTime  time.Duration
	HealthCheckPeriod time.Duration
	MaxRetries       int
	RetryBackoff     time.Duration
}

// DefaultTimescaleConfig returns default configuration
func DefaultTimescaleConfig(dsn string) *TimescaleConfig {
	return &TimescaleConfig{
		DSN:              dsn,
		MaxConns:         20,
		MinConns:         5,
		MaxConnLifetime:  30 * time.Minute,
		MaxConnIdleTime:  5 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     time.Second,
	}
}

// NewTimescaleWriter creates a new TimescaleDB writer
func NewTimescaleWriter(ctx context.Context, config *TimescaleConfig) (*TimescaleWriter, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Parse connection config
	poolConfig, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = int32(config.MaxConns)
	poolConfig.MinConns = int32(config.MinConns)
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	tw := &TimescaleWriter{
		pool:   pool,
		config: config,
		closed: false,
	}

	// Initialize schema
	if err := tw.initSchema(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return tw, nil
}

// initSchema creates tables and hypertables if they don't exist
func (tw *TimescaleWriter) initSchema(ctx context.Context) error {
	queries := []string{
		// Create tick_data table
		`CREATE TABLE IF NOT EXISTS tick_data (
			time TIMESTAMPTZ NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			price NUMERIC(20, 8) NOT NULL,
			volume BIGINT NOT NULL,
			change NUMERIC(20, 8),
			change_rate NUMERIC(10, 4)
		)`,

		// Create hypertable for tick_data
		`SELECT create_hypertable('tick_data', 'time', if_not_exists => TRUE)`,

		// Create index on tick_data
		`CREATE INDEX IF NOT EXISTS idx_tick_symbol_time ON tick_data (symbol, time DESC)`,

		// Create orderbook table
		`CREATE TABLE IF NOT EXISTS orderbook (
			time TIMESTAMPTZ NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			bids JSONB NOT NULL,
			asks JSONB NOT NULL
		)`,

		// Create hypertable for orderbook
		`SELECT create_hypertable('orderbook', 'time', if_not_exists => TRUE)`,

		// Create index on orderbook
		`CREATE INDEX IF NOT EXISTS idx_orderbook_symbol_time ON orderbook (symbol, time DESC)`,
	}

	for _, query := range queries {
		if _, err := tw.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// WriteMessages writes a batch of messages to TimescaleDB
func (tw *TimescaleWriter) WriteMessages(ctx context.Context, messages []buffer.BufferedMessage) error {
	if len(messages) == 0 {
		return nil
	}

	tw.mu.RLock()
	if tw.closed {
		tw.mu.RUnlock()
		return ErrWriterClosed
	}
	tw.mu.RUnlock()

	// Separate messages by type
	var ticks []*pb.TickData
	var orderbooks []*pb.OrderBook

	for _, msg := range messages {
		switch msg.Type {
		case buffer.MessageTypeTick:
			if msg.Tick != nil {
				ticks = append(ticks, msg.Tick)
			}
		case buffer.MessageTypeOrderBook:
			if msg.OrderBook != nil {
				orderbooks = append(orderbooks, msg.OrderBook)
			}
		}
	}

	// Write with retries
	var lastErr error
	for attempt := 0; attempt < tw.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(tw.config.RetryBackoff * time.Duration(attempt))
		}

		err := tw.writeBatch(ctx, ticks, orderbooks)
		if err == nil {
			tw.totalInserts.Add(int64(len(messages)))
			tw.lastInsertAt.Store(time.Now().Unix())
			return nil
		}

		lastErr = err
	}

	tw.totalErrors.Add(1)
	return fmt.Errorf("failed after %d retries: %w", tw.config.MaxRetries, lastErr)
}

// writeBatch writes a batch using a transaction
func (tw *TimescaleWriter) writeBatch(ctx context.Context, ticks []*pb.TickData, orderbooks []*pb.OrderBook) error {
	tx, err := tw.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Write tick data
	if len(ticks) > 0 {
		if err := tw.writeTickData(ctx, tx, ticks); err != nil {
			return fmt.Errorf("failed to write tick data: %w", err)
		}
	}

	// Write orderbook data
	if len(orderbooks) > 0 {
		if err := tw.writeOrderBooks(ctx, tx, orderbooks); err != nil {
			return fmt.Errorf("failed to write orderbooks: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// writeTickData writes tick data using COPY
func (tw *TimescaleWriter) writeTickData(ctx context.Context, tx pgx.Tx, ticks []*pb.TickData) error {
	// Use COPY for better performance
	copySource := pgx.CopyFromSlice(len(ticks), func(i int) ([]any, error) {
		tick := ticks[i]
		return []any{
			time.Unix(0, tick.Timestamp),
			tick.Symbol,
			tick.Price,
			tick.Volume,
			tick.Change,
			tick.ChangeRate,
		}, nil
	})

	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"tick_data"},
		[]string{"time", "symbol", "price", "volume", "change", "change_rate"},
		copySource,
	)

	if err != nil {
		return fmt.Errorf("failed to copy tick data: %w", err)
	}

	if count != int64(len(ticks)) {
		return fmt.Errorf("expected to insert %d rows, inserted %d", len(ticks), count)
	}

	return nil
}

// writeOrderBooks writes orderbook data using COPY
func (tw *TimescaleWriter) writeOrderBooks(ctx context.Context, tx pgx.Tx, orderbooks []*pb.OrderBook) error {
	copySource := pgx.CopyFromSlice(len(orderbooks), func(i int) ([]any, error) {
		ob := orderbooks[i]

		// Convert bids and asks to JSONB
		bidsJSON, err := orderbookLevelsToJSON(ob.Bids)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal bids: %w", err)
		}

		asksJSON, err := orderbookLevelsToJSON(ob.Asks)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal asks: %w", err)
		}

		return []any{
			time.Unix(0, ob.Timestamp),
			ob.Symbol,
			bidsJSON,
			asksJSON,
		}, nil
	})

	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"orderbook"},
		[]string{"time", "symbol", "bids", "asks"},
		copySource,
	)

	if err != nil {
		return fmt.Errorf("failed to copy orderbook data: %w", err)
	}

	if count != int64(len(orderbooks)) {
		return fmt.Errorf("expected to insert %d rows, inserted %d", len(orderbooks), count)
	}

	return nil
}

// orderbookLevelsToJSON converts orderbook levels to JSON
func orderbookLevelsToJSON(levels []*pb.Level) ([]byte, error) {
	type JSONLevel struct {
		Price  float64 `json:"price"`
		Volume int64   `json:"volume"`
	}

	jsonLevels := make([]JSONLevel, len(levels))
	for i, level := range levels {
		jsonLevels[i] = JSONLevel{
			Price:  level.Price,
			Volume: level.Quantity,
		}
	}

	return json.Marshal(jsonLevels)
}

// Close closes the writer and connection pool
func (tw *TimescaleWriter) Close() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.closed {
		return nil
	}

	tw.closed = true
	tw.pool.Close()
	return nil
}

// HealthCheck verifies database connection
func (tw *TimescaleWriter) HealthCheck(ctx context.Context) error {
	tw.mu.RLock()
	defer tw.mu.RUnlock()

	if tw.closed {
		return ErrWriterClosed
	}

	return tw.pool.Ping(ctx)
}

// Metrics returns writer metrics
func (tw *TimescaleWriter) Metrics() WriterMetrics {
	return WriterMetrics{
		TotalInserts: tw.totalInserts.Load(),
		TotalErrors:  tw.totalErrors.Load(),
		LastInsertAt: time.Unix(tw.lastInsertAt.Load(), 0),
	}
}

// WriterMetrics contains writer statistics
type WriterMetrics struct {
	TotalInserts int64
	TotalErrors  int64
	LastInsertAt time.Time
}
