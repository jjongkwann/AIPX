package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// Client wraps Redis client
type Client struct {
	client *redis.Client
	config *Config
}

// Config holds Redis configuration
type Config struct {
	Host         string
	Port         string
	Password     string
	DB           int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
}

// NewClient creates a new Redis client
func NewClient(config *Config) (*Client, error) {
	addr := config.Host + ":" + config.Port

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     config.Password,
		DB:           config.DB,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
	})

	// Ping to check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info().Str("addr", addr).Msg("Connected to Redis")

	return &Client{
		client: rdb,
		config: config,
	}, nil
}

// Set sets a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Del deletes one or more keys
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// HSet sets field in hash
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

// HGet gets field from hash
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HDel deletes fields from hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, key, fields...).Err()
}

// LPush prepends values to list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPush appends values to list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, key, values...).Err()
}

// LRange gets range of elements from list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// Publish publishes message to channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// DefaultConfig returns default Redis configuration
func DefaultConfig(host, port string) *Config {
	return &Config{
		Host:         host,
		Port:         port,
		Password:     "",
		DB:           0,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	}
}
