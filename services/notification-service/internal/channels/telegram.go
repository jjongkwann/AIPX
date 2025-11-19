package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

const (
	telegramAPIURL = "https://api.telegram.org/bot%s/%s"
)

// TelegramConfig holds Telegram-specific configuration
type TelegramConfig struct {
	*ChannelConfig
	BotToken      string
	ParseMode     string // "Markdown", "MarkdownV2", or "HTML"
	WebhookURL    string
	WebhookSecret string
}

// TelegramChannel implements NotificationChannel for Telegram
type TelegramChannel struct {
	config     *TelegramConfig
	logger     *logger.Logger
	httpClient *http.Client
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
}

// TelegramResponse represents Telegram API response
type TelegramResponse struct {
	Ok          bool                   `json:"ok"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ErrorCode   int                    `json:"error_code,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// NewTelegramChannel creates a new Telegram notification channel
func NewTelegramChannel(config *TelegramConfig, logger *logger.Logger) (*TelegramChannel, error) {
	if config == nil {
		config = &TelegramConfig{
			ChannelConfig: DefaultChannelConfig(),
			ParseMode:     "Markdown",
		}
	}

	channel := &TelegramChannel{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	if err := channel.Validate(); err != nil {
		return nil, fmt.Errorf("invalid telegram config: %w", err)
	}

	return channel, nil
}

// Send sends a notification to Telegram
func (t *TelegramChannel) Send(ctx context.Context, notification *Notification) error {
	// Get chat IDs from notification data
	chatIDs := t.getChatIDs(notification)
	if len(chatIDs) == 0 {
		return fmt.Errorf("%w: no chat IDs configured", ErrChannelNotConfigured)
	}

	// Format message
	messageText := t.formatTelegramMessage(notification)

	// Send to each chat ID with retry logic
	var lastErr error
	successCount := 0

	for _, chatID := range chatIDs {
		if err := t.sendWithRetry(ctx, chatID, messageText, notification); err != nil {
			lastErr = err
			t.logger.Error().
				Err(err).
				Str("chat_id", chatID).
				Msg("Failed to send Telegram notification")
			continue
		}
		successCount++
	}

	// If all sends failed, return error
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send to any chat: %w", lastErr)
	}

	t.logger.Info().
		Str("channel", "telegram").
		Str("user_id", notification.UserID).
		Str("event_type", notification.EventType).
		Int("success_count", successCount).
		Int("total_chats", len(chatIDs)).
		Msg("Telegram notification sent")

	return nil
}

// sendWithRetry sends message with exponential backoff retry
func (t *TelegramChannel) sendWithRetry(ctx context.Context, chatID, messageText string, notification *Notification) error {
	var lastErr error
	maxRetries := t.config.RetryAttempts
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

			t.logger.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Str("chat_id", chatID).
				Msg("Retrying Telegram notification")
		}

		// Send message
		if err := t.sendMessage(ctx, chatID, messageText); err != nil {
			lastErr = err
			t.logger.Error().
				Err(err).
				Int("attempt", attempt+1).
				Str("chat_id", chatID).
				Msg("Failed to send Telegram message")
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

// sendMessage sends a message via Telegram Bot API
func (t *TelegramChannel) sendMessage(ctx context.Context, chatID, messageText string) error {
	// Create message payload
	msg := &TelegramMessage{
		ChatID:                chatID,
		Text:                  messageText,
		ParseMode:             t.config.ParseMode,
		DisableWebPagePreview: true,
		DisableNotification:   false,
	}

	// Marshal to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	// Build API URL
	apiURL := fmt.Sprintf(telegramAPIURL, t.config.BotToken, "sendMessage")

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp TelegramResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if successful
	if !apiResp.Ok {
		return fmt.Errorf("%w: telegram API error (code: %d): %s",
			ErrSendFailed, apiResp.ErrorCode, apiResp.Description)
	}

	return nil
}

// formatTelegramMessage formats notification for Telegram
func (t *TelegramChannel) formatTelegramMessage(notification *Notification) string {
	var sb strings.Builder

	// Add emoji and title
	emoji := t.getEmojiForPriority(notification.Priority)
	sb.WriteString(fmt.Sprintf("%s *%s*\n\n", emoji, t.escapeTelegramMarkdown(notification.Title)))

	// Add message
	sb.WriteString(t.escapeTelegramMarkdown(notification.Message))
	sb.WriteString("\n\n")

	// Add metadata fields
	if len(notification.Data) > 0 {
		fields := t.extractImportantFields(notification.Data)
		if len(fields) > 0 {
			sb.WriteString("📊 *Details:*\n")
			for key, value := range fields {
				sb.WriteString(fmt.Sprintf("• %s: `%v`\n",
					formatFieldName(key), value))
			}
			sb.WriteString("\n")
		}
	}

	// Add footer
	sb.WriteString(fmt.Sprintf("_Event: %s • Time: %s_",
		notification.EventType,
		notification.Timestamp.Format("2006-01-02 15:04:05")))

	return sb.String()
}

// extractImportantFields extracts important fields from data
func (t *TelegramChannel) extractImportantFields(data map[string]interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	importantKeys := []string{
		"order_id", "symbol", "side", "quantity", "price",
		"reason", "status", "position_id", "profit_loss",
	}

	for _, key := range importantKeys {
		if value, ok := data[key]; ok {
			fields[key] = value
		}
	}

	return fields
}

// escapeTelegramMarkdown escapes special characters for Telegram Markdown
func (t *TelegramChannel) escapeTelegramMarkdown(text string) string {
	// For Markdown parse mode, escape these characters
	if t.config.ParseMode == "Markdown" {
		replacer := strings.NewReplacer(
			"_", "\\_",
			"*", "\\*",
			"[", "\\[",
			"`", "\\`",
		)
		return replacer.Replace(text)
	}

	// For MarkdownV2, escape more characters
	if t.config.ParseMode == "MarkdownV2" {
		replacer := strings.NewReplacer(
			"_", "\\_",
			"*", "\\*",
			"[", "\\[",
			"]", "\\]",
			"(", "\\(",
			")", "\\)",
			"~", "\\~",
			"`", "\\`",
			">", "\\>",
			"#", "\\#",
			"+", "\\+",
			"-", "\\-",
			"=", "\\=",
			"|", "\\|",
			"{", "\\{",
			"}", "\\}",
			".", "\\.",
			"!", "\\!",
		)
		return replacer.Replace(text)
	}

	// For HTML, escape HTML entities
	if t.config.ParseMode == "HTML" {
		replacer := strings.NewReplacer(
			"&", "&amp;",
			"<", "&lt;",
			">", "&gt;",
		)
		return replacer.Replace(text)
	}

	return text
}

// getEmojiForPriority returns emoji for priority
func (t *TelegramChannel) getEmojiForPriority(priority Priority) string {
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

// getChatIDs gets chat IDs from notification data
func (t *TelegramChannel) getChatIDs(notification *Notification) []string {
	// Try to get from notification data first
	if chatID, ok := notification.Data["telegram_chat_id"].(string); ok && chatID != "" {
		return []string{chatID}
	}

	// Try to get array of chat IDs
	if chatIDs, ok := notification.Data["telegram_chat_ids"].([]string); ok && len(chatIDs) > 0 {
		return chatIDs
	}

	// Try to get from interface slice
	if chatIDsInterface, ok := notification.Data["telegram_chat_ids"].([]interface{}); ok {
		var chatIDs []string
		for _, id := range chatIDsInterface {
			if strID, ok := id.(string); ok {
				chatIDs = append(chatIDs, strID)
			}
		}
		if len(chatIDs) > 0 {
			return chatIDs
		}
	}

	return nil
}

// Name returns the channel name
func (t *TelegramChannel) Name() string {
	return "telegram"
}

// Validate validates the Telegram channel configuration
func (t *TelegramChannel) Validate() error {
	if t.config == nil {
		return ErrChannelNotConfigured
	}

	if !t.config.Enabled {
		return nil
	}

	// Bot token is required
	if t.config.BotToken == "" {
		return fmt.Errorf("%w: bot token required", ErrInvalidConfig)
	}

	// Validate parse mode
	if t.config.ParseMode != "" {
		validModes := map[string]bool{
			"Markdown":   true,
			"MarkdownV2": true,
			"HTML":       true,
		}
		if !validModes[t.config.ParseMode] {
			return fmt.Errorf("%w: invalid parse mode", ErrInvalidConfig)
		}
	}

	return nil
}

// HealthCheck checks if Telegram Bot API is accessible
func (t *TelegramChannel) HealthCheck(ctx context.Context) error {
	if !t.config.Enabled {
		return fmt.Errorf("telegram channel is disabled")
	}

	if t.config.BotToken == "" {
		return fmt.Errorf("bot token not configured")
	}

	// Call getMe endpoint to verify bot token
	apiURL := fmt.Sprintf(telegramAPIURL, t.config.BotToken, "getMe")

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram health check failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp TelegramResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Ok {
		return fmt.Errorf("telegram health check failed: %s", apiResp.Description)
	}

	t.logger.Debug().
		Interface("bot_info", apiResp.Result).
		Msg("Telegram health check passed")

	return nil
}
