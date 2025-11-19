package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds notification service configuration
type Config struct {
	Environment string

	// Kafka
	Kafka KafkaConfig

	// Database
	Database DatabaseConfig

	// Notification Channels
	Slack    SlackConfig
	Telegram TelegramConfig
	Email    EmailConfig

	// Templates
	Templates TemplatesConfig

	// Logger
	Logger LoggerConfig

	// Server
	Server ServerConfig
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers              []string
	GroupID              string
	TopicTradeOrders     string
	TopicRiskAlerts      string
	TopicSystemAlerts    string
	EnableAutoCommit     bool
	AutoCommitInterval   time.Duration
	SessionTimeout       time.Duration
	HeartbeatInterval    time.Duration
	MaxProcessingTime    time.Duration
	InitialOffset        string // "newest" or "oldest"
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxConnections  int
	MinConnections  int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	SSLMode         string
}

// SlackConfig holds Slack channel configuration
type SlackConfig struct {
	Enabled           bool
	DefaultWebhookURL string
	BotToken          string
	SigningSecret     string
	Timeout           time.Duration
	MaxRetries        int
	RateLimitPerMin   int
}

// TelegramConfig holds Telegram channel configuration
type TelegramConfig struct {
	Enabled         bool
	BotToken        string
	ParseMode       string
	Timeout         time.Duration
	MaxRetries      int
	RateLimitPerMin int
}

// EmailConfig holds email channel configuration
type EmailConfig struct {
	Enabled      bool
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
	UseStartTLS  bool
	Timeout      time.Duration
	MaxRetries   int
}

// TemplatesConfig holds template configuration
type TemplatesConfig struct {
	Directory    string
	UseEmbedded  bool
	ReloadOnDev  bool
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level       string
	Format      string
	Output      string
	ServiceName string
	Environment string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	HTTPPort          string
	MetricsPort       string
	EnableMetrics     bool
	EnableHealthCheck bool
	GracefulTimeout   time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),

		Kafka: KafkaConfig{
			Brokers:              getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID:              getEnv("KAFKA_GROUP_ID", "notification-service"),
			TopicTradeOrders:     getEnv("KAFKA_TOPIC_TRADE_ORDERS", "trade.orders"),
			TopicRiskAlerts:      getEnv("KAFKA_TOPIC_RISK_ALERTS", "risk.alerts"),
			TopicSystemAlerts:    getEnv("KAFKA_TOPIC_SYSTEM_ALERTS", "system.alerts"),
			EnableAutoCommit:     getEnvAsBool("KAFKA_ENABLE_AUTO_COMMIT", true),
			AutoCommitInterval:   getEnvAsDuration("KAFKA_AUTO_COMMIT_INTERVAL", 1*time.Second),
			SessionTimeout:       getEnvAsDuration("KAFKA_SESSION_TIMEOUT", 30*time.Second),
			HeartbeatInterval:    getEnvAsDuration("KAFKA_HEARTBEAT_INTERVAL", 3*time.Second),
			MaxProcessingTime:    getEnvAsDuration("KAFKA_MAX_PROCESSING_TIME", 5*time.Minute),
			InitialOffset:        getEnv("KAFKA_INITIAL_OFFSET", "newest"),
		},

		Database: DatabaseConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnvAsInt("POSTGRES_PORT", 5432),
			User:            getEnv("POSTGRES_USER", "aipx"),
			Password:        getEnv("POSTGRES_PASSWORD", "aipx_dev_password"),
			Database:        getEnv("POSTGRES_DB", "aipx"),
			MaxConnections:  getEnvAsInt("POSTGRES_MAX_CONNECTIONS", 20),
			MinConnections:  getEnvAsInt("POSTGRES_MIN_CONNECTIONS", 5),
			MaxConnLifetime: getEnvAsDuration("POSTGRES_MAX_CONN_LIFETIME", 1*time.Hour),
			MaxConnIdleTime: getEnvAsDuration("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute),
			SSLMode:         getEnv("POSTGRES_SSL_MODE", "disable"),
		},

		Slack: SlackConfig{
			Enabled:           getEnvAsBool("SLACK_ENABLED", true),
			DefaultWebhookURL: getEnv("SLACK_WEBHOOK_URL", ""),
			BotToken:          getEnv("SLACK_BOT_TOKEN", ""),
			SigningSecret:     getEnv("SLACK_SIGNING_SECRET", ""),
			Timeout:           getEnvAsDuration("SLACK_TIMEOUT", 10*time.Second),
			MaxRetries:        getEnvAsInt("SLACK_MAX_RETRIES", 3),
			RateLimitPerMin:   getEnvAsInt("SLACK_RATE_LIMIT_PER_MIN", 60),
		},

		Telegram: TelegramConfig{
			Enabled:         getEnvAsBool("TELEGRAM_ENABLED", true),
			BotToken:        getEnv("TELEGRAM_BOT_TOKEN", ""),
			ParseMode:       getEnv("TELEGRAM_PARSE_MODE", "Markdown"),
			Timeout:         getEnvAsDuration("TELEGRAM_TIMEOUT", 10*time.Second),
			MaxRetries:      getEnvAsInt("TELEGRAM_MAX_RETRIES", 3),
			RateLimitPerMin: getEnvAsInt("TELEGRAM_RATE_LIMIT_PER_MIN", 30),
		},

		Email: EmailConfig{
			Enabled:      getEnvAsBool("EMAIL_ENABLED", true),
			SMTPHost:     getEnv("SMTP_HOST", "localhost"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("EMAIL_FROM_EMAIL", "noreply@aipx.io"),
			FromName:     getEnv("EMAIL_FROM_NAME", "AIPX Trading Platform"),
			UseTLS:       getEnvAsBool("SMTP_USE_TLS", false),
			UseStartTLS:  getEnvAsBool("SMTP_USE_STARTTLS", true),
			Timeout:      getEnvAsDuration("EMAIL_TIMEOUT", 30*time.Second),
			MaxRetries:   getEnvAsInt("EMAIL_MAX_RETRIES", 3),
		},

		Templates: TemplatesConfig{
			Directory:   getEnv("TEMPLATES_DIR", ""),
			UseEmbedded: getEnvAsBool("TEMPLATES_USE_EMBEDDED", true),
			ReloadOnDev: getEnvAsBool("TEMPLATES_RELOAD_ON_DEV", true),
		},

		Logger: LoggerConfig{
			Level:       getEnv("LOG_LEVEL", "info"),
			Format:      getEnv("LOG_FORMAT", "json"),
			Output:      getEnv("LOG_OUTPUT", "stdout"),
			ServiceName: "notification-service",
			Environment: getEnv("ENVIRONMENT", "development"),
		},

		Server: ServerConfig{
			HTTPPort:          getEnv("HTTP_PORT", "8085"),
			MetricsPort:       getEnv("METRICS_PORT", "9095"),
			EnableMetrics:     getEnvAsBool("ENABLE_METRICS", true),
			EnableHealthCheck: getEnvAsBool("ENABLE_HEALTH_CHECK", true),
			GracefulTimeout:   getEnvAsDuration("GRACEFUL_TIMEOUT", 30*time.Second),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Kafka
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}

	// Validate Database
	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}

	// Validate at least one channel is enabled
	if !c.Slack.Enabled && !c.Telegram.Enabled && !c.Email.Enabled {
		return fmt.Errorf("at least one notification channel must be enabled")
	}

	return nil
}

// GetDatabaseURL returns the PostgreSQL connection URL
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// Helper functions

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
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

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

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

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
