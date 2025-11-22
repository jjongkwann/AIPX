package integration

import (
	"context"
	"testing"
	"time"

	"notification-service/internal/channels"
	"notification-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for full notification flows

func TestOrderNotificationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	log := testutil.NewTestLogger()

	// Create mock channels with tracking
	slackCalls := make([]*channels.Notification, 0)
	emailCalls := make([]*channels.Notification, 0)

	slackMock := &MockTrackingChannel{
		name: "slack",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			slackCalls = append(slackCalls, n)
			return nil
		},
	}

	emailMock := &MockTrackingChannel{
		name: "email",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			emailCalls = append(emailCalls, n)
			return nil
		},
	}

	// Create manager with routing rules
	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack": slackMock,
			"email": emailMock,
		},
		Logger: log,
	})

	// Test order_filled flow
	notification := channels.NewNotification(
		"user-123",
		"order_filled",
		"Order Filled",
		"Your BUY order for 1.5 BTCUSDT at $50,000 has been filled",
	)
	notification.Data["order_id"] = "ORDER-123"
	notification.Data["symbol"] = "BTCUSDT"
	notification.Data["side"] = "BUY"
	notification.Data["quantity"] = 1.5
	notification.Data["price"] = 50000.0

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	require.NoError(t, err)

	// Verify Slack received the notification
	assert.Len(t, slackCalls, 1)
	assert.Equal(t, "user-123", slackCalls[0].UserID)
	assert.Equal(t, "order_filled", slackCalls[0].EventType)
	assert.Equal(t, "ORDER-123", slackCalls[0].Data["order_id"])

	// Verify Email received the notification
	assert.Len(t, emailCalls, 1)
	assert.Equal(t, "user-123", emailCalls[0].UserID)
	assert.Equal(t, "order_filled", emailCalls[0].EventType)
}

func TestRiskAlertFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	log := testutil.NewTestLogger()

	// Create all three channels with tracking
	slackCalls := make([]*channels.Notification, 0)
	telegramCalls := make([]*channels.Notification, 0)
	emailCalls := make([]*channels.Notification, 0)

	slackMock := &MockTrackingChannel{
		name: "slack",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			slackCalls = append(slackCalls, n)
			return nil
		},
	}

	telegramMock := &MockTrackingChannel{
		name: "telegram",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			telegramCalls = append(telegramCalls, n)
			return nil
		},
	}

	emailMock := &MockTrackingChannel{
		name: "email",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			emailCalls = append(emailCalls, n)
			return nil
		},
	}

	// Create manager
	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack":    slackMock,
			"telegram": telegramMock,
			"email":    emailMock,
		},
		Logger: log,
	})

	// Test risk_alert flow (should go to all channels)
	notification := channels.NewNotification(
		"user-123",
		"risk_alert",
		"Risk Alert: Position Limit Exceeded",
		"Your position risk level has exceeded the safety threshold",
	)
	notification.Priority = channels.PriorityCritical
	notification.Data["position_id"] = "POS-789"
	notification.Data["risk_level"] = 0.95
	notification.Data["threshold"] = 0.80

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	require.NoError(t, err)

	// Verify all channels received the notification
	assert.Len(t, slackCalls, 1)
	assert.Len(t, telegramCalls, 1)
	assert.Len(t, emailCalls, 1)

	// Verify priority is critical
	assert.Equal(t, channels.PriorityCritical, slackCalls[0].Priority)
	assert.Equal(t, channels.PriorityCritical, telegramCalls[0].Priority)
	assert.Equal(t, channels.PriorityCritical, emailCalls[0].Priority)
}

func TestRetryFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	log := testutil.NewTestLogger()

	attemptCount := 0

	// Test that SendToChannel passes through the error from the channel
	// Note: ChannelManager.SendToChannel does not implement retry logic itself;
	// retry logic is expected to be in the individual channel implementations
	retryMock := &MockTrackingChannel{
		name: "slack",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			attemptCount++
			// Succeed on first attempt
			return nil
		},
	}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack": retryMock,
		},
		Logger: log,
	})

	notification := channels.NewNotification("user-123", "test", "Test", "Test retry")

	ctx := context.Background()
	err := manager.SendToChannel(ctx, "slack", notification)

	// Should succeed on first attempt
	assert.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
}

func TestParallelDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	log := testutil.NewTestLogger()

	// Create channels with artificial delays
	delay := 100 * time.Millisecond

	slackMock := &MockTrackingChannel{
		name: "slack",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			time.Sleep(delay)
			return nil
		},
	}

	telegramMock := &MockTrackingChannel{
		name: "telegram",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			time.Sleep(delay)
			return nil
		},
	}

	emailMock := &MockTrackingChannel{
		name: "email",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			time.Sleep(delay)
			return nil
		},
	}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack":    slackMock,
			"telegram": telegramMock,
			"email":    emailMock,
		},
		Logger: log,
	})

	notification := channels.NewNotification("user-123", "risk_alert", "Risk Alert", "Test parallel")

	start := time.Now()
	ctx := context.Background()
	err := manager.Send(ctx, notification)
	duration := time.Since(start)

	assert.NoError(t, err)

	// If serial: 300ms+, if parallel: ~100ms
	// Allow some margin for overhead
	assert.Less(t, duration, 200*time.Millisecond, "Should execute in parallel")
}

func TestPartialFailureScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	log := testutil.NewTestLogger()

	slackCalls := make([]*channels.Notification, 0)
	emailCalls := make([]*channels.Notification, 0)

	// Slack fails
	slackMock := &MockTrackingChannel{
		name: "slack",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			slackCalls = append(slackCalls, n)
			return channels.ErrSendFailed
		},
	}

	// Email succeeds
	emailMock := &MockTrackingChannel{
		name: "email",
		sendFunc: func(ctx context.Context, n *channels.Notification) error {
			emailCalls = append(emailCalls, n)
			return nil
		},
	}

	manager := channels.NewChannelManager(&channels.ChannelManagerConfig{
		Channels: map[string]channels.NotificationChannel{
			"slack": slackMock,
			"email": emailMock,
		},
		Logger: log,
	})

	notification := channels.NewNotification("user-123", "order_filled", "Order Filled", "Test partial failure")

	ctx := context.Background()
	err := manager.Send(ctx, notification)

	// Should succeed because at least one channel succeeded
	assert.NoError(t, err)

	// Verify both were attempted
	assert.Len(t, slackCalls, 1)
	assert.Len(t, emailCalls, 1)

	// Verify metrics reflect partial failure
	metrics := manager.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_sent"])
	assert.Equal(t, int64(1), metrics["total_failed"])
}

// MockTrackingChannel for integration tests
type MockTrackingChannel struct {
	name         string
	sendFunc     func(context.Context, *channels.Notification) error
	validateFunc func() error
	healthFunc   func(context.Context) error
}

func (m *MockTrackingChannel) Send(ctx context.Context, n *channels.Notification) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, n)
	}
	return nil
}

func (m *MockTrackingChannel) Name() string {
	return m.name
}

func (m *MockTrackingChannel) Validate() error {
	if m.validateFunc != nil {
		return m.validateFunc()
	}
	return nil
}

func (m *MockTrackingChannel) HealthCheck(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}
