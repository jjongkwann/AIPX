package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"shared/pkg/kafka"
	"shared/pkg/logger"
	"shared/pkg/redis"
)

// Config holds all configuration for the data ingestion service
type Config struct {
	// Service configuration
	ServiceName string
	Environment string
	LogLevel    string

	// KIS API configuration
	KIS KISConfig

	// Kafka configuration
	Kafka *kafka.ProducerConfig

	// Redis configuration
	Redis *redis.Config

	// Logger configuration
	Logger *logger.Config
}

// KISConfig holds KIS API configuration
type KISConfig struct {
	APIKey       string
	APISecret    string
	WebSocketURL string
	APIURL       string

	// Connection settings
	ReconnectRetries   int
	ReconnectDelay     time.Duration
	HeartbeatInterval  time.Duration
	MessageBufferSize  int
	TokenCacheTTL      time.Duration
	TokenRefreshBefore time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServiceName: getEnv("SERVICE_NAME", "data-ingestion-service"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	// Load KIS configuration
	cfg.KIS = KISConfig{
		APIKey:             getEnvRequired("KIS_API_KEY"),
		APISecret:          getEnvRequired("KIS_API_SECRET"),
		WebSocketURL:       getEnv("KIS_WEBSOCKET_URL", "ws://ops.koreainvestment.com:21000"),
		APIURL:             getEnv("KIS_API_URL", "https://openapi.koreainvestment.com:9443"),
		ReconnectRetries:   getEnvAsInt("KIS_RECONNECT_RETRIES", 5),
		ReconnectDelay:     getEnvAsDuration("KIS_RECONNECT_DELAY", 5*time.Second),
		HeartbeatInterval:  getEnvAsDuration("KIS_HEARTBEAT_INTERVAL", 30*time.Second),
		MessageBufferSize:  getEnvAsInt("KIS_MESSAGE_BUFFER_SIZE", 1000),
		TokenCacheTTL:      getEnvAsDuration("KIS_TOKEN_CACHE_TTL", 24*time.Hour),
		TokenRefreshBefore: getEnvAsDuration("KIS_TOKEN_REFRESH_BEFORE", 10*time.Minute),
	}

	// Load Kafka configuration
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaConfig := kafka.DefaultProducerConfig()
	kafkaConfig.Brokers = brokers
	kafkaConfig.ClientID = cfg.ServiceName
	cfg.Kafka = kafkaConfig

	// Load Redis configuration
	redisAddr := fmt.Sprintf("%s:%s",
		getEnv("REDIS_HOST", "localhost"),
		getEnv("REDIS_PORT", "6379"),
	)
	redisConfig := redis.DefaultConfig(redisAddr)
	redisConfig.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis = redisConfig

	// Load Logger configuration
	loggerConfig := &logger.Config{
		Level:        cfg.LogLevel,
		Format:       getEnv("LOG_FORMAT", "json"),
		Output:       getEnv("LOG_OUTPUT", "stdout"),
		ServiceName:  cfg.ServiceName,
		Environment:  cfg.Environment,
		EnableCaller: getEnvAsBool("LOG_ENABLE_CALLER", true),
	}
	cfg.Logger = loggerConfig

	return cfg, cfg.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate KIS configuration
	if c.KIS.APIKey == "" {
		return fmt.Errorf("KIS_API_KEY is required")
	}
	if c.KIS.APISecret == "" {
		return fmt.Errorf("KIS_API_SECRET is required")
	}
	if c.KIS.WebSocketURL == "" {
		return fmt.Errorf("KIS_WEBSOCKET_URL is required")
	}
	if c.KIS.APIURL == "" {
		return fmt.Errorf("KIS_API_URL is required")
	}

	// Validate Kafka configuration
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}

	// Validate Redis configuration
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("invalid redis config: %w", err)
	}

	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired retrieves a required environment variable
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

// getEnvAsInt retrieves an environment variable as an integer
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsDuration retrieves an environment variable as a duration
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
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

// getEnvAsBool retrieves an environment variable as a boolean
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}
