package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jjongkwann/aipx/services/data-ingestion-service/internal/config"
	"github.com/jjongkwann/aipx/services/data-ingestion-service/internal/kis"
	"github.com/jjongkwann/aipx/services/data-ingestion-service/internal/producer"
	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
	"github.com/jjongkwann/aipx/shared/go/pkg/redis"
	"github.com/rs/zerolog/log"
)

// Service represents the data ingestion service
type Service struct {
	config        *config.Config
	redisClient   *redis.Client
	redisCache    *redis.Cache
	authManager   *kis.AuthManager
	wsClient      *kis.Client
	producer      *producer.MarketProducer
	parser        *kis.MessageParser
	shutdownChan  chan struct{}
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Logger); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Info().
		Str("service", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Msg("starting data ingestion service")

	// Create service
	svc, err := NewService(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create service")
	}

	// Start service
	if err := svc.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start service")
	}

	// Wait for shutdown signal
	svc.WaitForShutdown()

	log.Info().Msg("data ingestion service stopped")
}

// NewService creates a new service instance
func NewService(cfg *config.Config) (*Service, error) {
	// Initialize Redis client
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	log.Info().Str("addr", cfg.Redis.Addr).Msg("connected to redis")

	// Create Redis cache
	redisCache := redis.NewCache(redisClient)

	// Initialize authentication manager
	authManager := kis.NewAuthManager(&cfg.KIS, redisCache)

	// Initialize message parser
	parser := kis.NewMessageParser()

	// Initialize Kafka producer
	kafkaProducer, err := producer.NewMarketProducer(cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// Create service
	svc := &Service{
		config:       cfg,
		redisClient:  redisClient,
		redisCache:   redisCache,
		authManager:  authManager,
		producer:     kafkaProducer,
		parser:       parser,
		shutdownChan: make(chan struct{}),
	}

	// Create WebSocket client with message handler
	svc.wsClient = kis.NewClient(&cfg.KIS, authManager, svc.handleMessage)

	return svc, nil
}

// Start starts the service
func (s *Service) Start() error {
	log.Info().Msg("starting service components")

	// Start authentication manager
	if err := s.authManager.Start(); err != nil {
		return fmt.Errorf("failed to start auth manager: %w", err)
	}

	// Start WebSocket client
	if err := s.wsClient.Start(); err != nil {
		return fmt.Errorf("failed to start websocket client: %w", err)
	}

	// Subscribe to market data (example symbols)
	// In production, this should be configurable
	symbols := []string{"005930", "000660", "035420"} // Samsung, SK Hynix, NAVER
	if err := s.wsClient.Subscribe(symbols, "H0STCNT0"); err != nil {
		log.Warn().Err(err).Msg("failed to subscribe to symbols")
	}

	log.Info().Msg("service started successfully")

	return nil
}

// Stop gracefully stops the service
func (s *Service) Stop() error {
	log.Info().Msg("stopping service")

	// Stop WebSocket client
	if s.wsClient != nil {
		if err := s.wsClient.Stop(); err != nil {
			log.Error().Err(err).Msg("error stopping websocket client")
		}
	}

	// Stop authentication manager
	if s.authManager != nil {
		s.authManager.Stop()
	}

	// Close Kafka producer
	if s.producer != nil {
		if err := s.producer.Close(); err != nil {
			log.Error().Err(err).Msg("error closing kafka producer")
		}
	}

	// Close Redis client
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("error closing redis client")
		}
	}

	log.Info().Msg("service stopped")

	return nil
}

// WaitForShutdown waits for shutdown signal
func (s *Service) WaitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("received shutdown signal")
	case <-s.shutdownChan:
		log.Info().Msg("received shutdown request")
	}

	// Perform graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		if err := s.Stop(); err != nil {
			log.Error().Err(err).Msg("error during shutdown")
		}
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("graceful shutdown completed")
	case <-shutdownCtx.Done():
		log.Warn().Msg("shutdown timeout exceeded, forcing exit")
	}
}

// handleMessage processes incoming WebSocket messages
func (s *Service) handleMessage(ctx context.Context, msgType int, data []byte) error {
	if msgType != websocket.TextMessage {
		return nil
	}

	// Parse message
	parsed, err := s.parser.Parse(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse message")
		return err
	}

	// Handle based on message type
	switch parsed.Type {
	case kis.MessageTypeTick:
		if parsed.TickData == nil {
			return fmt.Errorf("nil tick data")
		}

		// Validate tick data
		if err := kis.ValidateTickData(parsed.TickData); err != nil {
			log.Warn().Err(err).Msg("invalid tick data")
			return err
		}

		// Publish to Kafka
		if err := s.producer.PublishTickData(ctx, parsed.TickData); err != nil {
			log.Error().
				Err(err).
				Str("symbol", parsed.TickData.Symbol).
				Msg("failed to publish tick data")
			return err
		}

	case kis.MessageTypeOrderBook:
		if parsed.OrderBook == nil {
			return fmt.Errorf("nil order book")
		}

		// Validate order book
		if err := kis.ValidateOrderBook(parsed.OrderBook); err != nil {
			log.Warn().Err(err).Msg("invalid order book")
			return err
		}

		// Publish to Kafka
		if err := s.producer.PublishOrderBook(ctx, parsed.OrderBook); err != nil {
			log.Error().
				Err(err).
				Str("symbol", parsed.OrderBook.Symbol).
				Msg("failed to publish order book")
			return err
		}

	case kis.MessageTypeError:
		log.Error().
			Err(parsed.Error).
			Msg("received error message")

	case kis.MessageTypeUnknown:
		log.Debug().Msg("received unknown message type")

	default:
		log.Warn().
			Str("type", string(parsed.Type)).
			Msg("unhandled message type")
	}

	return nil
}

// GetMetrics returns service metrics
func (s *Service) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Producer metrics
	if s.producer != nil {
		metrics["producer"] = s.producer.GetMetrics()
	}

	// WebSocket status
	if s.wsClient != nil {
		metrics["websocket_status"] = s.wsClient.GetStatus()
	}

	return metrics
}
