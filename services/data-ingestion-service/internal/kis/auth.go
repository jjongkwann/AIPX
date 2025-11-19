package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/jjongkwann/aipx/services/data-ingestion-service/internal/config"
	"github.com/jjongkwann/aipx/shared/go/pkg/redis"
	"github.com/rs/zerolog/log"
)

const (
	tokenCacheKey = "kis:access_token"
)

// TokenResponse represents the response from KIS token API
type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int    `json:"expires_in"`
	AccessTokenExpired    string `json:"access_token_token_expired"`
	AccessTokenExpiration string `json:"access_token_expiration"`
}

// TokenRequest represents the request to KIS token API
type TokenRequest struct {
	GrantType string `json:"grant_type"`
	AppKey    string `json:"appkey"`
	AppSecret string `json:"appsecret"`
}

// AuthManager manages KIS authentication tokens
type AuthManager struct {
	config      *config.KISConfig
	cache       *redis.Cache
	httpClient  *http.Client
	mu          sync.RWMutex
	token       string
	expiresAt   time.Time
	refreshTask *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(cfg *config.KISConfig, cache *redis.Cache) *AuthManager {
	ctx, cancel := context.WithCancel(context.Background())

	am := &AuthManager{
		config: cfg,
		cache:  cache,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return am
}

// Start initializes the authentication manager
func (am *AuthManager) Start() error {
	// Try to get token from cache first
	if err := am.loadTokenFromCache(); err != nil {
		log.Debug().Err(err).Msg("failed to load token from cache, will fetch new token")
	}

	// If no valid token, fetch a new one
	if am.token == "" || time.Now().After(am.expiresAt) {
		if err := am.refreshToken(); err != nil {
			return fmt.Errorf("failed to fetch initial token: %w", err)
		}
	}

	// Start background refresh task
	go am.startRefreshTask()

	log.Info().
		Time("expires_at", am.expiresAt).
		Msg("authentication manager started")

	return nil
}

// Stop stops the authentication manager
func (am *AuthManager) Stop() {
	am.cancel()
	if am.refreshTask != nil {
		am.refreshTask.Stop()
	}
	log.Info().Msg("authentication manager stopped")
}

// GetToken returns the current valid access token
func (am *AuthManager) GetToken() (string, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.token == "" {
		return "", fmt.Errorf("no token available")
	}

	if time.Now().After(am.expiresAt) {
		return "", fmt.Errorf("token expired")
	}

	return am.token, nil
}

// refreshToken fetches a new token from KIS API
func (am *AuthManager) refreshToken() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	log.Info().Msg("refreshing KIS access token")

	reqBody := TokenRequest{
		GrantType: "client_credentials",
		AppKey:    am.config.APIKey,
		AppSecret: am.config.APISecret,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/oauth2/tokenP", am.config.APIURL)
	req, err := http.NewRequestWithContext(am.ctx, http.MethodPost, url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("received empty access token")
	}

	// Update token and expiration
	am.token = tokenResp.AccessToken
	am.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Cache token
	if err := am.cacheToken(); err != nil {
		log.Warn().Err(err).Msg("failed to cache token")
	}

	log.Info().
		Time("expires_at", am.expiresAt).
		Msg("token refreshed successfully")

	return nil
}

// loadTokenFromCache loads token from Redis cache
func (am *AuthManager) loadTokenFromCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cachedData struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := am.cache.GetJSON(ctx, tokenCacheKey, &cachedData); err != nil {
		return err
	}

	if cachedData.Token == "" || time.Now().After(cachedData.ExpiresAt) {
		return fmt.Errorf("cached token is invalid or expired")
	}

	am.mu.Lock()
	am.token = cachedData.Token
	am.expiresAt = cachedData.ExpiresAt
	am.mu.Unlock()

	log.Info().
		Time("expires_at", cachedData.ExpiresAt).
		Msg("loaded token from cache")

	return nil
}

// cacheToken stores token in Redis cache
func (am *AuthManager) cacheToken() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}{
		Token:     am.token,
		ExpiresAt: am.expiresAt,
	}

	return am.cache.SetJSON(ctx, tokenCacheKey, data, am.config.TokenCacheTTL)
}

// startRefreshTask starts a background task to refresh token before expiration
func (am *AuthManager) startRefreshTask() {
	// Calculate next refresh time
	refreshInterval := am.config.TokenRefreshBefore
	if refreshInterval == 0 {
		refreshInterval = 10 * time.Minute
	}

	am.refreshTask = time.NewTicker(refreshInterval)
	defer am.refreshTask.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-am.refreshTask.C:
			// Check if token needs refresh
			am.mu.RLock()
			needsRefresh := time.Until(am.expiresAt) < am.config.TokenRefreshBefore
			am.mu.RUnlock()

			if needsRefresh {
				if err := am.refreshToken(); err != nil {
					log.Error().Err(err).Msg("failed to refresh token")
				}
			}
		}
	}
}
