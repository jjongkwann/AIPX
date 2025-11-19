package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the user service
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Encryption EncryptionConfig
	Logger     LoggerConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Environment     string // "development", "staging", "production"
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Issuer        string
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	MasterKey         string // Base64 encoded 256-bit key for API key encryption
	PreviousMasterKey string // For key rotation support
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json", "text"
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Server:     loadServerConfig(),
		Database:   loadDatabaseConfig(),
		JWT:        loadJWTConfig(),
		Encryption: loadEncryptionConfig(),
		Logger:     loadLoggerConfig(),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// loadServerConfig loads server configuration from environment variables
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:            getEnv("SERVER_HOST", "0.0.0.0"),
		Port:            getEnvInt("SERVER_PORT", 8080),
		ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
		Environment:     getEnv("ENVIRONMENT", "development"),
	}
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", ""),
		Database:        getEnv("DB_NAME", "aipx_users"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConns:        getEnvInt("DB_MAX_CONNS", 25),
		MinConns:        getEnvInt("DB_MIN_CONNS", 5),
		MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
		MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 10*time.Minute),
	}
}

// loadJWTConfig loads JWT configuration from environment variables
func loadJWTConfig() JWTConfig {
	return JWTConfig{
		AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
		RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		AccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		Issuer:        getEnv("JWT_ISSUER", "aipx-user-service"),
	}
}

// loadEncryptionConfig loads encryption configuration from environment variables
func loadEncryptionConfig() EncryptionConfig {
	return EncryptionConfig{
		MasterKey:         getEnv("ENCRYPTION_MASTER_KEY", ""),
		PreviousMasterKey: getEnv("ENCRYPTION_PREVIOUS_KEY", ""),
	}
}

// loadLoggerConfig loads logger configuration from environment variables
func loadLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate JWT config
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT access secret is required")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT refresh secret is required")
	}
	if len(c.JWT.AccessSecret) < 32 {
		return fmt.Errorf("JWT access secret must be at least 32 characters")
	}
	if len(c.JWT.RefreshSecret) < 32 {
		return fmt.Errorf("JWT refresh secret must be at least 32 characters")
	}

	// Validate encryption config
	if c.Encryption.MasterKey == "" {
		return fmt.Errorf("encryption master key is required")
	}

	// Validate logger config
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logger.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logger.Format)
	}

	return nil
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Database,
		c.SSLMode,
	)
}

// IsProduction returns true if running in production environment
func (c *ServerConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *ServerConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt retrieves an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvDuration retrieves a duration environment variable or returns a default value
// Expects duration in format like "10s", "5m", "1h"
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
