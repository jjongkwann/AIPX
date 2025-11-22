package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter provides general rate limiting for HTTP endpoints
type RateLimiter struct {
	redis       *redis.Client
	limit       int
	window      time.Duration
	keyPrefix   string
	localLimits sync.Map // Fallback for when Redis is unavailable
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:     redis,
		limit:     limit,
		window:    window,
		keyPrefix: "rate_limit:",
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		key := rl.keyPrefix + clientIP

		ctx := r.Context()
		
		// Try Redis-based rate limiting first
		allowed, remaining, resetTime, err := rl.checkRateLimit(ctx, key)
		
		// If Redis fails, fall back to in-memory rate limiting
		if err != nil {
			allowed, remaining, resetTime = rl.checkLocalRateLimit(clientIP)
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		w.Header().Set("X-RateLimit-Reset-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// checkRateLimit checks the rate limit using Redis
func (rl *RateLimiter) checkRateLimit(ctx context.Context, key string) (bool, int, time.Time, error) {
	// Use Redis pipeline for atomic operation
	pipe := rl.redis.Pipeline()
	
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.window)
	ttl := pipe.TTL(ctx, key)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, time.Now(), err
	}
	
	count, err := incr.Result()
	if err != nil {
		return false, 0, time.Now(), err
	}
	
	ttlDuration, err := ttl.Result()
	if err != nil {
		ttlDuration = rl.window
	}
	
	resetTime := time.Now().Add(ttlDuration)
	remaining := max(0, rl.limit-int(count))
	allowed := count <= int64(rl.limit)
	
	return allowed, remaining, resetTime, nil
}

// checkLocalRateLimit provides fallback in-memory rate limiting
func (rl *RateLimiter) checkLocalRateLimit(clientIP string) (bool, int, time.Time) {
	// Use golang.org/x/time/rate for in-memory rate limiting
	value, _ := rl.localLimits.LoadOrStore(clientIP, rate.NewLimiter(rate.Every(rl.window/time.Duration(rl.limit)), rl.limit))
	limiter := value.(*rate.Limiter)
	
	allowed := limiter.Allow()
	// Approximate remaining tokens
	remaining := int(limiter.Tokens())
	resetTime := time.Now().Add(rl.window)
	
	return allowed, remaining, resetTime
}

// AuthRateLimiter provides specialized rate limiting for authentication endpoints
type AuthRateLimiter struct {
	redis        *redis.Client
	loginLimit   int           // requests per window
	window       time.Duration // time window for rate limiting
	lockoutLimit int           // failed attempts before lockout
	lockoutTime  time.Duration // lockout duration
	mu           sync.RWMutex
	localState   map[string]*authState // Fallback for when Redis is unavailable
}

// authState tracks authentication attempts in memory
type authState struct {
	attempts     int
	failedCount  int
	lastAttempt  time.Time
	lockedUntil  time.Time
}

// NewAuthRateLimiter creates a new authentication rate limiter with strict defaults
func NewAuthRateLimiter(redis *redis.Client) *AuthRateLimiter {
	return &AuthRateLimiter{
		redis:        redis,
		loginLimit:   5,              // 5 requests per minute
		window:       time.Minute,    
		lockoutLimit: 10,             // 10 failed attempts triggers lockout
		lockoutTime:  15 * time.Minute, // 15 minutes lockout
		localState:   make(map[string]*authState),
	}
}

// Middleware returns the authentication rate limiting middleware
func (al *AuthRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to auth endpoints
		if !isAuthEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		identifier := getIdentifier(r)
		ctx := r.Context()

		// Check rate limit and lockout
		err := al.CheckAndIncrement(ctx, identifier)
		if err != nil {
			// Set appropriate headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(al.loginLimit))
			w.Header().Set("X-RateLimit-Window", al.window.String())
			
			if strings.Contains(err.Error(), "locked") {
				w.Header().Set("X-Account-Locked", "true")
				http.Error(w, "Account temporarily locked due to too many failed attempts", http.StatusTooManyRequests)
			} else {
				http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
			}
			return
		}

		// Wrap the response writer to detect failed auth attempts
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Record failed attempt if auth failed
		if wrapped.statusCode == http.StatusUnauthorized || wrapped.statusCode == http.StatusForbidden {
			al.RecordFailedAttempt(ctx, identifier)
		} else if wrapped.statusCode == http.StatusOK || wrapped.statusCode == http.StatusCreated {
			// Clear failed attempts on successful auth
			al.ClearFailedAttempts(ctx, identifier)
		}
	})
}

// CheckAndIncrement checks rate limit and lockout status
func (al *AuthRateLimiter) CheckAndIncrement(ctx context.Context, identifier string) error {
	// Try Redis first
	if al.redis != nil {
		return al.checkRedis(ctx, identifier)
	}
	
	// Fallback to in-memory
	return al.checkLocal(identifier)
}

// checkRedis performs rate limiting checks using Redis
func (al *AuthRateLimiter) checkRedis(ctx context.Context, identifier string) error {
	// Check lockout first
	lockoutKey := "lockout:" + identifier
	locked, err := al.redis.Exists(ctx, lockoutKey).Result()
	if err == nil && locked > 0 {
		ttl, _ := al.redis.TTL(ctx, lockoutKey).Result()
		return fmt.Errorf("account locked for %v", ttl)
	}

	// Check rate limit
	rateKey := "auth_rate:" + identifier
	
	// Use Lua script for atomic operation
	script := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local current = redis.call('INCR', key)
		if current == 1 then
			redis.call('EXPIRE', key, window)
		end
		return current
	`
	
	count, err := al.redis.Eval(ctx, script, []string{rateKey}, al.loginLimit, int(al.window.Seconds())).Int()
	if err != nil {
		// If Redis fails, allow the request but log the error
		return nil
	}

	if count > al.loginLimit {
		return fmt.Errorf("rate limit exceeded: %d/%d requests", count, al.loginLimit)
	}

	return nil
}

// checkLocal performs rate limiting checks using in-memory state
func (al *AuthRateLimiter) checkLocal(identifier string) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	state, exists := al.localState[identifier]
	if !exists {
		state = &authState{
			lastAttempt: time.Now(),
			attempts:    1,
		}
		al.localState[identifier] = state
		return nil
	}

	// Check if locked out
	if time.Now().Before(state.lockedUntil) {
		return fmt.Errorf("account locked for %v", time.Until(state.lockedUntil))
	}

	// Reset counter if window expired
	if time.Since(state.lastAttempt) > al.window {
		state.attempts = 1
		state.lastAttempt = time.Now()
		return nil
	}

	// Check rate limit
	state.attempts++
	state.lastAttempt = time.Now()
	
	if state.attempts > al.loginLimit {
		return fmt.Errorf("rate limit exceeded: %d/%d requests", state.attempts, al.loginLimit)
	}

	return nil
}

// RecordFailedAttempt records a failed authentication attempt
func (al *AuthRateLimiter) RecordFailedAttempt(ctx context.Context, identifier string) {
	if al.redis != nil {
		al.recordFailedRedis(ctx, identifier)
	} else {
		al.recordFailedLocal(identifier)
	}
}

// recordFailedRedis records failed attempt in Redis
func (al *AuthRateLimiter) recordFailedRedis(ctx context.Context, identifier string) {
	key := "failed_auth:" + identifier
	
	// Increment failed attempts
	count, err := al.redis.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	
	// Set expiration on first failure
	if count == 1 {
		al.redis.Expire(ctx, key, al.lockoutTime)
	}
	
	// Check if we should lock out the account
	if count >= int64(al.lockoutLimit) {
		lockoutKey := "lockout:" + identifier
		al.redis.Set(ctx, lockoutKey, "1", al.lockoutTime)
		
		// Log the lockout event (in production, this should go to a security log)
		// log.Warn().Str("identifier", identifier).Msg("Account locked due to failed attempts")
	}
}

// recordFailedLocal records failed attempt in memory
func (al *AuthRateLimiter) recordFailedLocal(identifier string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	state, exists := al.localState[identifier]
	if !exists {
		state = &authState{}
		al.localState[identifier] = state
	}

	state.failedCount++
	
	if state.failedCount >= al.lockoutLimit {
		state.lockedUntil = time.Now().Add(al.lockoutTime)
		state.failedCount = 0 // Reset counter after lockout
	}
}

// ClearFailedAttempts clears the failed attempt counter for successful authentication
func (al *AuthRateLimiter) ClearFailedAttempts(ctx context.Context, identifier string) {
	if al.redis != nil {
		al.redis.Del(ctx, "failed_auth:"+identifier)
	} else {
		al.mu.Lock()
		defer al.mu.Unlock()
		if state, exists := al.localState[identifier]; exists {
			state.failedCount = 0
		}
	}
}

// CleanupExpired removes expired entries from local state (should be called periodically)
func (al *AuthRateLimiter) CleanupExpired() {
	al.mu.Lock()
	defer al.mu.Unlock()

	now := time.Now()
	for id, state := range al.localState {
		// Remove entries that haven't been used recently and are not locked
		if time.Since(state.lastAttempt) > 24*time.Hour && now.After(state.lockedUntil) {
			delete(al.localState, id)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if !w.written {
		w.statusCode = statusCode
		w.written = true
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check CF-Connecting-IP (Cloudflare)
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}
	
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (client IP)
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])
		if clientIP != "" {
			return clientIP
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getIdentifier gets a unique identifier for rate limiting
// Uses a combination of IP and username/email if available
func getIdentifier(r *http.Request) string {
	ip := getClientIP(r)
	
	// For login attempts, also consider username/email
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			if username := r.FormValue("username"); username != "" {
				return ip + ":" + username
			}
			if email := r.FormValue("email"); email != "" {
				return ip + ":" + email
			}
		}
		
		// Also check JSON body for API requests
		// This would require reading and restoring the body
		// Simplified for this example
	}
	
	return ip
}

// isAuthEndpoint checks if the path is an authentication endpoint
func isAuthEndpoint(path string) bool {
	authPaths := []string{
		"/auth/login",
		"/auth/signin",
		"/auth/register",
		"/auth/signup",
		"/auth/token",
		"/auth/refresh",
		"/api/auth/login",
		"/api/auth/register",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/login",
		"/signin",
		"/register",
		"/signup",
	}
	
	path = strings.ToLower(strings.TrimSpace(path))
	for _, authPath := range authPaths {
		if path == authPath || strings.HasSuffix(path, authPath) {
			return true
		}
	}
	
	return false
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}