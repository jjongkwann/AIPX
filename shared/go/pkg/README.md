# AIPX Shared Go Packages

Production-ready shared packages for the AIPX trading platform.

## Packages

### 1. Kafka Package (`pkg/kafka`)

Thread-safe Kafka producer and consumer with protobuf support, retry logic, and graceful shutdown.

**Features:**
- Async producer with configurable retry and backoff
- Consumer group with manual offset control
- Protobuf and JSON serialization
- Context-aware operations
- Health checks
- Metrics hooks

**Usage:**

```go
import (
    "context"
    "github.com/jjongkwann/aipx/shared/go/pkg/kafka"
)

// Producer
producerConfig := kafka.DefaultProducerConfig()
producerConfig.Brokers = []string{"localhost:9092"}

producer, err := kafka.NewProducer(producerConfig)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// Send message
ctx := context.Background()
err = producer.SendJSON(ctx, "my-topic", []byte("key"), []byte(`{"data":"value"}`), nil)

// Send protobuf
var protoMsg pb.YourMessage
err = producer.SendProto(ctx, "my-topic", []byte("key"), &protoMsg, nil)

// Consumer
consumerConfig := kafka.DefaultConsumerConfig("my-group", []string{"my-topic"})
consumerConfig.Brokers = []string{"localhost:9092"}

handler := func(ctx context.Context, msg *kafka.ConsumedMessage) error {
    log.Printf("Received: %s", string(msg.Value))

    // Unmarshal protobuf
    var protoMsg pb.YourMessage
    if err := msg.UnmarshalProto(&protoMsg); err != nil {
        return err
    }

    return nil
}

consumer, err := kafka.NewConsumer(consumerConfig, handler)
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// Start consuming (blocks until context cancelled)
if err := consumer.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 2. Redis Package (`pkg/redis`)

Redis client wrapper with caching, distributed locking, and rate limiting.

**Features:**
- Connection pooling and retry
- Cache operations with TTL
- JSON and Protobuf serialization
- Distributed locks
- Rate limiting
- Health checks

**Usage:**

```go
import (
    "context"
    "time"
    "github.com/jjongkwann/aipx/shared/go/pkg/redis"
)

// Client
config := redis.DefaultConfig("localhost:6379")
client, err := redis.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Basic operations
ctx := context.Background()
err = client.Set(ctx, "key", "value", 5*time.Minute)
value, err := client.Get(ctx, "key")

// Cache with JSON
cache := redis.NewCache(client)

type User struct {
    ID   string
    Name string
}

user := User{ID: "1", Name: "Alice"}
err = cache.SetJSON(ctx, "user:1", user, 10*time.Minute)

var retrievedUser User
err = cache.GetJSON(ctx, "user:1", &retrievedUser)

// Distributed lock
locker := redis.NewLocker(client)
lock, err := locker.Lock(ctx, "my-lock", 30*time.Second)
if err != nil {
    // Lock not obtained
}
defer lock.Release(ctx)

// Critical section
// ...

// Rate limiter
limiter := redis.NewRateLimiter(client, 100, time.Minute) // 100 requests per minute
allowed, err := limiter.Allow(ctx, "user:123")
if !allowed {
    // Rate limit exceeded
}
```

### 3. Logger Package (`pkg/logger`)

Structured logging with zerolog, context propagation, and gRPC middleware.

**Features:**
- JSON and console output
- Log levels (trace, debug, info, warn, error, fatal)
- Context-aware logging
- Request/trace ID propagation
- gRPC interceptors
- Production/development configs

**Usage:**

```go
import (
    "context"
    "github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

// Initialize global logger
config := logger.DevelopmentConfig("my-service")
err := logger.InitLogger(config)

// Create logger instance
loggerInstance, err := logger.NewLogger(config)

// Basic logging
loggerInstance.Info().Msg("Application started")
loggerInstance.Error().Err(err).Msg("Failed to connect")

// Context-aware logging
ctx := context.Background()
ctx = logger.WithRequestID(ctx, "req-123")
ctx = logger.WithUserID(ctx, "user-456")

logger.Info(ctx).Msg("Processing request") // Includes request_id and user_id

// gRPC middleware
import "google.golang.org/grpc"

server := grpc.NewServer(
    grpc.UnaryInterceptor(logger.UnaryServerInterceptor(loggerInstance)),
    grpc.StreamInterceptor(logger.StreamServerInterceptor(loggerInstance)),
)

// Client interceptor
conn, err := grpc.Dial(
    address,
    grpc.WithUnaryInterceptor(logger.UnaryClientInterceptor(loggerInstance)),
)
```

### 4. Config Package (`pkg/config`)

Environment-based configuration with validation.

**Features:**
- Load from environment variables
- .env file support
- Type-safe configuration
- Validation with detailed errors
- Connection string builders

**Usage:**

```go
import (
    "github.com/jjongkwann/aipx/shared/go/pkg/config"
)

// Load .env file (optional)
err := config.LoadEnvFile(".env")

// Load configuration
cfg := config.LoadConfig()

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Access configuration
log.Printf("Environment: %s", cfg.Environment)
log.Printf("Kafka Brokers: %v", cfg.KafkaBrokers)

// Helper methods
if cfg.IsProduction() {
    // Production-specific logic
}

redisAddr := cfg.GetRedisAddr()
pgConnStr := cfg.GetPostgresConnectionString()

// Require specific env vars
err = config.RequireEnvVars("DATABASE_URL", "API_KEY")
```

## Project Structure

```
shared/go/pkg/
├── kafka/
│   ├── config.go      # Configuration structures and defaults
│   ├── producer.go    # Async producer with retry logic
│   ├── consumer.go    # Consumer group implementation
│   └── errors.go      # Custom error types
├── redis/
│   ├── config.go      # Redis configuration
│   ├── client.go      # Redis client wrapper
│   ├── cache.go       # High-level caching operations
│   └── lock.go        # Distributed locking and rate limiting
├── logger/
│   ├── logger.go      # Logger initialization and config
│   ├── context.go     # Context-aware logging
│   └── middleware.go  # gRPC interceptors
└── config/
    ├── config.go      # Application configuration
    ├── env.go         # Environment variable loading
    └── validate.go    # Configuration validation
```

## Best Practices

### Error Handling

All packages use explicit error handling with wrapped errors:

```go
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

### Context Usage

All blocking operations accept `context.Context` for cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := producer.Send(ctx, msg)
```

### Graceful Shutdown

```go
// Setup signal handling
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    log.Info().Msg("Shutting down...")
    cancel()
}()

// Start consumer
if err := consumer.Start(ctx); err != nil {
    log.Error().Err(err).Msg("Consumer error")
}

// Cleanup
consumer.Close()
producer.Close()
client.Close()
```

### Configuration Management

```go
// Development
config.LoadEnvFile(".env.local")
cfg := config.LoadConfig()

// Production
cfg := config.LoadConfig() // Reads from environment
cfg.MustValidate() // Panics if invalid

// Use validation for startup checks
if err := cfg.Validate(); err != nil {
    log.Fatal().Err(err).Msg("Invalid configuration")
}
```

## Thread Safety

All packages are thread-safe:
- Kafka producer/consumer use sync.RWMutex
- Redis client handles concurrent requests
- Logger is safe for concurrent use

## Testing

Run tests:

```bash
cd shared/go
go test ./pkg/...
```

Run tests with coverage:

```bash
go test -cover ./pkg/...
```

## Dependencies

- **Kafka**: IBM/sarama v1.46.3
- **Redis**: go-redis/redis/v8
- **Logger**: rs/zerolog v1.34.0
- **gRPC**: google.golang.org/grpc v1.77.0
- **Protobuf**: google.golang.org/protobuf v1.36.10

## License

MIT License - AIPX Project
