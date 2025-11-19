package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// ErrLockNotObtained is returned when a lock cannot be obtained
	ErrLockNotObtained = errors.New("lock not obtained")

	// ErrLockNotHeld is returned when trying to unlock a lock that is not held
	ErrLockNotHeld = errors.New("lock not held")
)

// Lock represents a distributed lock
type Lock struct {
	client *Client
	key    string
	value  string
	ttl    time.Duration
}

// NewLock creates a new distributed lock
func NewLock(client *Client, key string, ttl time.Duration) *Lock {
	return &Lock{
		client: client,
		key:    key,
		value:  generateLockValue(),
		ttl:    ttl,
	}
}

// Acquire attempts to acquire the lock
// Returns ErrLockNotObtained if the lock is already held
func (l *Lock) Acquire(ctx context.Context) error {
	// Use SET with NX (only set if not exists) and EX (expiration)
	cmd := l.client.GetClient().SetNX(ctx, l.key, l.value, l.ttl)
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	acquired, err := cmd.Result()
	if err != nil {
		return fmt.Errorf("failed to get lock result: %w", err)
	}

	if !acquired {
		return ErrLockNotObtained
	}

	return nil
}

// TryAcquire attempts to acquire the lock with retries
func (l *Lock) TryAcquire(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		if err := l.Acquire(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			continue
		}
	}

	return ErrLockNotObtained
}

// Release releases the lock
// Only releases if the lock value matches (prevents releasing someone else's lock)
func (l *Lock) Release(ctx context.Context) error {
	// Lua script to atomically check value and delete
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.GetClient().Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == int64(0) {
		return ErrLockNotHeld
	}

	return nil
}

// Refresh extends the lock TTL
func (l *Lock) Refresh(ctx context.Context) error {
	// Lua script to atomically check value and extend TTL
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	ttlMs := l.ttl.Milliseconds()
	result, err := l.client.GetClient().Eval(ctx, script, []string{l.key}, l.value, ttlMs).Result()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result == int64(0) {
		return ErrLockNotHeld
	}

	return nil
}

// WithLock executes a function while holding the lock
// Automatically acquires and releases the lock
func (l *Lock) WithLock(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := l.Acquire(ctx); err != nil {
		return err
	}
	defer l.Release(ctx)

	return fn(ctx)
}

// WithLockRetry executes a function while holding the lock with retries
func (l *Lock) WithLockRetry(ctx context.Context, maxRetries int, retryDelay time.Duration, fn func(ctx context.Context) error) error {
	if err := l.TryAcquire(ctx, maxRetries, retryDelay); err != nil {
		return err
	}
	defer l.Release(ctx)

	return fn(ctx)
}

// generateLockValue generates a unique value for the lock
func generateLockValue() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Locker provides a simple interface for distributed locking
type Locker struct {
	client *Client
}

// NewLocker creates a new Locker instance
func NewLocker(client *Client) *Locker {
	return &Locker{
		client: client,
	}
}

// Lock creates and acquires a lock
func (l *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	lock := NewLock(l.client, key, ttl)
	if err := lock.Acquire(ctx); err != nil {
		return nil, err
	}
	return lock, nil
}

// TryLock attempts to acquire a lock with retries
func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration, maxRetries int, retryDelay time.Duration) (*Lock, error) {
	lock := NewLock(l.client, key, ttl)
	if err := lock.TryAcquire(ctx, maxRetries, retryDelay); err != nil {
		return nil, err
	}
	return lock, nil
}

// RateLimiter implements a simple rate limiter using Redis
type RateLimiter struct {
	client *Client
	limit  int64
	window time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client *Client, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

// Allow checks if a request is allowed for the given key
func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-r.window)

	pipe := r.client.GetClient().Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current entries
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration
	pipe.Expire(ctx, key, r.window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	count, err := countCmd.Result()
	if err != nil {
		return false, err
	}

	return count < r.limit, nil
}

// Reset resets the rate limit for a key
func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, key)
}
