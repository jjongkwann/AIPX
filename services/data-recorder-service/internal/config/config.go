package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the data recorder service
type Config struct {
	// Service
	ServiceName string
	Environment string

	// Kafka
	KafkaBrokers   []string
	KafkaGroupID   string
	KafkaTopics    []string
	KafkaOffsetOld bool // Start from oldest if no offset

	// TimescaleDB
	TimescaleHost     string
	TimescalePort     string
	TimescaleUser     string
	TimescalePassword string
	TimescaleDB       string
	TimescaleMaxConns int

	// S3
	S3Bucket          string
	S3Region          string
	AWSAccessKeyID    string
	AWSSecretKey      string
	S3WriteEnabled    bool
	S3PartitionPrefix string

	// Batch Settings
	BatchSize        int
	BatchTimeout     time.Duration
	S3BatchSize      int           // Messages before S3 write
	S3BatchTimeout   time.Duration // Time before S3 write
	MaxBufferSize    int           // Max messages in buffer
	FlushWorkers     int           // Number of flush workers

	// Performance
	NumConsumers       int // Number of Kafka consumer goroutines
	DBWriteWorkers     int // Number of DB writer workers
	S3WriteWorkers     int // Number of S3 writer workers
	MaxRetries         int
	RetryBackoff       time.Duration
	ShutdownTimeout    time.Duration
	HealthCheckEnabled bool
	HealthCheckPort    string

	// Logging
	LogLevel  string
	LogFormat string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServiceName: getEnv("SERVICE_NAME", "data-recorder-service"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Kafka
		KafkaBrokers:   getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "data-recorder-group"),
		KafkaTopics:    getEnvAsSlice("KAFKA_TOPICS", []string{"market.tick", "market.orderbook"}),
		KafkaOffsetOld: getEnvAsBool("KAFKA_OFFSET_OLD", false),

		// TimescaleDB
		TimescaleHost:     getEnv("TIMESCALEDB_HOST", "localhost"),
		TimescalePort:     getEnv("TIMESCALEDB_PORT", "5433"),
		TimescaleUser:     getEnv("TIMESCALEDB_USER", "aipx"),
		TimescalePassword: getEnv("TIMESCALEDB_PASSWORD", "aipx_dev_password"),
		TimescaleDB:       getEnv("TIMESCALEDB_DB", "aipx_timeseries"),
		TimescaleMaxConns: getEnvAsInt("TIMESCALEDB_MAX_CONNS", 20),

		// S3
		S3Bucket:          getEnv("S3_BUCKET", "aipx-data-lake-dev"),
		S3Region:          getEnv("S3_REGION", "ap-northeast-2"),
		AWSAccessKeyID:    getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:      getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3WriteEnabled:    getEnvAsBool("S3_WRITE_ENABLED", true),
		S3PartitionPrefix: getEnv("S3_PARTITION_PREFIX", "market-data"),

		// Batch Settings
		BatchSize:      getEnvAsInt("BATCH_SIZE", 1000),
		BatchTimeout:   time.Duration(getEnvAsInt("BATCH_TIMEOUT_MS", 5000)) * time.Millisecond,
		S3BatchSize:    getEnvAsInt("S3_BATCH_SIZE", 100000), // 100k messages
		S3BatchTimeout: time.Duration(getEnvAsInt("S3_BATCH_TIMEOUT_SEC", 3600)) * time.Second,
		MaxBufferSize:  getEnvAsInt("MAX_BUFFER_SIZE", 100000),
		FlushWorkers:   getEnvAsInt("FLUSH_WORKERS", 4),

		// Performance
		NumConsumers:       getEnvAsInt("NUM_CONSUMERS", 4),
		DBWriteWorkers:     getEnvAsInt("DB_WRITE_WORKERS", 4),
		S3WriteWorkers:     getEnvAsInt("S3_WRITE_WORKERS", 2),
		MaxRetries:         getEnvAsInt("MAX_RETRIES", 3),
		RetryBackoff:       time.Duration(getEnvAsInt("RETRY_BACKOFF_MS", 1000)) * time.Millisecond,
		ShutdownTimeout:    time.Duration(getEnvAsInt("SHUTDOWN_TIMEOUT_SEC", 30)) * time.Second,
		HealthCheckEnabled: getEnvAsBool("HEALTH_CHECK_ENABLED", true),
		HealthCheckPort:    getEnv("HEALTH_CHECK_PORT", "8080"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}
	if len(c.KafkaTopics) == 0 {
		return fmt.Errorf("kafka topics cannot be empty")
	}
	if c.TimescaleHost == "" {
		return fmt.Errorf("timescaledb host cannot be empty")
	}
	if c.S3WriteEnabled && c.S3Bucket == "" {
		return fmt.Errorf("s3 bucket cannot be empty when s3 write is enabled")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.BatchTimeout <= 0 {
		return fmt.Errorf("batch timeout must be positive")
	}
	return nil
}

// TimescaleDSN returns the PostgreSQL DSN for TimescaleDB
func (c *Config) TimescaleDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.TimescaleUser,
		c.TimescalePassword,
		c.TimescaleHost,
		c.TimescalePort,
		c.TimescaleDB,
	)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
