package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AIPX/services/notification-service/internal/channels"
	"github.com/AIPX/services/notification-service/internal/config"
	"github.com/AIPX/services/notification-service/internal/consumer"
	"github.com/AIPX/services/notification-service/internal/repository"
	"github.com/AIPX/services/notification-service/internal/templates"
	"github.com/AIPX/shared/go/pkg/kafka"
	"github.com/AIPX/shared/go/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewLogger(&logger.Config{
		Level:        cfg.Logger.Level,
		Format:       cfg.Logger.Format,
		Output:       cfg.Logger.Output,
		ServiceName:  cfg.Logger.ServiceName,
		Environment:  cfg.Logger.Environment,
		EnableCaller: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Info().
		Str("environment", cfg.Environment).
		Str("version", getVersion()).
		Msg("Starting notification service")

	// Initialize database connection
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	log.Info().Msg("Database connection established")

	// Initialize repository
	repo := repository.NewPostgresRepository(db, log)

	// Initialize template engine
	templateEngine, err := templates.NewTemplateEngine(log, cfg.Templates.Directory)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize template engine")
	}

	log.Info().
		Strs("templates", templateEngine.ListTemplates()).
		Msg("Template engine initialized")

	// Initialize notification channels
	notifChannels, err := initChannels(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize notification channels")
	}

	log.Info().
		Int("count", len(notifChannels)).
		Msg("Notification channels initialized")

	// Initialize Kafka consumer
	kafkaConfig := &kafka.ConsumerConfig{
		Brokers:            cfg.Kafka.Brokers,
		GroupID:            cfg.Kafka.GroupID,
		Topics:             []string{cfg.Kafka.TopicTradeOrders, cfg.Kafka.TopicRiskAlerts, cfg.Kafka.TopicSystemAlerts},
		EnableAutoCommit:   cfg.Kafka.EnableAutoCommit,
		AutoCommitInterval: cfg.Kafka.AutoCommitInterval,
		SessionTimeout:     cfg.Kafka.SessionTimeout,
		HeartbeatInterval:  cfg.Kafka.HeartbeatInterval,
		InitialOffset:      cfg.Kafka.InitialOffset,
	}

	notificationConsumer, err := consumer.NewNotificationConsumer(&consumer.NotificationConsumerConfig{
		KafkaConfig:    kafkaConfig,
		Channels:       notifChannels,
		Repository:     repo,
		TemplateEngine: templateEngine,
		Logger:         log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create notification consumer")
	}

	log.Info().
		Strs("topics", notificationConsumer.Topics()).
		Msg("Notification consumer initialized")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in a goroutine
	consumerErrChan := make(chan error, 1)
	go func() {
		if err := notificationConsumer.Start(ctx); err != nil {
			consumerErrChan <- fmt.Errorf("consumer error: %w", err)
		}
	}()

	log.Info().Msg("Notification service started successfully")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received termination signal")
	case err := <-consumerErrChan:
		log.Error().Err(err).Msg("Consumer error")
	}

	// Graceful shutdown
	log.Info().Msg("Shutting down notification service")

	// Cancel context to stop consumer
	cancel()

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer shutdownCancel()

	// Close consumer
	if err := notificationConsumer.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing consumer")
	}

	// Wait for shutdown to complete or timeout
	<-shutdownCtx.Done()

	// Log final metrics
	metrics := notificationConsumer.GetMetrics()
	log.Info().
		Int64("processed", metrics["processed"]).
		Int64("failed", metrics["failed"]).
		Msg("Final metrics")

	log.Info().Msg("Notification service stopped")
}

// initDatabase initializes the database connection pool
func initDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.MinConnections)
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// initChannels initializes notification channels
func initChannels(cfg *config.Config, log *logger.Logger) (map[string]channels.NotificationChannel, error) {
	notifChannels := make(map[string]channels.NotificationChannel)

	// Initialize Slack channel
	if cfg.Slack.Enabled {
		slackChannel, err := channels.NewSlackChannel(&channels.SlackConfig{
			ChannelConfig: &channels.ChannelConfig{
				Enabled:       cfg.Slack.Enabled,
				Timeout:       cfg.Slack.Timeout,
				RetryAttempts: cfg.Slack.MaxRetries,
				RetryInterval: 5 * time.Second,
				RateLimit: channels.RateLimitConfig{
					Enabled:      true,
					MaxPerMinute: cfg.Slack.RateLimitPerMin,
				},
			},
			DefaultWebhookURL: cfg.Slack.DefaultWebhookURL,
			BotToken:          cfg.Slack.BotToken,
			SigningSecret:     cfg.Slack.SigningSecret,
		}, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create slack channel: %w", err)
		}
		notifChannels["slack"] = slackChannel
		log.Info().Msg("Slack channel initialized")
	}

	// Initialize Telegram channel
	if cfg.Telegram.Enabled {
		telegramChannel, err := channels.NewTelegramChannel(&channels.TelegramConfig{
			ChannelConfig: &channels.ChannelConfig{
				Enabled:       cfg.Telegram.Enabled,
				Timeout:       cfg.Telegram.Timeout,
				RetryAttempts: cfg.Telegram.MaxRetries,
				RetryInterval: 5 * time.Second,
				RateLimit: channels.RateLimitConfig{
					Enabled:      true,
					MaxPerMinute: cfg.Telegram.RateLimitPerMin,
				},
			},
			BotToken:  cfg.Telegram.BotToken,
			ParseMode: cfg.Telegram.ParseMode,
		}, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create telegram channel: %w", err)
		}
		notifChannels["telegram"] = telegramChannel
		log.Info().Msg("Telegram channel initialized")
	}

	// Initialize Email channel
	if cfg.Email.Enabled {
		emailChannel, err := channels.NewEmailChannel(&channels.EmailConfig{
			ChannelConfig: &channels.ChannelConfig{
				Enabled:       cfg.Email.Enabled,
				Timeout:       cfg.Email.Timeout,
				RetryAttempts: cfg.Email.MaxRetries,
				RetryInterval: 5 * time.Second,
			},
			SMTPHost:     cfg.Email.SMTPHost,
			SMTPPort:     cfg.Email.SMTPPort,
			SMTPUsername: cfg.Email.SMTPUsername,
			SMTPPassword: cfg.Email.SMTPPassword,
			FromEmail:    cfg.Email.FromEmail,
			FromName:     cfg.Email.FromName,
			UseTLS:       cfg.Email.UseTLS,
			UseStartTLS:  cfg.Email.UseStartTLS,
		}, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create email channel: %w", err)
		}
		notifChannels["email"] = emailChannel
		log.Info().Msg("Email channel initialized")
	}

	return notifChannels, nil
}

// getVersion returns the service version
func getVersion() string {
	version := os.Getenv("SERVICE_VERSION")
	if version == "" {
		return "dev"
	}
	return version
}
