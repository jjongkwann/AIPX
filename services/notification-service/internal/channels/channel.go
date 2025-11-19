package channels

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrChannelNotConfigured = errors.New("channel not configured")
	ErrInvalidConfig        = errors.New("invalid channel configuration")
	ErrSendFailed           = errors.New("failed to send notification")
	ErrRateLimited          = errors.New("rate limit exceeded")
	ErrInvalidPayload       = errors.New("invalid notification payload")
)

// NotificationChannel defines the interface for notification channels
type NotificationChannel interface {
	// Send sends a notification through the channel
	Send(ctx context.Context, notification *Notification) error

	// Name returns the channel name
	Name() string

	// Validate validates the channel configuration
	Validate() error

	// HealthCheck checks if the channel is operational
	HealthCheck(ctx context.Context) error
}

// Notification represents a notification to be sent
type Notification struct {
	// Core fields
	UserID    string
	EventType string
	Title     string
	Message   string

	// Metadata
	Data      map[string]interface{}
	Priority  Priority
	Timestamp time.Time

	// Retry configuration
	RetryCount    int
	MaxRetries    int
	RetryInterval time.Duration
}

// Priority defines notification priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ChannelConfig represents common configuration for channels
type ChannelConfig struct {
	Enabled       bool
	Timeout       time.Duration
	RetryAttempts int
	RetryInterval time.Duration
	RateLimit     RateLimitConfig
}

// RateLimitConfig defines rate limiting for channels
type RateLimitConfig struct {
	Enabled        bool
	MaxPerMinute   int
	MaxPerHour     int
	BurstSize      int
	CleanupTimeout time.Duration
}

// DefaultChannelConfig returns default channel configuration
func DefaultChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		Enabled:       true,
		Timeout:       10 * time.Second,
		RetryAttempts: 3,
		RetryInterval: 5 * time.Second,
		RateLimit: RateLimitConfig{
			Enabled:        true,
			MaxPerMinute:   60,
			MaxPerHour:     1000,
			BurstSize:      10,
			CleanupTimeout: 1 * time.Hour,
		},
	}
}

// NewNotification creates a new notification with defaults
func NewNotification(userID, eventType, title, message string) *Notification {
	return &Notification{
		UserID:        userID,
		EventType:     eventType,
		Title:         title,
		Message:       message,
		Data:          make(map[string]interface{}),
		Priority:      PriorityNormal,
		Timestamp:     time.Now(),
		RetryCount:    0,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
	}
}
