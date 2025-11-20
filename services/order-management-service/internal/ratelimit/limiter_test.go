package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, client
}

func TestNewLimiter(t *testing.T) {
	mr, redisClient := newTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	t.Run("Success", func(t *testing.T) {
		config := &Config{Rate: 5, Burst: 10}
		limiter, err := NewLimiter(redisClient, config)

		require.NoError(t, err)
		assert.NotNil(t, limiter)
		assert.Equal(t, 5, limiter.rate)
		assert.Equal(t, 10, limiter.burst)
	})

	t.Run("Nil redis client", func(t *testing.T) {
		config := &Config{Rate: 5, Burst: 10}
		limiter, err := NewLimiter(nil, config)

		assert.Error(t, err)
		assert.Nil(t, limiter)
	})

	t.Run("Nil config", func(t *testing.T) {
		limiter, err := NewLimiter(redisClient, nil)

		assert.Error(t, err)
		assert.Nil(t, limiter)
	})

	t.Run("Invalid rate", func(t *testing.T) {
		config := &Config{Rate: 0, Burst: 10}
		limiter, err := NewLimiter(redisClient, config)

		assert.Error(t, err)
		assert.Nil(t, limiter)
	})

	t.Run("Auto-set burst", func(t *testing.T) {
		config := &Config{Rate: 5, Burst: 0}
		limiter, err := NewLimiter(redisClient, config)

		require.NoError(t, err)
		assert.Equal(t, 5, limiter.burst)
	})
}

func TestLimiter_Allow(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("First request allowed", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Burst capacity", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Should allow burst requests
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(ctx, "user1")
			assert.NoError(t, err)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 6th request should be denied
		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.False(t, allowed, "6th request should be denied")
	})

	t.Run("Token refill", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Exhaust tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(ctx, "user1")
		}

		// Next request should be denied
		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.False(t, allowed)

		// Wait for tokens to refill (1 second = 5 tokens)
		time.Sleep(1100 * time.Millisecond)

		// Should allow requests again
		allowed, err = limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Multiple users isolated", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// User1 exhausts tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(ctx, "user1")
		}

		// User1 should be denied
		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.False(t, allowed)

		// User2 should still be allowed
		allowed, err = limiter.Allow(ctx, "user2")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestLimiter_AllowN(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("Multiple tokens", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 10, Burst: 10}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Request 5 tokens
		allowed, err := limiter.AllowN(ctx, "user1", 5)
		assert.NoError(t, err)
		assert.True(t, allowed)

		// Request another 5 tokens
		allowed, err = limiter.AllowN(ctx, "user1", 5)
		assert.NoError(t, err)
		assert.True(t, allowed)

		// No tokens left
		allowed, err = limiter.AllowN(ctx, "user1", 1)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("Exceeds burst capacity", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		allowed, err := limiter.AllowN(ctx, "user1", 10)
		assert.Error(t, err)
		assert.False(t, allowed)
	})

	t.Run("Invalid n", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		allowed, err := limiter.AllowN(ctx, "user1", 0)
		assert.Error(t, err)
		assert.False(t, allowed)
	})
}

func TestLimiter_Concurrent(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("Concurrent requests", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 10, Burst: 10}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		var wg sync.WaitGroup
		results := make([]bool, 20)
		errors := make([]error, 20)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errors[idx] = limiter.Allow(ctx, "user1")
			}(i)
		}

		wg.Wait()

		// Count allowed requests
		allowedCount := 0
		for i := 0; i < 20; i++ {
			assert.NoError(t, errors[i])
			if results[i] {
				allowedCount++
			}
		}

		// Should allow exactly burst capacity
		assert.Equal(t, 10, allowedCount)
	})

	t.Run("Multiple users concurrent", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		var wg sync.WaitGroup
		users := []string{"user1", "user2", "user3"}
		results := make(map[string]int)
		var mu sync.Mutex

		for _, user := range users {
			results[user] = 0
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(u string) {
					defer wg.Done()
					allowed, err := limiter.Allow(ctx, u)
					assert.NoError(t, err)
					if allowed {
						mu.Lock()
						results[u]++
						mu.Unlock()
					}
				}(user)
			}
		}

		wg.Wait()

		// Each user should get their full burst
		for _, user := range users {
			assert.Equal(t, 5, results[user], "User %s should have 5 allowed requests", user)
		}
	})
}

func TestLimiter_Reset(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("Reset user limit", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Exhaust tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(ctx, "user1")
		}

		// Should be denied
		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.False(t, allowed)

		// Reset
		err = limiter.Reset(ctx, "user1")
		assert.NoError(t, err)

		// Should be allowed again
		allowed, err = limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestLimiter_GetRemaining(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("Initial remaining", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		remaining, err := limiter.GetRemaining(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 5, remaining)
	})

	t.Run("Remaining after consumption", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Consume 3 tokens
		limiter.Allow(ctx, "user1")
		limiter.Allow(ctx, "user1")
		limiter.Allow(ctx, "user1")

		remaining, err := limiter.GetRemaining(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 2, remaining)
	})

	t.Run("Remaining after refill", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Consume all tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(ctx, "user1")
		}

		remaining, err := limiter.GetRemaining(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 0, remaining)

		// Wait for refill
		time.Sleep(1100 * time.Millisecond)

		remaining, err = limiter.GetRemaining(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 5, remaining)
	})
}

func TestLimiter_PriorityAllow(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("Priority always allowed", func(t *testing.T) {
		redis.Reset()
		config := &Config{Rate: 5, Burst: 5}
		limiter, err := NewLimiter(redis.Client, config)
		require.NoError(t, err)

		// Exhaust tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(ctx, "user1")
		}

		// Normal request should be denied
		allowed, err := limiter.Allow(ctx, "user1")
		assert.NoError(t, err)
		assert.False(t, allowed)

		// Priority request should be allowed
		allowed, err = limiter.PriorityAllow(ctx, "user1")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}
