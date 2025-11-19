package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client wraps Redis client with additional features
type Client struct {
	client *redis.Client
	config *Config
	mu     sync.RWMutex
	closed bool
}

// NewClient creates a new Redis client with connection retry
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxConnAge:   config.MaxConnAge,
		PoolTimeout:  config.PoolTimeout,
	})

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", config.Addr, err)
	}

	return &Client{
		client: rdb,
		config: config,
		closed: false,
	}, nil
}

// GetClient returns the underlying redis.Client
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Set sets a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return "", fmt.Errorf("client is closed")
	}

	return c.client.Get(ctx, key).Result()
}

// Del deletes one or more keys
func (c *Client) Del(ctx context.Context, keys ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, fmt.Errorf("client is closed")
	}

	return c.client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, fmt.Errorf("client is closed")
	}

	return c.client.TTL(ctx, key).Result()
}

// HSet sets field in hash
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.HSet(ctx, key, values...).Err()
}

// HGet gets field from hash
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return "", fmt.Errorf("client is closed")
	}

	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.HGetAll(ctx, key).Result()
}

// HDel deletes fields from hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.HDel(ctx, key, fields...).Err()
}

// LPush prepends values to list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.LPush(ctx, key, values...).Err()
}

// RPush appends values to list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.RPush(ctx, key, values...).Err()
}

// LRange gets range of elements from list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.LRange(ctx, key, start, stop).Result()
}

// SAdd adds members to set
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers gets all members of set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.SMembers(ctx, key).Result()
}

// ZAdd adds members to sorted set
func (c *Client) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRange gets range of members from sorted set
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.ZRange(ctx, key, start, stop).Result()
}

// Incr increments integer value
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, fmt.Errorf("client is closed")
	}

	return c.client.Incr(ctx, key).Result()
}

// IncrBy increments integer value by amount
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, fmt.Errorf("client is closed")
	}

	return c.client.IncrBy(ctx, key, value).Result()
}

// Decr decrements integer value
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, fmt.Errorf("client is closed")
	}

	return c.client.Decr(ctx, key).Result()
}

// Publish publishes message to channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client.Subscribe(ctx, channels...)
}

// HealthCheck verifies Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.client.Close()
}
