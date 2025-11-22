package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t testing.TB) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestNewRateLimiter(t *testing.T) {
	_, client := setupTestRedis(t)
	defer client.Close()

	rl := NewRateLimiter(client, 10, time.Minute)

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.limit)
	assert.Equal(t, time.Minute, rl.window)
	assert.Equal(t, "rate_limit:", rl.keyPrefix)
}

func TestRateLimiterMiddleware_AllowsRequestsWithinLimit(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	rl := NewRateLimiter(client, 5, time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := rl.Middleware(handler)

	// Make requests within limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Check headers
		assert.Equal(t, "5", rec.Header().Get("X-RateLimit-Limit"))
		remaining, _ := strconv.Atoi(rec.Header().Get("X-RateLimit-Remaining"))
		assert.Equal(t, 4-i, remaining)
	}
}

func TestRateLimiterMiddleware_BlocksExcessRequests(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	rl := NewRateLimiter(client, 3, time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Make requests exceeding limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if i < 3 {
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			assert.NotEmpty(t, rec.Header().Get("Retry-After"))
		}
	}
}

func TestRateLimiterMiddleware_ResetsAfterWindow(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	// Use 2 second window for reliable TTL expiration
	rl := NewRateLimiter(client, 2, 2*time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Use miniredis FastForward to expire the key
	mr.FastForward(3 * time.Second)

	// Should be allowed again after TTL expires
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiterMiddleware_DifferentIPsSeparateLimits(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	rl := NewRateLimiter(client, 2, time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	// Different IPs should have separate limits
	ips := []string{"1.1.1.1:1234", "2.2.2.2:1234", "3.3.3.3:1234"}

	for _, ip := range ips {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ip
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "Direct IP",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Forwarded-For single",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.1",
		},
		{
			name: "X-Forwarded-For multiple",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 10.0.0.2, 10.0.0.3",
			},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.2",
		},
		{
			name: "CF-Connecting-IP (Cloudflare)",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.3",
				"X-Forwarded-For":  "10.0.0.2",
			},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			assert.Equal(t, tc.expected, ip)
		})
	}
}

func TestAuthRateLimiter_CheckAndIncrement(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	al := NewAuthRateLimiter(client)
	ctx := context.Background()

	// Should allow first requests
	for i := 0; i < 5; i++ {
		err := al.CheckAndIncrement(ctx, "user1")
		assert.NoError(t, err)
	}

	// 6th request should be blocked
	err := al.CheckAndIncrement(ctx, "user1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestAuthRateLimiter_Lockout(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	al := NewAuthRateLimiter(client)
	ctx := context.Background()
	identifier := "user2"

	// Record multiple failed attempts
	for i := 0; i < 10; i++ {
		al.RecordFailedAttempt(ctx, identifier)
	}

	// Should be locked out
	err := al.CheckAndIncrement(ctx, identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account locked")
}

func TestAuthRateLimiter_ClearFailedAttempts(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	al := NewAuthRateLimiter(client)
	ctx := context.Background()
	identifier := "user3"

	// Record some failed attempts
	for i := 0; i < 5; i++ {
		al.RecordFailedAttempt(ctx, identifier)
	}

	// Clear failed attempts (successful login)
	al.ClearFailedAttempts(ctx, identifier)

	// Record more failed attempts - should start from 0
	for i := 0; i < 9; i++ {
		al.RecordFailedAttempt(ctx, identifier)
	}

	// Should not be locked out yet (9 < 10)
	err := al.CheckAndIncrement(ctx, identifier)
	assert.NoError(t, err)
}

func TestAuthRateLimiter_Middleware(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	al := NewAuthRateLimiter(client)

	// Mock handler that returns different status codes
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("X-Auth-Status")
		switch authHeader {
		case "success":
			w.WriteHeader(http.StatusOK)
		case "failed":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	middleware := al.Middleware(handler)

	// Test successful auth
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	req.Header.Set("X-Auth-Status", "success")
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Test failed auth
	req = httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	req.Header.Set("X-Auth-Status", "failed")
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Test non-auth endpoint (should bypass)
	req = httptest.NewRequest("GET", "/api/users", nil)
	req.RemoteAddr = "3.3.3.3:1234"
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIsAuthEndpoint(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"/auth/login", true},
		{"/auth/signin", true},
		{"/auth/register", true},
		{"/auth/signup", true},
		{"/api/auth/login", true},
		{"/api/v1/auth/login", true},
		{"/login", true},
		{"/signin", true},
		{"/register", true},
		{"/signup", true},
		{"/api/users", false},
		{"/health", false},
		{"/", false},
		{"/dashboard", false},
		{"/api/v1/users/123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := isAuthEndpoint(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAuthRateLimiter_LocalFallback(t *testing.T) {
	// Test without Redis (local fallback)
	al := NewAuthRateLimiter(nil)
	identifier := "local-user"

	// Should allow first requests
	for i := 0; i < 5; i++ {
		err := al.CheckAndIncrement(context.Background(), identifier)
		assert.NoError(t, err)
	}

	// 6th request should be blocked
	err := al.CheckAndIncrement(context.Background(), identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")

	// Test lockout with local state
	for i := 0; i < 10; i++ {
		al.RecordFailedAttempt(context.Background(), identifier)
	}

	// Should be locked out
	err = al.CheckAndIncrement(context.Background(), identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account locked")
}

func TestAuthRateLimiter_CleanupExpired(t *testing.T) {
	al := NewAuthRateLimiter(nil)

	// Add some state
	al.localState["old-user"] = &authState{
		lastAttempt: time.Now().Add(-25 * time.Hour),
		attempts:    1,
	}
	al.localState["recent-user"] = &authState{
		lastAttempt: time.Now(),
		attempts:    1,
	}
	al.localState["locked-user"] = &authState{
		lastAttempt: time.Now().Add(-25 * time.Hour),
		lockedUntil: time.Now().Add(1 * time.Hour),
		attempts:    1,
	}

	// Cleanup
	al.CleanupExpired()

	// Old user should be removed
	assert.NotContains(t, al.localState, "old-user")
	// Recent user should remain
	assert.Contains(t, al.localState, "recent-user")
	// Locked user should remain even if old
	assert.Contains(t, al.localState, "locked-user")
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	rl := NewRateLimiter(client, 10, time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(handler)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Make 20 concurrent requests (10 should succeed, 10 should fail)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "concurrent.test:1234"
			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)

			mu.Lock()
			if rec.Code == http.StatusOK {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Exactly 10 requests should succeed
	assert.Equal(t, 10, successCount)
}

func TestResponseWriter(t *testing.T) {
	original := httptest.NewRecorder()
	wrapped := &responseWriter{
		ResponseWriter: original,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	wrapped.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, wrapped.statusCode)
	assert.True(t, wrapped.written)

	// Second WriteHeader should be ignored
	wrapped.WriteHeader(http.StatusInternalServerError)
	assert.Equal(t, http.StatusNotFound, wrapped.statusCode)

	// Test Write
	wrapped2 := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	n, err := wrapped2.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, wrapped2.written)
	assert.Equal(t, http.StatusOK, wrapped2.statusCode)
}

func BenchmarkRateLimiter_Redis(b *testing.B) {
	mr, client := setupTestRedis(b)
	defer mr.Close()
	defer client.Close()

	rl := NewRateLimiter(client, 1000, time.Second)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := fmt.Sprintf("bench_user_%d", b.N)
		for pb.Next() {
			rl.checkRateLimit(ctx, key)
		}
	})
}

func BenchmarkRateLimiter_Local(b *testing.B) {
	rl := NewRateLimiter(nil, 1000, time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ip := fmt.Sprintf("192.168.1.%d", b.N%255)
		for pb.Next() {
			rl.checkLocalRateLimit(ip)
		}
	})
}
