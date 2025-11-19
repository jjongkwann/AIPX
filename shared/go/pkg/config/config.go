package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	Environment string

	// Kafka
	KafkaBrokers              []string
	KafkaTopicMarketData      string
	KafkaTopicTradingSignals  string
	KafkaTopicOrderEvents     string
	KafkaTopicExecutions      string
	KafkaConsumerGroup        string

	// Redis
	RedisHost           string
	RedisPort           string
	RedisPassword       string
	RedisDB             int
	RedisMaxConnections int

	// PostgreSQL
	PostgresHost           string
	PostgresPort           string
	PostgresUser           string
	PostgresPassword       string
	PostgresDB             string
	PostgresMaxConnections int

	// TimescaleDB
	TimescaleHost     string
	TimescalePort     string
	TimescaleUser     string
	TimescalePassword string
	TimescaleDB       string

	// AWS
	AWSRegion            string
	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	S3BucketName         string
	MSKBootstrapServers  string
	ElastiCacheEndpoint  string
	RDSEndpoint          string

	// Service Ports
	DataIngestionPort   string
	DataIngestionWSPort string
	OMSGRPCPort         string
	OMSHTTPPort         string
	CognitiveAPIPort    string

	// JWT
	JWTSecret            string
	JWTExpiration        int
	JWTRefreshExpiration int

	// Trading
	MaxPositionSize        float64
	MaxDailyLossPercent    float64
	MaxLossPerTradePercent float64
	DefaultOrderTimeout    int

	// Feature Flags
	EnablePaperTrading bool
	EnableLiveTrading  bool
	EnableBacktesting  bool

	// Monitoring
	EnablePrometheusMetrics bool
	PrometheusPort          string
	EnableJaegerTracing     bool
	JaegerAgentHost         string
	JaegerAgentPort         string

	// Logging
	LogLevel  string
	LogFormat string
	LogOutput string

	// Development
	EnableHotReload bool
	DebugMode       bool
	DebugPort       string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "dev"),

		// Kafka
		KafkaBrokers:             getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopicMarketData:     getEnv("KAFKA_TOPIC_MARKET_DATA", "market-data-raw"),
		KafkaTopicTradingSignals: getEnv("KAFKA_TOPIC_TRADING_SIGNALS", "trading-signals"),
		KafkaTopicOrderEvents:    getEnv("KAFKA_TOPIC_ORDER_EVENTS", "order-events"),
		KafkaTopicExecutions:     getEnv("KAFKA_TOPIC_EXECUTIONS", "executions"),
		KafkaConsumerGroup:       getEnv("KAFKA_CONSUMER_GROUP", "aipx-consumer-group"),

		// Redis
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvAsInt("REDIS_DB", 0),
		RedisMaxConnections: getEnvAsInt("REDIS_MAX_CONNECTIONS", 10),

		// PostgreSQL
		PostgresHost:           getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:           getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:           getEnv("POSTGRES_USER", "aipx"),
		PostgresPassword:       getEnv("POSTGRES_PASSWORD", "aipx_dev_password"),
		PostgresDB:             getEnv("POSTGRES_DB", "aipx"),
		PostgresMaxConnections: getEnvAsInt("POSTGRES_MAX_CONNECTIONS", 20),

		// TimescaleDB
		TimescaleHost:     getEnv("TIMESCALEDB_HOST", "localhost"),
		TimescalePort:     getEnv("TIMESCALEDB_PORT", "5433"),
		TimescaleUser:     getEnv("TIMESCALEDB_USER", "aipx"),
		TimescalePassword: getEnv("TIMESCALEDB_PASSWORD", "aipx_dev_password"),
		TimescaleDB:       getEnv("TIMESCALEDB_DB", "aipx_timeseries"),

		// AWS
		AWSRegion:           getEnv("AWS_REGION", "ap-northeast-2"),
		AWSAccessKeyID:      getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:  getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3BucketName:        getEnv("S3_BUCKET_NAME", "aipx-data-lake-dev"),
		MSKBootstrapServers: getEnv("MSK_BOOTSTRAP_SERVERS", ""),
		ElastiCacheEndpoint: getEnv("ELASTICACHE_ENDPOINT", ""),
		RDSEndpoint:         getEnv("RDS_ENDPOINT", ""),

		// Service Ports
		DataIngestionPort:   getEnv("DATA_INGESTION_PORT", "8081"),
		DataIngestionWSPort: getEnv("DATA_INGESTION_WS_PORT", "8082"),
		OMSGRPCPort:         getEnv("OMS_GRPC_PORT", "50051"),
		OMSHTTPPort:         getEnv("OMS_HTTP_PORT", "8083"),
		CognitiveAPIPort:    getEnv("COGNITIVE_API_PORT", "8084"),

		// JWT
		JWTSecret:            getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		JWTExpiration:        getEnvAsInt("JWT_EXPIRATION", 3600),
		JWTRefreshExpiration: getEnvAsInt("JWT_REFRESH_EXPIRATION", 604800),

		// Trading
		MaxPositionSize:        getEnvAsFloat("MAX_POSITION_SIZE", 1000000),
		MaxDailyLossPercent:    getEnvAsFloat("MAX_DAILY_LOSS_PERCENT", 5.0),
		MaxLossPerTradePercent: getEnvAsFloat("MAX_LOSS_PER_TRADE_PERCENT", 2.0),
		DefaultOrderTimeout:    getEnvAsInt("DEFAULT_ORDER_TIMEOUT", 30),

		// Feature Flags
		EnablePaperTrading: getEnvAsBool("ENABLE_PAPER_TRADING", true),
		EnableLiveTrading:  getEnvAsBool("ENABLE_LIVE_TRADING", false),
		EnableBacktesting:  getEnvAsBool("ENABLE_BACKTESTING", true),

		// Monitoring
		EnablePrometheusMetrics: getEnvAsBool("ENABLE_PROMETHEUS_METRICS", true),
		PrometheusPort:          getEnv("PROMETHEUS_PORT", "9090"),
		EnableJaegerTracing:     getEnvAsBool("ENABLE_JAEGER_TRACING", false),
		JaegerAgentHost:         getEnv("JAEGER_AGENT_HOST", "localhost"),
		JaegerAgentPort:         getEnv("JAEGER_AGENT_PORT", "6831"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),

		// Development
		EnableHotReload: getEnvAsBool("ENABLE_HOT_RELOAD", true),
		DebugMode:       getEnvAsBool("DEBUG_MODE", false),
		DebugPort:       getEnv("DEBUG_PORT", "2345"),
	}
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

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
