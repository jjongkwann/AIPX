package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
)

// Cache provides high-level caching operations
type Cache struct {
	client *Client
}

// NewCache creates a new Cache instance
func NewCache(client *Client) *Cache {
	return &Cache{
		client: client,
	}
}

// GetBytes retrieves raw bytes from cache
func (c *Cache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// SetBytes stores raw bytes in cache
func (c *Cache) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl)
}

// GetJSON retrieves and unmarshals JSON from cache
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.GetBytes(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// SetJSON marshals and stores JSON in cache
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.SetBytes(ctx, key, data, ttl)
}

// GetProto retrieves and unmarshals protobuf from cache
func (c *Cache) GetProto(ctx context.Context, key string, dest proto.Message) error {
	data, err := c.GetBytes(ctx, key)
	if err != nil {
		return err
	}

	if err := proto.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal proto: %w", err)
	}

	return nil
}

// SetProto marshals and stores protobuf in cache
func (c *Cache) SetProto(ctx context.Context, key string, value proto.Message, ttl time.Duration) error {
	data, err := proto.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal proto: %w", err)
	}

	return c.SetBytes(ctx, key, data, ttl)
}

// Delete removes a key from cache
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...)
}

// Exists checks if keys exist in cache
func (c *Cache) Exists(ctx context.Context, keys ...string) (bool, error) {
	count, err := c.client.Exists(ctx, keys...)
	if err != nil {
		return false, err
	}
	return count == int64(len(keys)), nil
}

// GetOrSet retrieves from cache or sets using the provided function
func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	// Try to get from cache first
	data, err := c.GetBytes(ctx, key)
	if err == nil {
		return data, nil
	}

	// Cache miss, call the function
	data, err = fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors to avoid blocking)
	_ = c.SetBytes(ctx, key, data, ttl)

	return data, nil
}

// GetOrSetJSON retrieves from cache or sets using the provided function (JSON)
func (c *Cache) GetOrSetJSON(ctx context.Context, key string, ttl time.Duration, dest interface{}, fn func() (interface{}, error)) error {
	// Try to get from cache first
	if err := c.GetJSON(ctx, key, dest); err == nil {
		return nil
	}

	// Cache miss, call the function
	value, err := fn()
	if err != nil {
		return err
	}

	// Store in cache (ignore errors to avoid blocking)
	_ = c.SetJSON(ctx, key, value, ttl)

	// Marshal the result into dest
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// GetOrSetProto retrieves from cache or sets using the provided function (Protobuf)
func (c *Cache) GetOrSetProto(ctx context.Context, key string, ttl time.Duration, dest proto.Message, fn func() (proto.Message, error)) error {
	// Try to get from cache first
	if err := c.GetProto(ctx, key, dest); err == nil {
		return nil
	}

	// Cache miss, call the function
	value, err := fn()
	if err != nil {
		return err
	}

	// Store in cache (ignore errors to avoid blocking)
	_ = c.SetProto(ctx, key, value, ttl)

	// Marshal and unmarshal to populate dest
	data, err := proto.Marshal(value)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, dest)
}

// TTL returns the remaining time to live of a key
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key)
}

// Refresh updates the TTL of a key without changing its value
func (c *Cache) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl)
}

// DeletePattern deletes all keys matching a pattern (use with caution)
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	// Note: KEYS command is blocking and should not be used in production
	// Consider using SCAN instead for production use
	iter := c.client.GetClient().Scan(ctx, 0, pattern, 0).Iterator()

	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...)
	}

	return nil
}
