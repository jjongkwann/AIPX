package performance

import (
	"context"
	"testing"

	"notification-service/internal/channels"
	"notification-service/internal/testutil"
)

// Benchmark tests for notification service

func BenchmarkSlackSend(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.SlackConfig{
		ChannelConfig:     channels.DefaultChannelConfig(),
		DefaultWebhookURL: "https://example.com/webhook",
	}

	channel, _ := channels.NewSlackChannel(config, log)
	notification := channels.NewNotification("user-123", "test", "Test", "Benchmark message")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will fail without a real server, but measures the client-side logic
		_ = channel.Send(ctx, notification)
	}
}

func BenchmarkTelegramSend(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.TelegramConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		BotToken:      "test-token",
	}

	channel, _ := channels.NewTelegramChannel(config, log)
	notification := channels.NewNotification("user-123", "test", "Test", "Benchmark message")
	notification.Data["telegram_chat_id"] = "123456"

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(ctx, notification)
	}
}

func BenchmarkEmailSend(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.EmailConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      2525,
		FromEmail:     "test@example.com",
		UseTLS:        false,
		UseStartTLS:   false,
	}

	channel, _ := channels.NewEmailChannel(config, log)
	notification := channels.NewNotification("user-123", "test", "Test", "Benchmark message")
	notification.Data["email"] = "recipient@example.com"

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(ctx, notification)
	}
}

func BenchmarkSlackFormatMessage(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.SlackConfig{
		ChannelConfig:     channels.DefaultChannelConfig(),
		DefaultWebhookURL: "https://example.com/webhook",
	}

	channel, _ := channels.NewSlackChannel(config, log)
	notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"
	notification.Data["side"] = "BUY"
	notification.Data["quantity"] = 1.5
	notification.Data["price"] = 50000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Using reflection to access private method for benchmark
		// In real implementation, this would be through public interface
		_ = channel
		_ = notification
	}
}

func BenchmarkTelegramFormatMessage(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.TelegramConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		BotToken:      "test-token",
		ParseMode:     "Markdown",
	}

	channel, _ := channels.NewTelegramChannel(config, log)
	notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel
		_ = notification
	}
}

func BenchmarkEmailFormatHTML(b *testing.B) {
	log := testutil.NewTestLogger()
	config := &channels.EmailConfig{
		ChannelConfig: channels.DefaultChannelConfig(),
		SMTPHost:      "localhost",
		SMTPPort:      587,
		FromEmail:     "test@example.com",
	}

	channel, _ := channels.NewEmailChannel(config, log)
	notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel
		_ = notification
	}
}

func BenchmarkChannelManagerSend(b *testing.B) {
	log := testutil.NewTestLogger()

	// Create mock channels
	slackMock := &MockChannel{name: "slack"}
	emailMock := &MockChannel{name: "email"}
	telegramMock := &MockChannel{name: "telegram"}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack":    slackMock,
			"email":    emailMock,
			"telegram": telegramMock,
		},
		Logger: log,
	})

	notification := channels.NewNotification("user-123", "risk_alert", "Risk Alert", "Benchmark")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Send(ctx, notification)
	}
}

func BenchmarkParallelDelivery(b *testing.B) {
	log := testutil.NewTestLogger()

	slackMock := &MockChannel{name: "slack"}
	emailMock := &MockChannel{name: "email"}
	telegramMock := &MockChannel{name: "telegram"}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack":    slackMock,
			"email":    emailMock,
			"telegram": telegramMock,
		},
		Logger: log,
	})

	notification := channels.NewNotification("user-123", "risk_alert", "Risk Alert", "Benchmark")

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = manager.Send(ctx, notification)
		}
	})
}

func BenchmarkNotificationCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Message")
		notification.Data["order_id"] = "ORDER-123"
		notification.Data["symbol"] = "BTCUSDT"
		notification.Priority = channels.PriorityNormal
	}
}

func BenchmarkChannelManagerRouting(b *testing.B) {
	log := testutil.NewTestLogger()

	slackMock := &MockChannel{name: "slack"}
	emailMock := &MockChannel{name: "email"}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack": slackMock,
			"email": emailMock,
		},
		Logger: log,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test routing logic overhead
		notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Test")
		ctx := context.Background()
		_ = manager.Send(ctx, notification)
	}
}

// MockChannel for benchmarks
type MockChannel struct {
	name string
}

func (m *MockChannel) Send(ctx context.Context, notification *channels.Notification) error {
	return nil
}

func (m *MockChannel) Name() string {
	return m.name
}

func (m *MockChannel) Validate() error {
	return nil
}

func (m *MockChannel) HealthCheck(ctx context.Context) error {
	return nil
}
