package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/AIPX/services/notification-service/internal/channels"
	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests with real external services
// These tests require environment variables to be set

func TestE2E_Slack(t *testing.T) {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("SLACK_WEBHOOK_URL not set, skipping E2E test")
	}

	log := logger.New("test", logger.InfoLevel, false)
	config := &channels.SlackConfig{
		ChannelConfig:     channels.DefaultChannelConfig(),
		DefaultWebhookURL: webhookURL,
	}

	channel, err := channels.NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := channels.NewNotification(
		"e2e-test-user",
		"test_notification",
		"E2E Test - Slack Channel",
		"This is an end-to-end test notification from the automated test suite",
	)
	notification.Data["test_id"] = "E2E-SLACK-001"
	notification.Data["timestamp"] = "2024-01-01T00:00:00Z"
	notification.Priority = channels.PriorityNormal

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.NoError(t, err)

	t.Log("Successfully sent E2E test notification to Slack")
}

func TestE2E_Telegram(t *testing.T) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" || chatID == "" {
		t.Skip("TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set, skipping E2E test")
	}

	log := logger.New("test", logger.InfoLevel, false)
	config := &channels.TelegramConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		BotToken:      botToken,
		ParseMode:     "Markdown",
	}

	channel, err := channels.NewTelegramChannel(config, log)
	require.NoError(t, err)

	notification := channels.NewNotification(
		"e2e-test-user",
		"test_notification",
		"E2E Test - Telegram Channel",
		"This is an end-to-end test notification from the automated test suite",
	)
	notification.Data["telegram_chat_id"] = chatID
	notification.Data["test_id"] = "E2E-TELEGRAM-001"
	notification.Priority = channels.PriorityNormal

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.NoError(t, err)

	t.Log("Successfully sent E2E test notification to Telegram")
}

func TestE2E_Email(t *testing.T) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUser := os.Getenv("SMTP_USERNAME")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	toEmail := os.Getenv("TEST_EMAIL_RECIPIENT")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" || fromEmail == "" || toEmail == "" {
		t.Skip("SMTP credentials not set, skipping E2E test")
	}

	log := logger.New("test", logger.InfoLevel, false)
	config := &channels.EmailConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		SMTPHost:      smtpHost,
		SMTPPort:      587,
		SMTPUsername:  smtpUser,
		SMTPPassword:  smtpPass,
		FromEmail:     fromEmail,
		FromName:      "AIPX Test Suite",
		UseTLS:        false,
		UseStartTLS:   true,
	}

	channel, err := channels.NewEmailChannel(config, log)
	require.NoError(t, err)

	notification := channels.NewNotification(
		"e2e-test-user",
		"test_notification",
		"E2E Test - Email Channel",
		"This is an end-to-end test notification from the automated test suite",
	)
	notification.Data["email"] = toEmail
	notification.Data["test_id"] = "E2E-EMAIL-001"
	notification.Priority = channels.PriorityNormal

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.NoError(t, err)

	t.Log("Successfully sent E2E test notification via Email")
}

func TestE2E_AllChannels(t *testing.T) {
	// Check if all credentials are set
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	smtpHost := os.Getenv("SMTP_HOST")
	toEmail := os.Getenv("TEST_EMAIL_RECIPIENT")

	if slackWebhook == "" || telegramToken == "" || telegramChatID == "" || smtpHost == "" || toEmail == "" {
		t.Skip("Not all channel credentials set, skipping multi-channel E2E test")
	}

	log := logger.New("test", logger.InfoLevel, false)

	// Create Slack channel
	slackConfig := &channels.SlackConfig{
		ChannelConfig:     channels.DefaultChannelConfig(),
		DefaultWebhookURL: slackWebhook,
	}
	slackChannel, err := channels.NewSlackChannel(slackConfig, log)
	require.NoError(t, err)

	// Create Telegram channel
	telegramConfig := &channels.TelegramConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		BotToken:      telegramToken,
		ParseMode:     "Markdown",
	}
	telegramChannel, err := channels.NewTelegramChannel(telegramConfig, log)
	require.NoError(t, err)

	// Create Email channel
	emailConfig := &channels.EmailConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		SMTPHost:      smtpHost,
		SMTPPort:      587,
		SMTPUsername:  os.Getenv("SMTP_USERNAME"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		FromEmail:     os.Getenv("SMTP_FROM_EMAIL"),
		FromName:      "AIPX Test Suite",
		UseStartTLS:   true,
	}
	emailChannel, err := channels.NewEmailChannel(emailConfig, log)
	require.NoError(t, err)

	// Create channel manager
	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack":    slackChannel,
			"telegram": telegramChannel,
			"email":    emailChannel,
		},
		Logger: log,
	})

	// Send critical notification to all channels
	notification := channels.NewNotification(
		"e2e-test-user",
		"risk_alert",
		"E2E Test - Multi-Channel Critical Alert",
		"This is a critical alert sent to all notification channels",
	)
	notification.Data["telegram_chat_id"] = telegramChatID
	notification.Data["email"] = toEmail
	notification.Data["test_id"] = "E2E-MULTI-001"
	notification.Priority = channels.PriorityCritical

	ctx := context.Background()
	err = manager.Send(ctx, notification)
	assert.NoError(t, err)

	// Verify metrics
	metrics := manager.GetMetrics()
	assert.Equal(t, int64(3), metrics["total_sent"], "Should have sent to all 3 channels")

	t.Log("Successfully sent E2E test notification to all channels")
}

func TestE2E_HealthChecks(t *testing.T) {
	log := logger.New("test", logger.InfoLevel, false)

	// Test Slack health check
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")
	if slackWebhook != "" {
		config := &channels.SlackConfig{
			ChannelConfig:     channels.DefaultChannelConfig(),
			DefaultWebhookURL: slackWebhook,
		}
		channel, _ := channels.NewSlackChannel(config, log)
		ctx := context.Background()
		err := channel.HealthCheck(ctx)
		assert.NoError(t, err, "Slack health check should pass")
		t.Log("Slack health check passed")
	}

	// Test Telegram health check
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken != "" {
		config := &channels.TelegramConfig{
			ChannelConfig: channels.DefaultChannelConfig(),
			BotToken:      telegramToken,
		}
		channel, _ := channels.NewTelegramChannel(config, log)
		ctx := context.Background()
		err := channel.HealthCheck(ctx)
		assert.NoError(t, err, "Telegram health check should pass")
		t.Log("Telegram health check passed")
	}

	// Test Email health check
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost != "" {
		config := &channels.EmailConfig{
			ChannelConfig: channels.DefaultChannelConfig(),
			SMTPHost:      smtpHost,
			SMTPPort:      587,
			FromEmail:     os.Getenv("SMTP_FROM_EMAIL"),
			SMTPUsername:  os.Getenv("SMTP_USERNAME"),
			SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		}
		channel, _ := channels.NewEmailChannel(config, log)
		ctx := context.Background()
		err := channel.HealthCheck(ctx)
		assert.NoError(t, err, "Email health check should pass")
		t.Log("Email health check passed")
	}
}
