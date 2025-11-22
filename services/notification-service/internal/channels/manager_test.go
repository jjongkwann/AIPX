package channels

import (
	"context"
	"testing"
	"time"

	"notification-service/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// MockChannel for testing
type MockChannel struct {
	name         string
	sendCalls    []*Notification
	shouldFail   bool
	sendDelay    time.Duration
	healthStatus error
}

func NewMockChannel(name string) *MockChannel {
	return &MockChannel{
		name:      name,
		sendCalls: make([]*Notification, 0),
	}
}

func (m *MockChannel) Send(ctx context.Context, notification *Notification) error {
	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}
	m.sendCalls = append(m.sendCalls, notification)
	if m.shouldFail {
		return ErrSendFailed
	}
	return nil
}

func (m *MockChannel) Name() string {
	return m.name
}

func (m *MockChannel) Validate() error {
	return nil
}

func (m *MockChannel) HealthCheck(ctx context.Context) error {
	return m.healthStatus
}

func TestChannelManager_Route_OrderNotifications(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	telegramCh := NewMockChannel("telegram")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	// Test order_filled routing (should go to slack + email)
	notification := NewNotification("user-123", "order_filled", "Order Filled", "Your order has been filled")

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	assert.NoError(t, err)

	// Verify routing
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 1)
	assert.Len(t, telegramCh.sendCalls, 0)
}

func TestChannelManager_Route_RiskAlerts(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	telegramCh := NewMockChannel("telegram")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	// Test risk_alert routing (should go to all channels)
	notification := NewNotification("user-123", "risk_alert", "Risk Alert", "Position risk exceeded")
	notification.Priority = PriorityCritical

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	assert.NoError(t, err)

	// Verify all channels received the notification
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 1)
	assert.Len(t, telegramCh.sendCalls, 1)
}

func TestChannelManager_Route_SystemAlerts(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	telegramCh := NewMockChannel("telegram")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	// Test system_alert routing (should go to slack + telegram)
	notification := NewNotification("user-123", "system_alert", "System Alert", "Service degradation")

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	assert.NoError(t, err)

	// Verify routing
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 0)
	assert.Len(t, telegramCh.sendCalls, 1)
}

func TestChannelManager_Send_ParallelExecution(t *testing.T) {
	log := testutil.NewTestLogger()

	// Create channels with delays
	slackCh := NewMockChannel("slack")
	slackCh.sendDelay = 100 * time.Millisecond

	emailCh := NewMockChannel("email")
	emailCh.sendDelay = 100 * time.Millisecond

	telegramCh := NewMockChannel("telegram")
	telegramCh.sendDelay = 100 * time.Millisecond

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	notification := NewNotification("user-123", "risk_alert", "Risk Alert", "Test parallel")

	start := time.Now()
	ctx := context.Background()
	err := manager.Send(ctx, notification)
	duration := time.Since(start)

	assert.NoError(t, err)

	// If executed serially, it would take 300ms+
	// If executed in parallel, it should take ~100ms
	assert.Less(t, duration, 200*time.Millisecond, "Should execute in parallel")
}

func TestChannelManager_Send_PartialFailure(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	slackCh.shouldFail = true

	emailCh := NewMockChannel("email")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
			"email": emailCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	notification := NewNotification("user-123", "order_filled", "Order Filled", "Test partial failure")

	ctx := context.Background()
	err := manager.Send(ctx, notification)

	// Should succeed because at least one channel succeeded
	assert.NoError(t, err)

	// Verify both channels were attempted
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 1)
}

func TestChannelManager_Metrics(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	emailCh.shouldFail = true

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
			"email": emailCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	// Send multiple notifications
	for i := 0; i < 5; i++ {
		notification := NewNotification("user-123", "order_filled", "Order Filled", "Test metrics")
		ctx := context.Background()
		manager.Send(ctx, notification)
	}

	metrics := manager.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(5), metrics["total_sent"])
	assert.Equal(t, int64(5), metrics["total_failed"])

	channelMetrics := metrics["channel_metrics"].(map[string]map[string]int64)
	assert.Equal(t, int64(5), channelMetrics["slack"]["success"])
	assert.Equal(t, int64(0), channelMetrics["slack"]["failure"])
	assert.Equal(t, int64(0), channelMetrics["email"]["success"])
	assert.Equal(t, int64(5), channelMetrics["email"]["failure"])
}

func TestChannelManager_RegisterChannel(t *testing.T) {
	log := testutil.NewTestLogger()

	config := &ChannelManagerConfig{
		Channels: make(map[string]NotificationChannel),
		Logger:   log,
	}

	manager := NewChannelManager(config)

	channel := NewMockChannel("test-channel")

	// Register channel
	err := manager.RegisterChannel("test-channel", channel)
	assert.NoError(t, err)

	// Try to register again (should fail)
	err = manager.RegisterChannel("test-channel", channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestChannelManager_UnregisterChannel(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	// Unregister channel
	manager.UnregisterChannel("slack")

	// Try to get channel (should not exist)
	_, exists := manager.GetChannel("slack")
	assert.False(t, exists)
}

func TestChannelManager_HealthCheck(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	slackCh.healthStatus = nil

	emailCh := NewMockChannel("email")
	emailCh.healthStatus = ErrSendFailed

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
			"email": emailCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	ctx := context.Background()
	results := manager.HealthCheck(ctx)

	assert.Len(t, results, 2)
	assert.NoError(t, results["slack"])
	assert.Error(t, results["email"])
}

func TestChannelManager_AddRoutingRule(t *testing.T) {
	log := testutil.NewTestLogger()

	config := &ChannelManagerConfig{
		Channels: make(map[string]NotificationChannel),
		Logger:   log,
	}

	manager := NewChannelManager(config)

	// Add custom routing rule
	manager.AddRoutingRule("custom_event", []string{"slack", "email"})

	rules := manager.GetRoutingRules()
	assert.Contains(t, rules, "custom_event")
	assert.Equal(t, []string{"slack", "email"}, rules["custom_event"])
}

func TestChannelManager_ListChannels(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	telegramCh := NewMockChannel("telegram")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	channels := manager.ListChannels()

	assert.Len(t, channels, 3)
	assert.Contains(t, channels, "slack")
	assert.Contains(t, channels, "email")
	assert.Contains(t, channels, "telegram")
}

func TestChannelManager_SendToChannel(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
			"email": emailCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	notification := NewNotification("user-123", "test", "Test", "Test message")

	ctx := context.Background()
	err := manager.SendToChannel(ctx, "slack", notification)
	assert.NoError(t, err)

	// Verify only slack received the notification
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 0)

	// Try to send to non-existent channel
	err = manager.SendToChannel(ctx, "nonexistent", notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestChannelManager_SendToChannels(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")
	telegramCh := NewMockChannel("telegram")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack":    slackCh,
			"email":    emailCh,
			"telegram": telegramCh,
		},
		Logger: log,
	}

	manager := NewChannelManager(config)

	notification := NewNotification("user-123", "test", "Test", "Test message")

	ctx := context.Background()
	err := manager.SendToChannels(ctx, []string{"slack", "telegram"}, notification)
	assert.NoError(t, err)

	// Verify correct channels received the notification
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 0)
	assert.Len(t, telegramCh.sendCalls, 1)
}

func TestChannelManager_NoChannelsConfigured(t *testing.T) {
	log := testutil.NewTestLogger()

	config := &ChannelManagerConfig{
		Channels: make(map[string]NotificationChannel),
		Logger:   log,
	}

	manager := NewChannelManager(config)

	notification := NewNotification("user-123", "unknown_event", "Test", "Test message")

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no channels configured")
}

func TestChannelManager_DefaultRoutingForCriticalPriority(t *testing.T) {
	log := testutil.NewTestLogger()

	slackCh := NewMockChannel("slack")
	emailCh := NewMockChannel("email")

	config := &ChannelManagerConfig{
		Channels: map[string]NotificationChannel{
			"slack": slackCh,
			"email": emailCh,
		},
		RoutingRules: make(map[string][]string), // Empty routing rules
		Logger:       log,
	}

	manager := NewChannelManager(config)

	// Unknown event type with critical priority should go to all channels
	notification := NewNotification("user-123", "unknown_critical_event", "Critical", "Critical issue")
	notification.Priority = PriorityCritical

	ctx := context.Background()
	err := manager.Send(ctx, notification)
	assert.NoError(t, err)

	// Verify all channels received the notification
	assert.Len(t, slackCh.sendCalls, 1)
	assert.Len(t, emailCh.sendCalls, 1)
}
