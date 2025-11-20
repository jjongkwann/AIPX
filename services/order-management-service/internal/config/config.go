package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	sharedConfig "github.com/jjongkwann/aipx/shared/go/pkg/config"
)

// Config holds the configuration for the Order Management Service
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Kafka     KafkaConfig
	KIS       KISConfig
	Risk      RiskConfig
	RateLimit RateLimitConfig
	Logger    LoggerConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	GRPCPort         string
	HTTPPort         string
	ShutdownTimeout  time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxConnections   int
	EnableReflection bool
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxConnections  int
	MinConnections  int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	SSLMode         string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host            string
	Port            string
	Password        string
	DB              int
	MaxConnections  int
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolSize        int
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers              []string
	TopicOrderEvents     string
	TopicExecutions      string
	TopicTradingSignals  string
	ConsumerGroup        string
	EnableAutoCommit     bool
	SessionTimeout       time.Duration
	HeartbeatInterval    time.Duration
	RebalanceTimeout     time.Duration
	MaxProcessingTime    time.Duration
}

// KISConfig holds Korea Investment & Securities API configuration
type KISConfig struct {
	BaseURL      string
	AppKey       string
	AppSecret    string
	AccountNo    string
	Timeout      time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
	RateLimitRPS int
}

// RiskConfig holds risk management configuration
type RiskConfig struct {
	MaxPositionSize        float64
	MaxDailyLossPercent    float64
	MaxLossPerTradePercent float64
	MaxOrdersPerMinute     int
	MaxOrdersPerDay        int
	DefaultOrderTimeout    time.Duration
	EnableRiskChecks       bool
	MaxOrderValue          float64
	PriceDeviation         float64
	DailyLossLimit         float64
	AllowedSymbols         []string
	DuplicateWindow        int64
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Rate  int // orders per second per user
	Burst int // burst capacity
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level        string
	Format       string
	Output       string
	EnableCaller bool
	ServiceName  string
	Environment  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load shared config first
	sharedCfg := sharedConfig.LoadConfig()

	config := &Config{
		Server: ServerConfig{
			GRPCPort:         getEnv("OMS_GRPC_PORT", sharedCfg.OMSGRPCPort),
			HTTPPort:         getEnv("OMS_HTTP_PORT", sharedCfg.OMSHTTPPort),
			ShutdownTimeout:  getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
			ReadTimeout:      getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:     getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			MaxConnections:   getEnvAsInt("SERVER_MAX_CONNECTIONS", 1000),
			EnableReflection: getEnvAsBool("SERVER_ENABLE_REFLECTION", true),
		},
		Database: DatabaseConfig{
			Host:            getEnv("POSTGRES_HOST", sharedCfg.PostgresHost),
			Port:            getEnv("POSTGRES_PORT", sharedCfg.PostgresPort),
			User:            getEnv("POSTGRES_USER", sharedCfg.PostgresUser),
			Password:        getEnv("POSTGRES_PASSWORD", sharedCfg.PostgresPassword),
			Database:        getEnv("POSTGRES_DB", sharedCfg.PostgresDB),
			MaxConnections:  getEnvAsInt("POSTGRES_MAX_CONNECTIONS", sharedCfg.PostgresMaxConnections),
			MinConnections:  getEnvAsInt("POSTGRES_MIN_CONNECTIONS", 5),
			MaxConnLifetime: getEnvAsDuration("POSTGRES_MAX_CONN_LIFETIME", 1*time.Hour),
			MaxConnIdleTime: getEnvAsDuration("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute),
			SSLMode:         getEnv("POSTGRES_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:           getEnv("REDIS_HOST", sharedCfg.RedisHost),
			Port:           getEnv("REDIS_PORT", sharedCfg.RedisPort),
			Password:       getEnv("REDIS_PASSWORD", sharedCfg.RedisPassword),
			DB:             getEnvAsInt("REDIS_DB", sharedCfg.RedisDB),
			MaxConnections: getEnvAsInt("REDIS_MAX_CONNECTIONS", sharedCfg.RedisMaxConnections),
			ConnectTimeout: getEnvAsDuration("REDIS_CONNECT_TIMEOUT", 5*time.Second),
			ReadTimeout:    getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:   getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:       getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		Kafka: KafkaConfig{
			Brokers:             getEnvAsSlice("KAFKA_BROKERS", sharedCfg.KafkaBrokers),
			TopicOrderEvents:    getEnv("KAFKA_TOPIC_ORDER_EVENTS", sharedCfg.KafkaTopicOrderEvents),
			TopicExecutions:     getEnv("KAFKA_TOPIC_EXECUTIONS", sharedCfg.KafkaTopicExecutions),
			TopicTradingSignals: getEnv("KAFKA_TOPIC_TRADING_SIGNALS", sharedCfg.KafkaTopicTradingSignals),
			ConsumerGroup:       getEnv("KAFKA_CONSUMER_GROUP", "oms-consumer-group"),
			EnableAutoCommit:    getEnvAsBool("KAFKA_ENABLE_AUTO_COMMIT", false),
			SessionTimeout:      getEnvAsDuration("KAFKA_SESSION_TIMEOUT", 10*time.Second),
			HeartbeatInterval:   getEnvAsDuration("KAFKA_HEARTBEAT_INTERVAL", 3*time.Second),
			RebalanceTimeout:    getEnvAsDuration("KAFKA_REBALANCE_TIMEOUT", 60*time.Second),
			MaxProcessingTime:   getEnvAsDuration("KAFKA_MAX_PROCESSING_TIME", 5*time.Minute),
		},
		KIS: KISConfig{
			BaseURL:      getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
			AppKey:       getEnv("KIS_APP_KEY", ""),
			AppSecret:    getEnv("KIS_APP_SECRET", ""),
			AccountNo:    getEnv("KIS_ACCOUNT_NO", ""),
			Timeout:      getEnvAsDuration("KIS_TIMEOUT", 30*time.Second),
			MaxRetries:   getEnvAsInt("KIS_MAX_RETRIES", 3),
			RetryDelay:   getEnvAsDuration("KIS_RETRY_DELAY", 1*time.Second),
			RateLimitRPS: getEnvAsInt("KIS_RATE_LIMIT_RPS", 20),
		},
		Risk: RiskConfig{
			MaxPositionSize:        getEnvAsFloat("MAX_POSITION_SIZE", sharedCfg.MaxPositionSize),
			MaxDailyLossPercent:    getEnvAsFloat("MAX_DAILY_LOSS_PERCENT", sharedCfg.MaxDailyLossPercent),
			MaxLossPerTradePercent: getEnvAsFloat("MAX_LOSS_PER_TRADE_PERCENT", sharedCfg.MaxLossPerTradePercent),
			MaxOrdersPerMinute:     getEnvAsInt("MAX_ORDERS_PER_MINUTE", 60),
			MaxOrdersPerDay:        getEnvAsInt("MAX_ORDERS_PER_DAY", 500),
			DefaultOrderTimeout:    getEnvAsDuration("DEFAULT_ORDER_TIMEOUT", time.Duration(sharedCfg.DefaultOrderTimeout)*time.Second),
			EnableRiskChecks:       getEnvAsBool("ENABLE_RISK_CHECKS", true),
			MaxOrderValue:          getEnvAsFloat("RISK_MAX_ORDER_VALUE", 10000000.0), // 1000만원
			PriceDeviation:         getEnvAsFloat("RISK_PRICE_DEVIATION", 0.05),        // 5%
			DailyLossLimit:         getEnvAsFloat("RISK_DAILY_LOSS_LIMIT", 1000000.0),  // 100만원
			AllowedSymbols:         getEnvAsSlice("RISK_ALLOWED_SYMBOLS", []string{}),
			DuplicateWindow:        int64(getEnvAsInt("RISK_DUPLICATE_WINDOW", 10)), // 10 seconds
		},
		RateLimit: RateLimitConfig{
			Rate:  getEnvAsInt("RATE_LIMIT_RATE", 5),  // 5 orders per second
			Burst: getEnvAsInt("RATE_LIMIT_BURST", 10), // burst of 10
		},
		Logger: LoggerConfig{
			Level:        getEnv("LOG_LEVEL", sharedCfg.LogLevel),
			Format:       getEnv("LOG_FORMAT", sharedCfg.LogFormat),
			Output:       getEnv("LOG_OUTPUT", sharedCfg.LogOutput),
			EnableCaller: getEnvAsBool("LOG_ENABLE_CALLER", true),
			ServiceName:  "order-management-service",
			Environment:  getEnv("ENVIRONMENT", sharedCfg.Environment),
		},
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.GRPCPort == "" {
		return fmt.Errorf("server.grpc_port is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers is required")
	}

	// Note: KIS credentials are optional in dev mode but required for production
	if c.Logger.Environment == "production" {
		if c.KIS.AppKey == "" {
			return fmt.Errorf("kis.app_key is required in production")
		}
		if c.KIS.AppSecret == "" {
			return fmt.Errorf("kis.app_secret is required in production")
		}
	}

	return nil
}

// GetDatabaseDSN returns the PostgreSQL DSN connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
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

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
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
