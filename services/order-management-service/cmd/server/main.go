package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/jjongkwann/aipx/services/order-management-service/internal/config"
	"github.com/jjongkwann/aipx/services/order-management-service/internal/repository"
	sharedLogger "github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	loggerCfg := &sharedLogger.Config{
		Level:        cfg.Logger.Level,
		Format:       cfg.Logger.Format,
		Output:       cfg.Logger.Output,
		ServiceName:  cfg.Logger.ServiceName,
		Environment:  cfg.Logger.Environment,
		EnableCaller: cfg.Logger.EnableCaller,
	}

	if err := sharedLogger.InitLogger(loggerCfg); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info().
		Str("service", cfg.Logger.ServiceName).
		Str("environment", cfg.Logger.Environment).
		Msg("Starting Order Management Service")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection pool
	dbPool, err := initDatabase(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer dbPool.Close()

	log.Info().Msg("Database connection pool initialized")

	// Initialize repository
	_ = repository.NewPostgresOrderRepository(dbPool)
	log.Info().Msg("Order repository initialized")

	// TODO (T4): Initialize Redis client
	// redisClient := redis.NewClient(&redis.Options{
	//     Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
	//     Password: cfg.Redis.Password,
	//     DB:       cfg.Redis.DB,
	// })
	// defer redisClient.Close()

	// TODO (T4): Initialize Kafka producer
	// kafkaProducer, err := kafka.NewProducer(&kafka.Config{
	//     Brokers: cfg.Kafka.Brokers,
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to initialize kafka producer: %w", err)
	// }
	// defer kafkaProducer.Close()

	// TODO (T4): Initialize Risk Engine
	// riskEngine := risk.NewEngine(&risk.Config{
	//     MaxPositionSize:        cfg.Risk.MaxPositionSize,
	//     MaxDailyLossPercent:    cfg.Risk.MaxDailyLossPercent,
	//     MaxLossPerTradePercent: cfg.Risk.MaxLossPerTradePercent,
	//     MaxOrdersPerMinute:     cfg.Risk.MaxOrdersPerMinute,
	//     MaxOrdersPerDay:        cfg.Risk.MaxOrdersPerDay,
	// })

	// TODO (T4): Initialize Rate Limiter
	// rateLimiter := ratelimit.NewLimiter(redisClient, &ratelimit.Config{
	//     RequestsPerMinute: cfg.Risk.MaxOrdersPerMinute,
	// })

	// TODO (T4): Initialize KIS Broker Client
	// kisClient := broker.NewKISClient(&broker.Config{
	//     BaseURL:      cfg.KIS.BaseURL,
	//     AppKey:       cfg.KIS.AppKey,
	//     AppSecret:    cfg.KIS.AppSecret,
	//     AccountNo:    cfg.KIS.AccountNo,
	//     Timeout:      cfg.KIS.Timeout,
	//     MaxRetries:   cfg.KIS.MaxRetries,
	//     RetryDelay:   cfg.KIS.RetryDelay,
	//     RateLimitRPS: cfg.KIS.RateLimitRPS,
	// })

	// TODO (T4): Initialize gRPC server
	// grpcServer, err := grpc.NewServer(&grpc.Config{
	//     Port:           cfg.Server.GRPCPort,
	//     OrderRepo:      orderRepo,
	//     RiskEngine:     riskEngine,
	//     RateLimiter:    rateLimiter,
	//     KISClient:      kisClient,
	//     KafkaProducer:  kafkaProducer,
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to create grpc server: %w", err)
	// }

	// TODO (T4): Start gRPC server in goroutine
	// go func() {
	//     log.Info().Str("port", cfg.Server.GRPCPort).Msg("Starting gRPC server")
	//     if err := grpcServer.Start(); err != nil {
	//         log.Error().Err(err).Msg("gRPC server error")
	//         cancel()
	//     }
	// }()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Info().Msg("Context cancelled, shutting down")
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	}

	// Graceful shutdown
	log.Info().Msg("Starting graceful shutdown")

	// Create shutdown context with timeout
	_, shutdownCancel := context.WithTimeout(
		context.Background(),
		cfg.Server.ShutdownTimeout,
	)
	defer shutdownCancel()

	// TODO (T4): Stop gRPC server
	// if err := grpcServer.GracefulStop(shutdownCtx); err != nil {
	//     log.Error().Err(err).Msg("Error during gRPC server shutdown")
	// }

	// Close database connections
	dbPool.Close()
	log.Info().Msg("Database connections closed")

	log.Info().Msg("Shutdown complete")
	return nil
}

// initDatabase initializes the PostgreSQL connection pool
func initDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	// Build connection string
	connString := cfg.GetDatabaseDSN()

	// Parse connection string and create pool config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.MinConnections)
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Database.Host).
		Str("port", cfg.Database.Port).
		Str("database", cfg.Database.Database).
		Int("max_conns", cfg.Database.MaxConnections).
		Msg("Database connection successful")

	return pool, nil
}
