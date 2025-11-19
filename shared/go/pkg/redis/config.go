package redis

import (
	"fmt"
	"time"
)

// Config holds Redis client configuration
type Config struct {
	// Addr is the Redis server address (host:port)
	Addr string

	// Password for authentication (optional)
	Password string

	// DB is the database number to use (0-15)
	DB int

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// DialTimeout is the timeout for establishing new connections
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes
	WriteTimeout time.Duration

	// PoolSize is the maximum number of socket connections
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int

	// MaxConnAge is the connection age at which client retires (closes) the connection
	MaxConnAge time.Duration

	// PoolTimeout is the amount of time client waits for connection if all connections are busy
	PoolTimeout time.Duration
}

// DefaultConfig returns Redis configuration with sensible defaults
func DefaultConfig(addr string) *Config {
	return &Config{
		Addr:         addr,
		Password:     "",
		DB:           0,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxConnAge:   30 * time.Minute,
		PoolTimeout:  4 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("redis address is required")
	}

	if c.DB < 0 || c.DB > 15 {
		return fmt.Errorf("redis DB must be between 0 and 15")
	}

	if c.PoolSize < 1 {
		return fmt.Errorf("pool size must be at least 1")
	}

	if c.DialTimeout <= 0 {
		return fmt.Errorf("dial timeout must be positive")
	}

	return nil
}
