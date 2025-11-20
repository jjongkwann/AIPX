package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Limiter implements a Redis-based token bucket rate limiter
type Limiter struct {
	redis *redis.Client
	rate  int // tokens per second
	burst int // max burst size
}

// Config holds the rate limiter configuration
type Config struct {
	Rate  int // tokens per second per user
	Burst int // maximum burst capacity
}

// NewLimiter creates a new rate limiter
func NewLimiter(redisClient *redis.Client, config *Config) (*Limiter, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Rate <= 0 {
		return nil, fmt.Errorf("rate must be positive")
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}

	limiter := &Limiter{
		redis: redisClient,
		rate:  config.Rate,
		burst: config.Burst,
	}

	log.Info().
		Int("rate", config.Rate).
		Int("burst", config.Burst).
		Msg("Rate limiter initialized")

	return limiter, nil
}

// Allow checks if a single request is allowed for the given key (e.g., user ID)
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allowed for the given key
func (l *Limiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return false, fmt.Errorf("n must be positive")
	}
	if n > l.burst {
		return false, fmt.Errorf("requested tokens %d exceeds burst capacity %d", n, l.burst)
	}

	// Use Redis Lua script for atomic token bucket operation
	allowed, err := l.tokenBucket(ctx, key, n)
	if err != nil {
		log.Error().
			Err(err).
			Str("key", key).
			Int("tokens", n).
			Msg("Rate limit check failed")
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	if !allowed {
		log.Warn().
			Str("key", key).
			Int("requested", n).
			Msg("Rate limit exceeded")
	}

	return allowed, nil
}

// tokenBucket implements the token bucket algorithm using Redis Lua script
func (l *Limiter) tokenBucket(ctx context.Context, key string, tokens int) (bool, error) {
	now := time.Now().UnixNano()
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	// Lua script for atomic token bucket implementation
	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		-- Get current bucket state
		local bucket = redis.call('HMGET', key, 'tokens', 'last_update')
		local tokens = tonumber(bucket[1])
		local last_update = tonumber(bucket[2])

		-- Initialize if first time
		if tokens == nil then
			tokens = burst
			last_update = now
		end

		-- Calculate tokens to add based on time elapsed
		local elapsed = (now - last_update) / 1e9  -- convert nanoseconds to seconds
		local tokens_to_add = math.floor(elapsed * rate)

		-- Update tokens (max = burst)
		tokens = math.min(tokens + tokens_to_add, burst)

		-- Check if we have enough tokens
		if tokens >= requested then
			tokens = tokens - requested
			-- Update bucket state
			redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
			redis.call('EXPIRE', key, 60)  -- expire after 60 seconds of inactivity
			return 1  -- allowed
		else
			-- Update last_update even if request is denied
			redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
			redis.call('EXPIRE', key, 60)
			return 0  -- denied
		end
	`

	result, err := l.redis.Eval(
		ctx,
		script,
		[]string{rateLimitKey},
		l.rate,
		l.burst,
		now,
		tokens,
	).Result()

	if err != nil {
		return false, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	allowed := result.(int64) == 1
	return allowed, nil
}

// Reset resets the rate limit for a given key
func (l *Limiter) Reset(ctx context.Context, key string) error {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	return l.redis.Del(ctx, rateLimitKey).Err()
}

// GetRemaining returns the number of remaining tokens for a key
func (l *Limiter) GetRemaining(ctx context.Context, key string) (int, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	now := time.Now().UnixNano()

	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		local bucket = redis.call('HMGET', key, 'tokens', 'last_update')
		local tokens = tonumber(bucket[1])
		local last_update = tonumber(bucket[2])

		if tokens == nil then
			return burst
		end

		local elapsed = (now - last_update) / 1e9
		local tokens_to_add = math.floor(elapsed * rate)
		tokens = math.min(tokens + tokens_to_add, burst)

		return tokens
	`

	result, err := l.redis.Eval(
		ctx,
		script,
		[]string{rateLimitKey},
		l.rate,
		l.burst,
		now,
	).Result()

	if err != nil {
		return 0, fmt.Errorf("failed to get remaining tokens: %w", err)
	}

	return int(result.(int64)), nil
}

// PriorityAllow allows priority requests (e.g., market orders) to bypass rate limits
// This should be used sparingly and with caution
func (l *Limiter) PriorityAllow(ctx context.Context, key string) (bool, error) {
	log.Info().
		Str("key", key).
		Msg("Priority request bypassing rate limit")
	return true, nil
}
