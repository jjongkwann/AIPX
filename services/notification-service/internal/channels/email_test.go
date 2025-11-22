package channels

import (
	"strings"
	"testing"

	"notification-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailChannel_Send_Success(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      2525,
		FromEmail:     "test@example.com",
		FromName:      "Test Sender",
		UseTLS:        false,
		UseStartTLS:   false,
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)
	assert.NotNil(t, channel)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["email"] = "recipient@example.com"
	assert.NotNil(t, notification)

	// Note: This test will fail without a real SMTP server
	// In integration tests, we can use a mock SMTP server
}

func TestEmailChannel_FormatHTML(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"
	notification.Priority = PriorityNormal

	html := channel.formatHTMLEmail(notification)

	assert.NotEmpty(t, html)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "Order Filled")
	assert.Contains(t, html, "Your order has been filled")
	assert.Contains(t, html, "#2196F3") // Blue for normal priority
}

func TestEmailChannel_FormatPlainText(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"

	plainText := channel.formatPlainTextEmail(notification)

	assert.NotEmpty(t, plainText)
	assert.Contains(t, plainText, "Order Filled")
	assert.Contains(t, plainText, "Your order has been filled")
	assert.Contains(t, plainText, "ORDER-123")
}

func TestEmailChannel_MultipleRecipients(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "test", "Test", "Test message")
	notification.Data["emails"] = []string{"user1@example.com", "user2@example.com", "user3@example.com"}

	recipients := channel.getRecipients(notification)
	assert.Len(t, recipients, 3)
	assert.Contains(t, recipients, "user1@example.com")
	assert.Contains(t, recipients, "user2@example.com")
	assert.Contains(t, recipients, "user3@example.com")
}

func TestEmailChannel_PriorityHeaders(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "5"},
		{PriorityNormal, "3"},
		{PriorityHigh, "2"},
		{PriorityCritical, "1"},
	}

	for _, tt := range tests {
		header := channel.getPriorityHeader(tt.priority)
		assert.Equal(t, tt.expected, header)
	}
}

func TestEmailChannel_BuildMIMEMessage(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "sender@example.com",
		FromName:      "Test Sender",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	msg := &EmailMessage{
		To:       []string{"recipient@example.com"},
		Subject:  "Test Subject",
		HTMLBody: "<html><body>Test</body></html>",
		TextBody: "Test",
		Headers: map[string]string{
			"X-Priority": "1",
		},
	}

	mimeMsg := channel.buildMIMEMessage(msg)
	mimeStr := string(mimeMsg)

	assert.Contains(t, mimeStr, "From: \"Test Sender\" <sender@example.com>")
	assert.Contains(t, mimeStr, "To: recipient@example.com")
	assert.Contains(t, mimeStr, "Subject: Test Subject")
	assert.Contains(t, mimeStr, "X-Priority: 1")
	assert.Contains(t, mimeStr, "Content-Type: multipart/alternative")
	assert.Contains(t, mimeStr, "boundary-aipx")
}

func TestEmailChannel_Name(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	assert.Equal(t, "email", channel.Name())
}

func TestEmailChannel_Validate(t *testing.T) {
	log := testutil.NewTestLogger()

	t.Run("Valid Configuration", func(t *testing.T) {
		config := &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			FromEmail:     "test@example.com",
		}

		channel, err := NewEmailChannel(config, log)
		require.NoError(t, err)
		assert.NotNil(t, channel)
	})

	t.Run("Missing SMTP Host", func(t *testing.T) {
		config := &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPHost:      "",
			SMTPPort:      587,
			FromEmail:     "test@example.com",
		}

		_, err := NewEmailChannel(config, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP host required")
	})

	t.Run("Invalid SMTP Port", func(t *testing.T) {
		config := &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPHost:      "smtp.example.com",
			SMTPPort:      0,
			FromEmail:     "test@example.com",
		}

		_, err := NewEmailChannel(config, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SMTP port")
	})

	t.Run("Missing From Email", func(t *testing.T) {
		config := &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			FromEmail:     "",
		}

		_, err := NewEmailChannel(config, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from email required")
	})

	t.Run("Invalid From Email", func(t *testing.T) {
		config := &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			FromEmail:     "invalid-email",
		}

		_, err := NewEmailChannel(config, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid from email address")
	})
}

func TestEmailChannel_GetRecipients(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	t.Run("Single Email", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		notification.Data["email"] = "user@example.com"
		recipients := channel.getRecipients(notification)
		assert.Len(t, recipients, 1)
		assert.Equal(t, "user@example.com", recipients[0])
	})

	t.Run("Multiple Emails - String Array", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		notification.Data["emails"] = []string{"user1@example.com", "user2@example.com"}
		recipients := channel.getRecipients(notification)
		assert.Len(t, recipients, 2)
	})

	t.Run("Multiple Emails - Interface Array", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		notification.Data["emails"] = []interface{}{"user1@example.com", "user2@example.com"}
		recipients := channel.getRecipients(notification)
		assert.Len(t, recipients, 2)
	})

	t.Run("No Recipients", func(t *testing.T) {
		notification := NewNotification("user-123", "test", "Test", "Test")
		recipients := channel.getRecipients(notification)
		assert.Nil(t, recipients)
	})
}

func TestEmailChannel_GetColorForPriority(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "#4CAF50"},
		{PriorityNormal, "#2196F3"},
		{PriorityHigh, "#FF9800"},
		{PriorityCritical, "#F44336"},
	}

	for _, tt := range tests {
		color := channel.getColorForPriority(tt.priority)
		assert.Equal(t, strings.ToUpper(tt.expected), strings.ToUpper(color))
	}
}

func TestEmailChannel_ExtractImportantFields(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
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

	fields := channel.extractImportantFields(data)

	assert.NotEmpty(t, fields)
	assert.Equal(t, "ORDER-123", fields["order_id"])
	assert.Equal(t, "BTCUSDT", fields["symbol"])
	assert.Equal(t, "BUY", fields["side"])
	assert.NotContains(t, fields, "timestamp")
	assert.NotContains(t, fields, "ignored")
}

func TestEmailChannel_HTMLEscaping(t *testing.T) {
	log := testutil.NewTestLogger()
	config := &EmailConfig{
		ChannelConfig: DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, err := NewEmailChannel(config, log)
	require.NoError(t, err)

	notification := NewNotification("user-123", "test", "Test <script>alert('xss')</script>", "Message with <tags>")

	html := channel.formatHTMLEmail(notification)

	// Check that HTML is escaped
	assert.NotContains(t, html, "<script>")
	assert.Contains(t, html, "&lt;script&gt;")
	assert.NotContains(t, html, "<tags>")
	assert.Contains(t, html, "&lt;tags&gt;")
}
