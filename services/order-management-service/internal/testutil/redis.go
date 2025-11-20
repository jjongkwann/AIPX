package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// RedisContainer represents a test Redis instance
type RedisContainer struct {
	Server *miniredis.Miniredis
	Client *redis.Client
}

// NewTestRedis creates a test Redis client using miniredis
func NewTestRedis(t *testing.T) *RedisContainer {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &RedisContainer{
		Server: mr,
		Client: client,
	}
}

// Close closes the test Redis instance
func (rc *RedisContainer) Close() {
	if rc.Client != nil {
		rc.Client.Close()
	}
	if rc.Server != nil {
		rc.Server.Close()
	}
}

// Reset resets the Redis data
func (rc *RedisContainer) Reset() {
	if rc.Server != nil {
		rc.Server.FlushAll()
	}
}
