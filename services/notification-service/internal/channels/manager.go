package channels

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jjongkwann/aipx/shared/go/pkg/logger"
)

// ChannelManager manages multiple notification channels
type ChannelManager struct {
	channels         map[string]NotificationChannel
	routingRules     map[string][]string // event_type -> channel names
	logger           *logger.Logger
	mu               sync.RWMutex

	// Metrics
	totalSent        atomic.Int64
	totalFailed      atomic.Int64
	channelSuccesses map[string]*atomic.Int64
	channelFailures  map[string]*atomic.Int64
	metricsLock      sync.RWMutex
}

// ChannelManagerConfig holds channel manager configuration
type ChannelManagerConfig struct {
	Channels     map[string]NotificationChannel
	RoutingRules map[string][]string
	Logger       *logger.Logger
}

// NewChannelManager creates a new channel manager
func NewChannelManager(config *ChannelManagerConfig) *ChannelManager {
	if config == nil {
		config = &ChannelManagerConfig{
			Channels:     make(map[string]NotificationChannel),
			RoutingRules: make(map[string][]string),
		}
	}

	if config.Logger == nil {
		panic("logger is required for ChannelManager")
	}

	cm := &ChannelManager{
		channels:         config.Channels,
		routingRules:     config.RoutingRules,
		logger:           config.Logger,
		channelSuccesses: make(map[string]*atomic.Int64),
		channelFailures:  make(map[string]*atomic.Int64),
	}

	// Initialize metrics for each channel
	for name := range config.Channels {
		cm.channelSuccesses[name] = &atomic.Int64{}
		cm.channelFailures[name] = &atomic.Int64{}
	}

	// Set default routing rules if none provided
	if len(cm.routingRules) == 0 {
		cm.setDefaultRoutingRules()
	}

	return cm
}

// setDefaultRoutingRules sets default routing rules based on event types
func (cm *ChannelManager) setDefaultRoutingRules() {
	cm.routingRules = map[string][]string{
		// Order notifications: Slack + Email
		"order_filled":    {"slack", "email"},
		"order_rejected":  {"slack", "email"},
		"order_cancelled": {"slack", "email"},

		// Risk alerts: All channels (critical)
		"risk_alert":           {"slack", "telegram", "email"},
		"position_limit_alert": {"slack", "telegram", "email"},
		"margin_call":          {"slack", "telegram", "email"},

		// Position notifications: Slack + Email
		"position_opened": {"slack", "email"},
		"position_closed": {"slack", "email"},

		// System alerts: Slack + Telegram
		"system_alert":        {"slack", "telegram"},
		"maintenance_notice":  {"slack", "telegram"},
		"service_degradation": {"slack", "telegram"},
	}
}

// RegisterChannel registers a notification channel
func (cm *ChannelManager) RegisterChannel(name string, channel NotificationChannel) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.channels[name]; exists {
		return fmt.Errorf("channel %s already registered", name)
	}

	cm.channels[name] = channel

	// Initialize metrics
	cm.metricsLock.Lock()
	cm.channelSuccesses[name] = &atomic.Int64{}
	cm.channelFailures[name] = &atomic.Int64{}
	cm.metricsLock.Unlock()

	cm.logger.Info().
		Str("channel", name).
		Msg("Channel registered")

	return nil
}

// UnregisterChannel unregisters a notification channel
func (cm *ChannelManager) UnregisterChannel(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.channels, name)

	cm.logger.Info().
		Str("channel", name).
		Msg("Channel unregistered")
}

// GetChannel returns a channel by name
func (cm *ChannelManager) GetChannel(name string) (NotificationChannel, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	channel, exists := cm.channels[name]
	return channel, exists
}

// AddRoutingRule adds a routing rule for an event type
func (cm *ChannelManager) AddRoutingRule(eventType string, channelNames []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.routingRules[eventType] = channelNames

	cm.logger.Info().
		Str("event_type", eventType).
		Strs("channels", channelNames).
		Msg("Routing rule added")
}

// Send sends a notification to appropriate channels based on routing rules
func (cm *ChannelManager) Send(ctx context.Context, notification *Notification) error {
	// Get target channels for this event type
	targetChannels := cm.getTargetChannels(notification.EventType, notification.Priority)

	if len(targetChannels) == 0 {
		cm.logger.Warn().
			Str("event_type", notification.EventType).
			Msg("No channels configured for event type")
		return fmt.Errorf("no channels configured for event type: %s", notification.EventType)
	}

	// Send to all target channels in parallel
	return cm.sendToChannels(ctx, targetChannels, notification)
}

// SendToChannel sends a notification to a specific channel
func (cm *ChannelManager) SendToChannel(ctx context.Context, channelName string, notification *Notification) error {
	channel, exists := cm.GetChannel(channelName)
	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	// Send with timeout
	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := channel.Send(sendCtx, notification); err != nil {
		cm.recordFailure(channelName)
		return fmt.Errorf("failed to send via %s: %w", channelName, err)
	}

	cm.recordSuccess(channelName)
	return nil
}

// SendToChannels sends a notification to multiple specific channels
func (cm *ChannelManager) SendToChannels(ctx context.Context, channelNames []string, notification *Notification) error {
	channels := make([]NotificationChannel, 0, len(channelNames))

	cm.mu.RLock()
	for _, name := range channelNames {
		if channel, exists := cm.channels[name]; exists {
			channels = append(channels, channel)
		} else {
			cm.logger.Warn().
				Str("channel", name).
				Msg("Channel not found, skipping")
		}
	}
	cm.mu.RUnlock()

	if len(channels) == 0 {
		return fmt.Errorf("no valid channels found")
	}

	return cm.sendToChannels(ctx, channels, notification)
}

// sendToChannels sends notification to multiple channels in parallel
func (cm *ChannelManager) sendToChannels(ctx context.Context, channels []NotificationChannel, notification *Notification) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(channels))
	successCount := atomic.Int64{}

	for _, channel := range channels {
		wg.Add(1)

		go func(ch NotificationChannel) {
			defer wg.Done()

			// Create channel-specific context with timeout
			sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Send notification
			if err := ch.Send(sendCtx, notification); err != nil {
				cm.recordFailure(ch.Name())
				errChan <- fmt.Errorf("%s: %w", ch.Name(), err)
				return
			}

			cm.recordSuccess(ch.Name())
			successCount.Add(1)
		}(channel)
	}

	// Wait for all sends to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// If at least one channel succeeded, consider it a success
	if successCount.Load() > 0 {
		if len(errors) > 0 {
			cm.logger.Warn().
				Int("success", int(successCount.Load())).
				Int("failed", len(errors)).
				Msg("Partial notification delivery success")
		}
		return nil
	}

	// All channels failed
	if len(errors) > 0 {
		return fmt.Errorf("all channels failed: %v", errors)
	}

	return fmt.Errorf("no channels attempted")
}

// getTargetChannels returns target channels based on event type and priority
func (cm *ChannelManager) getTargetChannels(eventType string, priority Priority) []NotificationChannel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get channel names from routing rules
	var channelNames []string
	if names, exists := cm.routingRules[eventType]; exists {
		channelNames = names
	} else {
		// Default: send to all channels for critical priority
		if priority == PriorityCritical {
			for name := range cm.channels {
				channelNames = append(channelNames, name)
			}
		} else {
			// Default: send to slack for other priorities
			channelNames = []string{"slack"}
		}
	}

	// Get actual channel instances
	var channels []NotificationChannel
	for _, name := range channelNames {
		if channel, exists := cm.channels[name]; exists {
			channels = append(channels, channel)
		}
	}

	return channels
}

// HealthCheck performs health check on all channels
func (cm *ChannelManager) HealthCheck(ctx context.Context) map[string]error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, channel := range cm.channels {
		wg.Add(1)

		go func(n string, ch NotificationChannel) {
			defer wg.Done()

			// Create health check context with timeout
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			err := ch.HealthCheck(checkCtx)

			mu.Lock()
			results[n] = err
			mu.Unlock()

			if err != nil {
				cm.logger.Error().
					Err(err).
					Str("channel", n).
					Msg("Channel health check failed")
			} else {
				cm.logger.Debug().
					Str("channel", n).
					Msg("Channel health check passed")
			}
		}(name, channel)
	}

	wg.Wait()

	return results
}

// GetMetrics returns metrics for all channels
func (cm *ChannelManager) GetMetrics() map[string]interface{} {
	cm.metricsLock.RLock()
	defer cm.metricsLock.RUnlock()

	channelMetrics := make(map[string]map[string]int64)
	for name := range cm.channels {
		channelMetrics[name] = map[string]int64{
			"success": cm.channelSuccesses[name].Load(),
			"failure": cm.channelFailures[name].Load(),
		}
	}

	return map[string]interface{}{
		"total_sent":      cm.totalSent.Load(),
		"total_failed":    cm.totalFailed.Load(),
		"channel_metrics": channelMetrics,
	}
}

// recordSuccess records a successful send for a channel
func (cm *ChannelManager) recordSuccess(channelName string) {
	cm.totalSent.Add(1)

	cm.metricsLock.RLock()
	defer cm.metricsLock.RUnlock()

	if counter, exists := cm.channelSuccesses[channelName]; exists {
		counter.Add(1)
	}
}

// recordFailure records a failed send for a channel
func (cm *ChannelManager) recordFailure(channelName string) {
	cm.totalFailed.Add(1)

	cm.metricsLock.RLock()
	defer cm.metricsLock.RUnlock()

	if counter, exists := cm.channelFailures[channelName]; exists {
		counter.Add(1)
	}
}

// ListChannels returns a list of registered channel names
func (cm *ChannelManager) ListChannels() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]string, 0, len(cm.channels))
	for name := range cm.channels {
		names = append(names, name)
	}

	return names
}

// GetRoutingRules returns current routing rules
func (cm *ChannelManager) GetRoutingRules() map[string][]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	rules := make(map[string][]string, len(cm.routingRules))
	for eventType, channels := range cm.routingRules {
		channelsCopy := make([]string, len(channels))
		copy(channelsCopy, channels)
		rules[eventType] = channelsCopy
	}

	return rules
}

// Close closes all channels gracefully
func (cm *ChannelManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.logger.Info().Msg("Closing channel manager")

	// Channels don't need explicit closing in current implementation
	// But this method is here for future extensibility

	return nil
}
