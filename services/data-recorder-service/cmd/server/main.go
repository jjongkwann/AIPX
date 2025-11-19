package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/buffer"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/config"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/consumer"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/pkg/logger"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/writer"
)

// Service represents the data recorder service
type Service struct {
	config *config.Config

	// Writers
	timescaleWriter *writer.TimescaleWriter
	s3Writer        *writer.S3Writer

	// Buffers
	tickBuffer      *buffer.BatchBuffer
	orderbookBuffer *buffer.BatchBuffer
	s3TickBuffer    *buffer.BatchBuffer
	s3OrderbookBuffer *buffer.BatchBuffer

	// Consumers
	consumer *consumer.MarketConsumer

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logConfig := &logger.Config{
		Level:        cfg.LogLevel,
		Format:       cfg.LogFormat,
		Output:       "stdout",
		ServiceName:  cfg.ServiceName,
		Environment:  cfg.Environment,
		EnableCaller: true,
	}

	if err := logger.InitLogger(logConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Info().
		Str("service", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Msg("Starting data recorder service")

	// Create service
	svc, err := NewService(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create service")
	}

	// Start service
	if err := svc.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start service")
	}

	// Start health check server
	if cfg.HealthCheckEnabled {
		go startHealthCheckServer(svc, cfg.HealthCheckPort)
	}

	// Wait for shutdown signal
	svc.WaitForShutdown()

	log.Info().Msg("Data recorder service stopped")
}

// NewService creates a new service instance
func NewService(cfg *config.Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &Service{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize TimescaleDB writer
	tsConfig := writer.DefaultTimescaleConfig(cfg.TimescaleDSN())
	tsConfig.MaxConns = cfg.TimescaleMaxConns
	tsConfig.MaxRetries = cfg.MaxRetries
	tsConfig.RetryBackoff = cfg.RetryBackoff

	tsWriter, err := writer.NewTimescaleWriter(ctx, tsConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create timescale writer: %w", err)
	}
	svc.timescaleWriter = tsWriter

	log.Info().Msg("TimescaleDB writer initialized")

	// Initialize S3 writer (if enabled)
	if cfg.S3WriteEnabled {
		s3Config := writer.DefaultS3Config(cfg.S3Bucket, cfg.S3Region)
		s3Config.AccessKeyID = cfg.AWSAccessKeyID
		s3Config.SecretAccessKey = cfg.AWSSecretKey
		s3Config.PartitionPrefix = cfg.S3PartitionPrefix
		s3Config.MaxRetries = cfg.MaxRetries
		s3Config.RetryBackoff = cfg.RetryBackoff

		s3Writer, err := writer.NewS3Writer(ctx, s3Config)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create s3 writer: %w", err)
		}
		svc.s3Writer = s3Writer

		log.Info().
			Str("bucket", cfg.S3Bucket).
			Str("region", cfg.S3Region).
			Msg("S3 writer initialized")
	}

	// Create TimescaleDB buffers (hot path)
	svc.tickBuffer = buffer.NewBatchBuffer(
		cfg.BatchSize,
		cfg.BatchTimeout,
		func(messages []buffer.BufferedMessage) error {
			return tsWriter.WriteMessages(ctx, messages)
		},
	)

	svc.orderbookBuffer = buffer.NewBatchBuffer(
		cfg.BatchSize,
		cfg.BatchTimeout,
		func(messages []buffer.BufferedMessage) error {
			return tsWriter.WriteMessages(ctx, messages)
		},
	)

	// Create S3 buffers (cold path) if enabled
	if cfg.S3WriteEnabled && svc.s3Writer != nil {
		svc.s3TickBuffer = buffer.NewBatchBuffer(
			cfg.S3BatchSize,
			cfg.S3BatchTimeout,
			func(messages []buffer.BufferedMessage) error {
				return svc.s3Writer.WriteMessages(ctx, messages)
			},
		)

		svc.s3OrderbookBuffer = buffer.NewBatchBuffer(
			cfg.S3BatchSize,
			cfg.S3BatchTimeout,
			func(messages []buffer.BufferedMessage) error {
				return svc.s3Writer.WriteMessages(ctx, messages)
			},
		)
	}

	// Create consumer
	consumerConfig := &consumer.ConsumerConfig{
		Brokers:           cfg.KafkaBrokers,
		GroupID:           cfg.KafkaGroupID,
		Topics:            cfg.KafkaTopics,
		OffsetOldest:      cfg.KafkaOffsetOld,
		SessionTimeout:    30 * time.Second,
		HeartbeatInterval: 3 * time.Second,
	}

	marketConsumer, err := consumer.NewMarketConsumer(
		consumerConfig,
		svc.tickBuffer,
		svc.orderbookBuffer,
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}
	svc.consumer = marketConsumer

	log.Info().
		Strs("topics", cfg.KafkaTopics).
		Str("group_id", cfg.KafkaGroupID).
		Msg("Kafka consumer initialized")

	return svc, nil
}

// Start starts all service components
func (svc *Service) Start() error {
	// Start TimescaleDB buffer flushers
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.tickBuffer.Start(svc.ctx)
	}()

	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.orderbookBuffer.Start(svc.ctx)
	}()

	// Start S3 buffer flushers if enabled
	if svc.config.S3WriteEnabled && svc.s3TickBuffer != nil {
		svc.wg.Add(1)
		go func() {
			defer svc.wg.Done()
			svc.s3TickBuffer.Start(svc.ctx)
		}()

		svc.wg.Add(1)
		go func() {
			defer svc.wg.Done()
			svc.s3OrderbookBuffer.Start(svc.ctx)
		}()
	}

	// Start consumer
	if err := svc.consumer.Start(svc.ctx); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start metrics reporter
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		svc.reportMetrics()
	}()

	log.Info().Msg("All components started successfully")
	return nil
}

// WaitForShutdown waits for shutdown signal and performs graceful shutdown
func (svc *Service) WaitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Info().Msg("Shutdown signal received")

	// Start shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), svc.config.ShutdownTimeout)
	defer shutdownCancel()

	svc.Shutdown(shutdownCtx)
}

// Shutdown performs graceful shutdown
func (svc *Service) Shutdown(ctx context.Context) {
	log.Info().Msg("Starting graceful shutdown")

	// Stop consumer first to prevent new messages
	if err := svc.consumer.Stop(); err != nil {
		log.Error().Err(err).Msg("Error stopping consumer")
	}

	// Cancel context to stop all goroutines
	svc.cancel()

	// Flush all buffers
	log.Info().Msg("Flushing buffers...")
	svc.tickBuffer.Flush()
	svc.orderbookBuffer.Flush()

	if svc.s3TickBuffer != nil {
		svc.s3TickBuffer.Flush()
	}
	if svc.s3OrderbookBuffer != nil {
		svc.s3OrderbookBuffer.Flush()
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All goroutines finished")
	case <-ctx.Done():
		log.Warn().Msg("Shutdown timeout reached")
	}

	// Close writers
	if svc.timescaleWriter != nil {
		if err := svc.timescaleWriter.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing TimescaleDB writer")
		}
	}

	if svc.s3Writer != nil {
		if err := svc.s3Writer.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing S3 writer")
		}
	}

	log.Info().Msg("Graceful shutdown completed")
}

// reportMetrics periodically reports service metrics
func (svc *Service) reportMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-svc.ctx.Done():
			return
		case <-ticker.C:
			svc.logMetrics()
		}
	}
}

// logMetrics logs current service metrics
func (svc *Service) logMetrics() {
	// Consumer metrics
	consumerMetrics := svc.consumer.Metrics()
	log.Info().
		Int64("total_messages", consumerMetrics.TotalMessages).
		Int64("total_errors", consumerMetrics.TotalErrors).
		Int64("tick_count", consumerMetrics.TickCount).
		Int64("orderbook_count", consumerMetrics.OrderBookCount).
		Time("last_message_at", consumerMetrics.LastMessageAt).
		Msg("Consumer metrics")

	// Buffer metrics
	tickBufferMetrics := svc.tickBuffer.Metrics()
	orderbookBufferMetrics := svc.orderbookBuffer.Metrics()
	log.Info().
		Int("tick_buffer_size", tickBufferMetrics.CurrentSize).
		Int64("tick_buffer_flushes", tickBufferMetrics.TotalFlushes).
		Int("orderbook_buffer_size", orderbookBufferMetrics.CurrentSize).
		Int64("orderbook_buffer_flushes", orderbookBufferMetrics.TotalFlushes).
		Msg("Buffer metrics")

	// TimescaleDB metrics
	tsMetrics := svc.timescaleWriter.Metrics()
	log.Info().
		Int64("timescale_inserts", tsMetrics.TotalInserts).
		Int64("timescale_errors", tsMetrics.TotalErrors).
		Time("last_insert_at", tsMetrics.LastInsertAt).
		Msg("TimescaleDB metrics")

	// S3 metrics (if enabled)
	if svc.s3Writer != nil {
		s3Metrics := svc.s3Writer.Metrics()
		log.Info().
			Int64("s3_uploads", s3Metrics.TotalUploads).
			Int64("s3_errors", s3Metrics.TotalErrors).
			Int64("s3_bytes", s3Metrics.TotalBytes).
			Time("last_upload_at", s3Metrics.LastUploadAt).
			Msg("S3 metrics")
	}
}

// startHealthCheckServer starts HTTP health check server
func startHealthCheckServer(svc *Service, port string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check all components
		healthy := true
		status := make(map[string]string)

		// Check consumer
		if err := svc.consumer.HealthCheck(); err != nil {
			status["consumer"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			status["consumer"] = "healthy"
		}

		// Check TimescaleDB
		if err := svc.timescaleWriter.HealthCheck(r.Context()); err != nil {
			status["timescaledb"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			status["timescaledb"] = "healthy"
		}

		// Check S3
		if svc.s3Writer != nil {
			if err := svc.s3Writer.HealthCheck(r.Context()); err != nil {
				status["s3"] = "unhealthy: " + err.Error()
				healthy = false
			} else {
				status["s3"] = "healthy"
			}
		}

		if healthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK\n")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "UNHEALTHY\n")
		}

		for component, s := range status {
			fmt.Fprintf(w, "%s: %s\n", component, s)
		}
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		consumerMetrics := svc.consumer.Metrics()
		tsMetrics := svc.timescaleWriter.Metrics()
		tickBufferMetrics := svc.tickBuffer.Metrics()
		orderbookBufferMetrics := svc.orderbookBuffer.Metrics()

		fmt.Fprintf(w, "# Consumer Metrics\n")
		fmt.Fprintf(w, "total_messages: %d\n", consumerMetrics.TotalMessages)
		fmt.Fprintf(w, "total_errors: %d\n", consumerMetrics.TotalErrors)
		fmt.Fprintf(w, "tick_count: %d\n", consumerMetrics.TickCount)
		fmt.Fprintf(w, "orderbook_count: %d\n", consumerMetrics.OrderBookCount)

		fmt.Fprintf(w, "\n# Buffer Metrics\n")
		fmt.Fprintf(w, "tick_buffer_size: %d\n", tickBufferMetrics.CurrentSize)
		fmt.Fprintf(w, "orderbook_buffer_size: %d\n", orderbookBufferMetrics.CurrentSize)

		fmt.Fprintf(w, "\n# TimescaleDB Metrics\n")
		fmt.Fprintf(w, "timescale_inserts: %d\n", tsMetrics.TotalInserts)
		fmt.Fprintf(w, "timescale_errors: %d\n", tsMetrics.TotalErrors)

		if svc.s3Writer != nil {
			s3Metrics := svc.s3Writer.Metrics()
			fmt.Fprintf(w, "\n# S3 Metrics\n")
			fmt.Fprintf(w, "s3_uploads: %d\n", s3Metrics.TotalUploads)
			fmt.Fprintf(w, "s3_errors: %d\n", s3Metrics.TotalErrors)
			fmt.Fprintf(w, "s3_bytes: %d\n", s3Metrics.TotalBytes)
		}
	})

	addr := ":" + port
	log.Info().Str("addr", addr).Msg("Starting health check server")

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("Health check server error")
	}
}
