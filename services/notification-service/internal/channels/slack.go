package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

// SlackConfig holds Slack-specific configuration
type SlackConfig struct {
	*ChannelConfig
	DefaultWebhookURL string
	BotToken          string
	SigningSecret     string
}

// SlackChannel implements NotificationChannel for Slack
type SlackChannel struct {
	config     *SlackConfig
	logger     *logger.Logger
	httpClient *http.Client
}

// SlackMessage represents a Slack message with Block Kit
type SlackMessage struct {
	Text        string                   `json:"text,omitempty"`
	Blocks      []map[string]interface{} `json:"blocks,omitempty"`
	Attachments []SlackAttachment        `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color  string                 `json:"color,omitempty"`
	Title  string                 `json:"title,omitempty"`
	Text   string                 `json:"text,omitempty"`
	Fields []SlackAttachmentField `json:"fields,omitempty"`
	Footer string                 `json:"footer,omitempty"`
	Ts     int64                  `json:"ts,omitempty"`
}

// SlackAttachmentField represents a field in Slack attachment
type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(config *SlackConfig, logger *logger.Logger) (*SlackChannel, error) {
	if config == nil {
		config = &SlackConfig{
			ChannelConfig: DefaultChannelConfig(),
		}
	}

	channel := &SlackChannel{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	if err := channel.Validate(); err != nil {
		return nil, fmt.Errorf("invalid slack config: %w", err)
	}

	return channel, nil
}

// Send sends a notification to Slack
func (s *SlackChannel) Send(ctx context.Context, notification *Notification) error {
	// Get webhook URL from notification data or use default
	webhookURL := s.getWebhookURL(notification)
	if webhookURL == "" {
		return fmt.Errorf("%w: webhook URL not configured", ErrChannelNotConfigured)
	}

	// Format message with Slack blocks
	slackMsg := s.formatSlackMessage(notification)

	// Retry logic with exponential backoff
	var lastErr error
	maxRetries := s.config.RetryAttempts
	if notification.MaxRetries > 0 {
		maxRetries = notification.MaxRetries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			s.logger.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Retrying Slack notification")
		}

		// Send HTTP POST to webhook
		if err := s.sendWebhook(ctx, webhookURL, slackMsg); err != nil {
			lastErr = err
			s.logger.Error().
				Err(err).
				Int("attempt", attempt+1).
				Msg("Failed to send Slack notification")
			continue
		}

		// Success
		s.logger.Info().
			Str("channel", "slack").
			Str("user_id", notification.UserID).
			Str("event_type", notification.EventType).
			Msg("Slack notification sent successfully")
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

// sendWebhook sends the message to Slack webhook
func (s *SlackChannel) sendWebhook(ctx context.Context, webhookURL string, msg *SlackMessage) error {
	// Marshal message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: slack API returned status %d: %s", ErrSendFailed, resp.StatusCode, string(body))
	}

	return nil
}

// formatSlackMessage formats notification into Slack Block Kit format
func (s *SlackChannel) formatSlackMessage(notification *Notification) *SlackMessage {
	// Determine color based on priority
	color := s.getColorForPriority(notification.Priority)

	// Create header block
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": s.getEmojiForPriority(notification.Priority) + " " + notification.Title,
			},
		},
	}

	// Add divider
	blocks = append(blocks, map[string]interface{}{
		"type": "divider",
	})

	// Add message content
	blocks = append(blocks, map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": notification.Message,
		},
	})

	// Add metadata fields if available
	if len(notification.Data) > 0 {
		fields := s.extractFieldsFromData(notification.Data)
		if len(fields) > 0 {
			blocks = append(blocks, map[string]interface{}{
				"type":   "section",
				"fields": fields,
			})
		}
	}

	// Add footer with timestamp
	blocks = append(blocks, map[string]interface{}{
		"type": "context",
		"elements": []map[string]interface{}{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Event:* %s | *Time:* <!date^%d^{date_short_pretty} {time}|%s>",
					notification.EventType,
					notification.Timestamp.Unix(),
					notification.Timestamp.Format(time.RFC3339)),
			},
		},
	})

	return &SlackMessage{
		Text:   notification.Title, // Fallback text
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color: color,
			},
		},
	}
}

// extractFieldsFromData extracts key fields from notification data
func (s *SlackChannel) extractFieldsFromData(data map[string]interface{}) []map[string]interface{} {
	var fields []map[string]interface{}

	// Define important fields to display
	importantFields := []string{
		"order_id", "symbol", "side", "quantity", "price",
		"reason", "status", "position_id", "profit_loss",
	}

	for _, key := range importantFields {
		if value, ok := data[key]; ok {
			fields = append(fields, map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s:*\n%v", formatFieldName(key), value),
			})
		}
	}

	return fields
}

// formatFieldName converts snake_case to Title Case
func formatFieldName(s string) string {
	if s == "" {
		return ""
	}

	var result []rune
	capitalize := true

	for _, r := range s {
		if r == '_' {
			result = append(result, ' ')
			capitalize = true
			continue
		}

		if capitalize {
			result = append(result, []rune(string(r))[0])
			capitalize = false
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

// getColorForPriority returns Slack color for priority
func (s *SlackChannel) getColorForPriority(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "#36a64f" // green
	case PriorityNormal:
		return "#2196F3" // blue
	case PriorityHigh:
		return "#ff9800" // orange
	case PriorityCritical:
		return "#f44336" // red
	default:
		return "#757575" // gray
	}
}

// getEmojiForPriority returns emoji for priority
func (s *SlackChannel) getEmojiForPriority(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "🟢"
	case PriorityNormal:
		return "🔵"
	case PriorityHigh:
		return "⚠️"
	case PriorityCritical:
		return "🚨"
	default:
		return "ℹ️"
	}
}

// getWebhookURL gets webhook URL from notification data or config
func (s *SlackChannel) getWebhookURL(notification *Notification) string {
	// Try to get from notification data first
	if webhookURL, ok := notification.Data["slack_webhook_url"].(string); ok && webhookURL != "" {
		return webhookURL
	}

	// Fall back to default
	return s.config.DefaultWebhookURL
}

// Name returns the channel name
func (s *SlackChannel) Name() string {
	return "slack"
}

// Validate validates the Slack channel configuration
func (s *SlackChannel) Validate() error {
	if s.config == nil {
		return ErrChannelNotConfigured
	}

	if !s.config.Enabled {
		return nil
	}

	// At least one of webhook URL or bot token must be configured
	if s.config.DefaultWebhookURL == "" && s.config.BotToken == "" {
		return fmt.Errorf("%w: webhook URL or bot token required", ErrInvalidConfig)
	}

	return nil
}

// HealthCheck checks if Slack webhook is accessible
func (s *SlackChannel) HealthCheck(ctx context.Context) error {
	if !s.config.Enabled {
		return fmt.Errorf("slack channel is disabled")
	}

	if s.config.DefaultWebhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	// Send a test message (with empty blocks to avoid actual notification)
	testMsg := &SlackMessage{
		Text: "Health check from AIPX Notification Service",
	}

	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to send (this will fail gracefully if webhook is invalid)
	if err := s.sendWebhook(testCtx, s.config.DefaultWebhookURL, testMsg); err != nil {
		return fmt.Errorf("slack health check failed: %w", err)
	}

	s.logger.Debug().Msg("Slack health check passed")
	return nil
}
