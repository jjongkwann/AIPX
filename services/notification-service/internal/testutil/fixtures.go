package testutil

import (
	"time"
)

// Note: Fixture functions that create channel-specific types have been removed
// to avoid import cycles. Create test notifications directly in test files using
// channels.NewNotification() instead.

// TestEventData contains sample event data for different event types
var TestEventData = map[string]map[string]interface{}{
	"order_filled": {
		"event_type": "order_filled",
		"user_id":    "test-user-123",
		"order_id":   "ORDER-123",
		"symbol":     "BTCUSDT",
		"side":       "BUY",
		"quantity":   1.5,
		"price":      50000.0,
		"timestamp":  time.Now().Unix(),
	},
	"order_rejected": {
		"event_type": "order_rejected",
		"user_id":    "test-user-123",
		"order_id":   "ORDER-456",
		"symbol":     "ETHUSDT",
		"reason":     "insufficient_margin",
		"timestamp":  time.Now().Unix(),
	},
	"risk_alert": {
		"event_type": "risk_alert",
		"user_id":    "test-user-123",
		"position_id": "POS-789",
		"symbol":     "BTCUSDT",
		"risk_level": 0.85,
		"threshold":  0.75,
		"timestamp":  time.Now().Unix(),
	},
	"system_alert": {
		"event_type": "system_alert",
		"user_id":    "test-user-123",
		"service":    "trading-service",
		"status":     "degraded",
		"timestamp":  time.Now().Unix(),
	},
}
