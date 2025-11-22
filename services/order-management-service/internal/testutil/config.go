package testutil

import (
	"time"

	"order-management-service/internal/broker"
	"order-management-service/internal/ratelimit"
)

// NewTestRateLimitConfig creates a test rate limiter configuration
func NewTestRateLimitConfig() *ratelimit.Config {
	return &ratelimit.Config{
		Rate:  5,
		Burst: 5,
	}
}

// NewTestKISConfig creates a test KIS client configuration
func NewTestKISConfig(baseURL string) *broker.Config {
	return &broker.Config{
		BaseURL:    baseURL,
		AppKey:     "test_app_key",
		AppSecret:  "test_app_secret",
		AccountNo:  "12345678-01",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
	}
}
