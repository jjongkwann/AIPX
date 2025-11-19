package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AIPX/services/notification-service/internal/channels"
	"github.com/AIPX/services/notification-service/internal/repository"
	"github.com/AIPX/services/notification-service/internal/templates"
	"github.com/jjongkwann/aipx/shared/go/pkg/kafka"
	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

// Topics that notification service subscribes to
const (
	TopicTradeOrders  = "trade.orders"
	TopicRiskAlerts   = "risk.alerts"
	TopicSystemAlerts = "system.alerts"
)

// Event types for notifications
const (
	EventOrderFilled      = "order_filled"
	EventOrderRejected    = "order_rejected"
	EventOrderCancelled   = "order_cancelled"
	EventRiskAlert        = "risk_alert"
	EventPositionOpened   = "position_opened"
	EventPositionClosed   = "position_closed"
	EventSystemAlert      = "system_alert"
	EventMaintenanceNotice = "maintenance_notice"
)

// NotificationConsumer consumes Kafka messages and sends notifications
type NotificationConsumer struct {
	consumer      *kafka.Consumer
	channels      map[string]channels.NotificationChannel
	repo          repository.NotificationRepository
	templateEng   *templates.TemplateEngine
	logger        *logger.Logger
	topics        []string
	processedMsgs int64
	failedMsgs    int64
}

// NotificationConsumerConfig holds consumer configuration
type NotificationConsumerConfig struct {
	KafkaConfig   *kafka.ConsumerConfig
	Channels      map[string]channels.NotificationChannel
	Repository    repository.NotificationRepository
	TemplateEngine *templates.TemplateEngine
	Logger        *logger.Logger
}

// NewNotificationConsumer creates a new notification consumer
func NewNotificationConsumer(config *NotificationConsumerConfig) (*NotificationConsumer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	nc := &NotificationConsumer{
		channels:    config.Channels,
		repo:        config.Repository,
		templateEng: config.TemplateEngine,
		logger:      config.Logger,
		topics:      config.KafkaConfig.Topics,
	}

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(config.KafkaConfig, nc.handleMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	nc.consumer = consumer

	// Set metrics handler
	consumer.SetMessageProcessedHandler(nc.onMessageProcessed)

	return nc, nil
}

// Start starts consuming messages
func (nc *NotificationConsumer) Start(ctx context.Context) error {
	nc.logger.Info().
		Strs("topics", nc.topics).
		Msg("Starting notification consumer")

	return nc.consumer.Start(ctx)
}

// Close closes the consumer gracefully
func (nc *NotificationConsumer) Close() error {
	nc.logger.Info().
		Int64("processed", nc.processedMsgs).
		Int64("failed", nc.failedMsgs).
		Msg("Closing notification consumer")

	return nc.consumer.Close()
}

// Topics returns the list of subscribed topics
func (nc *NotificationConsumer) Topics() []string {
	return nc.topics
}

// handleMessage processes a consumed Kafka message
func (nc *NotificationConsumer) handleMessage(ctx context.Context, msg *kafka.ConsumedMessage) error {
	nc.logger.Debug().
		Str("topic", msg.Topic).
		Int32("partition", msg.Partition).
		Int64("offset", msg.Offset).
		Msg("Processing notification message")

	// Parse event data
	var eventData map[string]interface{}
	if err := json.Unmarshal(msg.Value, &eventData); err != nil {
		nc.logger.Error().Err(err).Msg("Failed to parse event data")
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	// Extract event type
	eventType, ok := eventData["event_type"].(string)
	if !ok {
		nc.logger.Error().Msg("Missing or invalid event_type")
		return fmt.Errorf("missing or invalid event_type")
	}

	// Extract user ID
	userID, ok := eventData["user_id"].(string)
	if !ok {
		nc.logger.Error().Msg("Missing or invalid user_id")
		return fmt.Errorf("missing or invalid user_id")
	}

	// Get user's notification preferences
	preferences, err := nc.repo.GetUserPreferences(ctx, userID)
	if err != nil {
		nc.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user preferences")
		return fmt.Errorf("failed to get user preferences: %w", err)
	}

	// If no preferences found, skip notification
	if len(preferences) == 0 {
		nc.logger.Debug().Str("user_id", userID).Msg("No notification preferences found for user")
		return nil
	}

	// Render notification template
	title, message, err := nc.renderNotification(eventType, eventData)
	if err != nil {
		nc.logger.Error().Err(err).Str("event_type", eventType).Msg("Failed to render notification")
		return fmt.Errorf("failed to render notification: %w", err)
	}

	// Send notification to each enabled channel
	for _, pref := range preferences {
		if !pref.Enabled {
			continue
		}

		channel, ok := nc.channels[pref.Channel]
		if !ok {
			nc.logger.Warn().Str("channel", pref.Channel).Msg("Channel not available")
			continue
		}

		// Create notification
		notification := channels.NewNotification(userID, eventType, title, message)
		notification.Data = eventData

		// Send notification
		if err := nc.sendWithRetry(ctx, channel, notification, pref); err != nil {
			nc.logger.Error().
				Err(err).
				Str("user_id", userID).
				Str("channel", pref.Channel).
				Msg("Failed to send notification")

			// Save failed notification to history
			nc.saveHistory(ctx, userID, pref.Channel, eventType, title, message, eventData, "failed", err.Error())
			continue
		}

		// Save successful notification to history
		nc.saveHistory(ctx, userID, pref.Channel, eventType, title, message, eventData, "sent", "")
	}

	return nil
}

// sendWithRetry sends notification with retry logic
func (nc *NotificationConsumer) sendWithRetry(
	ctx context.Context,
	channel channels.NotificationChannel,
	notification *channels.Notification,
	pref *repository.Preference,
) error {
	var lastErr error

	for attempt := 0; attempt <= notification.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(notification.RetryInterval):
			}
		}

		// Create channel-specific context with timeout
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Merge channel config into notification data
		if pref.Config != nil {
			for key, value := range pref.Config {
				notification.Data[key] = value
			}
		}

		// Send notification
		if err := channel.Send(sendCtx, notification); err != nil {
			lastErr = err
			nc.logger.Warn().
				Err(err).
				Str("channel", channel.Name()).
				Int("attempt", attempt+1).
				Msg("Notification send attempt failed")
			continue
		}

		// Success
		nc.logger.Info().
			Str("channel", channel.Name()).
			Str("user_id", notification.UserID).
			Str("event_type", notification.EventType).
			Msg("Notification sent successfully")
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", notification.MaxRetries+1, lastErr)
}

// renderNotification renders notification template
func (nc *NotificationConsumer) renderNotification(eventType string, data map[string]interface{}) (string, string, error) {
	// Determine template name from event type
	templateName := eventType

	// Render template
	rendered, err := nc.templateEng.Render(templateName, data)
	if err != nil {
		return "", "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	// Extract title and message from rendered template
	// Format: "TITLE\n---\nMESSAGE"
	title, message := nc.parseRenderedTemplate(rendered)

	return title, message, nil
}

// parseRenderedTemplate parses rendered template into title and message
func (nc *NotificationConsumer) parseRenderedTemplate(rendered string) (string, string) {
	// Simple parser: first line is title, rest is message
	lines := []byte(rendered)
	for i, b := range lines {
		if b == '\n' {
			return string(lines[:i]), string(lines[i+1:])
		}
	}
	return rendered, rendered
}

// saveHistory saves notification to history
func (nc *NotificationConsumer) saveHistory(
	ctx context.Context,
	userID, channel, eventType, title, message string,
	payload map[string]interface{},
	status, errorMsg string,
) {
	history := &repository.History{
		UserID:       userID,
		Channel:      channel,
		EventType:    eventType,
		Title:        title,
		Message:      message,
		Payload:      payload,
		Status:       status,
		ErrorMessage: errorMsg,
	}

	if status == "sent" {
		now := time.Now()
		history.SentAt = &now
	}

	if err := nc.repo.SaveHistory(ctx, history); err != nil {
		nc.logger.Error().Err(err).Msg("Failed to save notification history")
	}
}

// onMessageProcessed is called when a message is processed
func (nc *NotificationConsumer) onMessageProcessed(topic string, partition int32, offset int64, err error) {
	if err != nil {
		nc.failedMsgs++
		nc.logger.Error().
			Err(err).
			Str("topic", topic).
			Int32("partition", partition).
			Int64("offset", offset).
			Msg("Message processing failed")
	} else {
		nc.processedMsgs++
	}
}

// GetMetrics returns consumer metrics
func (nc *NotificationConsumer) GetMetrics() map[string]int64 {
	return map[string]int64{
		"processed": nc.processedMsgs,
		"failed":    nc.failedMsgs,
	}
}
