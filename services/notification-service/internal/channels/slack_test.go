package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AIPX/services/notification-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlackChannel_Send_Success(t *testing.T) {
	// Create mock Slack server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create Slack channel
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: server.URL,
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	// Create notification
	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"

	// Send notification
	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.NoError(t, err)
}

func TestSlackChannel_Send_InvalidWebhook(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "", // No webhook URL
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Test message")

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook URL not configured")
}

func TestSlackChannel_Send_NetworkError(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "http://invalid-webhook-url-that-does-not-exist.test",
	}
	config.Timeout = 1 * time.Second

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Test message")
	notification.MaxRetries = 0 // Don't retry for this test

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.Error(t, err)
}

func TestSlackChannel_Send_Retry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: server.URL,
	}
	config.RetryAttempts = 3

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Test message")

	ctx := context.Background()
	err = channel.Send(ctx, notification)
	assert.NoError(t, err)
	assert.Equal(t, 3, attemptCount, "Should have attempted 3 times")
}

func TestSlackChannel_FormatMessage(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "https://hooks.slack.com/test",
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"
	notification.Data["quantity"] = 1.5
	notification.Priority = PriorityNormal

	msg := channel.formatSlackMessage(notification)

	assert.NotNil(t, msg)
	assert.Equal(t, "Order Filled", msg.Text)
	assert.NotEmpty(t, msg.Blocks)
	assert.Len(t, msg.Attachments, 1)
	assert.Equal(t, "#2196F3", msg.Attachments[0].Color) // Blue for normal priority
}

func TestSlackChannel_Priority_Colors(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "https://hooks.slack.com/test",
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	tests := []struct {
		name     string
		priority Priority
		expected string
	}{
		{"Low Priority", PriorityLow, "#36a64f"},
		{"Normal Priority", PriorityNormal, "#2196F3"},
		{"High Priority", PriorityHigh, "#ff9800"},
		{"Critical Priority", PriorityCritical, "#f44336"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification := NewNotification("user-123", "test", "Test", "Test message")
			notification.Priority = tt.priority

			msg := channel.formatSlackMessage(notification)
			assert.Equal(t, tt.expected, msg.Attachments[0].Color)
		})
	}
}

func TestSlackChannel_HealthCheck(t *testing.T) {
	t.Run("Healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))
		defer server.Close()

		log := testutil.NewTestLogger()
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: server.URL,
		}

		channel, err := NewSlackChannel(config, log)
		require.NoError(t, err)

		ctx := context.Background()
		err = channel.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Unhealthy - No Webhook", func(t *testing.T) {
		log := testutil.NewTestLogger()
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: "",
		}

		channel, err := NewSlackChannel(config, log)
		require.NoError(t, err)

		ctx := context.Background()
		err = channel.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no webhook URL configured")
	})

	t.Run("Unhealthy - Disabled", func(t *testing.T) {
		log := testutil.NewTestLogger()
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: "https://hooks.slack.com/test",
		}
		config.Enabled = false

		channel, err := NewSlackChannel(config, log)
		require.NoError(t, err)

		ctx := context.Background()
		err = channel.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

func TestSlackChannel_Name(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "https://hooks.slack.com/test",
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	assert.Equal(t, "slack", channel.Name())
}

func TestSlackChannel_Validate(t *testing.T) {
	log := testutil.NewTestLogger()

	t.Run("Valid Configuration", func(t *testing.T) {
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: "https://hooks.slack.com/test",
		}

		channel, err := NewSlackChannel(config, log)
		require.NoError(t, err)
		assert.NotNil(t, channel)
	})

	t.Run("Missing Webhook and Token", func(t *testing.T) {
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: "",
			BotToken:          "",
		}

		_, err := NewSlackChannel(config, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook URL or bot token required")
	})

	t.Run("Disabled Channel", func(t *testing.T) {
		config := &SlackConfig{
			ChannelConfig:     DefaultChannelConfig(),
			DefaultWebhookURL: "",
		}
		config.Enabled = false

		channel, err := NewSlackChannel(config, log)
		require.NoError(t, err)
		assert.NotNil(t, channel)
	})
}

func TestSlackChannel_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: server.URL,
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "test", "Test", "Test message")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = channel.Send(ctx, notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestSlackChannel_ExtractFields(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "https://hooks.slack.com/test",
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	data := map[string]interface{}{
		"order_id":  "ORDER-123",
		"symbol":    "BTCUSDT",
		"side":      "BUY",
		"quantity":  1.5,
		"price":     50000.0,
		"timestamp": 1234567890,
		"ignored":   "this should be ignored",
	}

	fields := channel.extractFieldsFromData(data)

	assert.NotEmpty(t, fields)
	assert.GreaterOrEqual(t, len(fields), 5)

	// Check that important fields are included
	hasOrderID := false
	for _, field := range fields {
		if text, ok := field["text"].(string); ok {
			if contains := func(s, substr string) bool {
				return len(s) >= len(substr) && s[:len(substr)] == substr
			}(text, "**order_id:**"); contains {
				hasOrderID = true
				break
			}
		}
	}
	assert.True(t, hasOrderID, "Should include order_id field")
}

func TestSlackChannel_GetEmojiForPriority(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: "https://hooks.slack.com/test",
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "🟢"},
		{PriorityNormal, "🔵"},
		{PriorityHigh, "⚠️"},
		{PriorityCritical, "🚨"},
	}

	for _, tt := range tests {
		emoji := channel.getEmojiForPriority(tt.priority)
		assert.Equal(t, tt.expected, emoji)
	}
}

func TestSlackChannel_GetWebhookURL(t *testing.T) {
	log := testutil.NewTestLogger()
	defaultURL := "https://hooks.slack.com/default"
	config := &SlackConfig{
		ChannelConfig:     DefaultChannelConfig(),
		DefaultWebhookURL: defaultURL,
	}

	channel, err := NewSlackChannel(config, log)
	require.NoError(t, err)

	t.Run("Use Default", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		url := channel.getWebhookURL(notification)
		assert.Equal(t, defaultURL, url)
	})

	t.Run("Use Custom from Data", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		customURL := "https://hooks.slack.com/custom"
		notification.Data["slack_webhook_url"] = customURL
		url := channel.getWebhookURL(notification)
		assert.Equal(t, customURL, url)
	})
}
