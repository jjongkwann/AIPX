package risk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"order-management-service/internal/repository"
)

// MaxOrderValueRule checks if order value exceeds maximum allowed
type MaxOrderValueRule struct {
	maxValue float64
	enabled  bool
}

// NewMaxOrderValueRule creates a new max order value rule
func NewMaxOrderValueRule(maxValue float64, enabled bool) *MaxOrderValueRule {
	return &MaxOrderValueRule{
		maxValue: maxValue,
		enabled:  enabled,
	}
}

func (r *MaxOrderValueRule) Name() string {
	return "MaxOrderValue"
}

func (r *MaxOrderValueRule) Enabled() bool {
	return r.enabled
}

func (r *MaxOrderValueRule) Validate(ctx context.Context, order *repository.Order) error {
	// For market orders, we can't calculate exact value
	if order.OrderType == repository.OrderTypeMarket {
		log.Debug().
			Str("order_id", order.ID).
			Msg("Skipping max value check for market order")
		return nil
	}

	if order.Price == nil {
		return fmt.Errorf("price is required for limit orders")
	}

	orderValue := *order.Price * float64(order.Quantity)
	if orderValue > r.maxValue {
		return fmt.Errorf("order value %.2f exceeds maximum allowed %.2f", orderValue, r.maxValue)
	}

	return nil
}

// PriceDeviationRule checks if price deviates too much from market price
type PriceDeviationRule struct {
	redisClient       *redis.Client
	maxDeviationRatio float64
	enabled           bool
}

// NewPriceDeviationRule creates a new price deviation rule
func NewPriceDeviationRule(redisClient *redis.Client, maxDeviationRatio float64, enabled bool) *PriceDeviationRule {
	return &PriceDeviationRule{
		redisClient:       redisClient,
		maxDeviationRatio: maxDeviationRatio,
		enabled:           enabled,
	}
}

func (r *PriceDeviationRule) Name() string {
	return "PriceDeviation"
}

func (r *PriceDeviationRule) Enabled() bool {
	return r.enabled
}

func (r *PriceDeviationRule) Validate(ctx context.Context, order *repository.Order) error {
	// Skip for market orders
	if order.OrderType == repository.OrderTypeMarket {
		log.Debug().
			Str("order_id", order.ID).
			Msg("Skipping price deviation check for market order")
		return nil
	}

	if order.Price == nil {
		return fmt.Errorf("price is required for limit orders")
	}

	// Get current market price from Redis (set by Data Ingestion Service)
	marketPriceKey := fmt.Sprintf("market_price:%s", order.Symbol)
	marketPriceStr, err := r.redisClient.Get(ctx, marketPriceKey).Result()
	if err == redis.Nil {
		// No market price available, allow the order
		log.Warn().
			Str("symbol", order.Symbol).
			Msg("No market price available, skipping price deviation check")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get market price: %w", err)
	}

	marketPrice, err := strconv.ParseFloat(marketPriceStr, 64)
	if err != nil {
		return fmt.Errorf("invalid market price format: %w", err)
	}

	// Calculate price deviation
	deviation := math.Abs(*order.Price-marketPrice) / marketPrice
	if deviation > r.maxDeviationRatio {
		return fmt.Errorf(
			"price deviation %.2f%% exceeds maximum allowed %.2f%% (order price: %.2f, market price: %.2f)",
			deviation*100,
			r.maxDeviationRatio*100,
			*order.Price,
			marketPrice,
		)
	}

	return nil
}

// DailyLossLimitRule checks if daily loss limit is exceeded
type DailyLossLimitRule struct {
	repository  repository.OrderRepository
	redisClient *redis.Client
	lossLimit   float64
	enabled     bool
}

// NewDailyLossLimitRule creates a new daily loss limit rule
func NewDailyLossLimitRule(
	repo repository.OrderRepository,
	redisClient *redis.Client,
	lossLimit float64,
	enabled bool,
) *DailyLossLimitRule {
	return &DailyLossLimitRule{
		repository:  repo,
		redisClient: redisClient,
		lossLimit:   lossLimit,
		enabled:     enabled,
	}
}

func (r *DailyLossLimitRule) Name() string {
	return "DailyLossLimit"
}

func (r *DailyLossLimitRule) Enabled() bool {
	return r.enabled
}

func (r *DailyLossLimitRule) Validate(ctx context.Context, order *repository.Order) error {
	// Check Redis cache first (5 min TTL)
	cacheKey := fmt.Sprintf("daily_pnl:%s:%s", order.UserID, time.Now().Format("2006-01-02"))
	cachedPnL, err := r.redisClient.Get(ctx, cacheKey).Result()

	var todayPnL float64
	if err == redis.Nil {
		// Cache miss, query database
		todayPnL, err = r.calculateDailyPnL(ctx, order.UserID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to calculate daily PnL")
			// Don't block order on calculation error in production
			// TODO: Add metric to track this
			return nil
		}

		// Cache the result
		r.redisClient.Set(ctx, cacheKey, fmt.Sprintf("%.2f", todayPnL), 5*time.Minute)
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to get cached PnL")
		return nil
	} else {
		todayPnL, _ = strconv.ParseFloat(cachedPnL, 64)
	}

	// Check if loss limit is exceeded
	if todayPnL < -r.lossLimit {
		return fmt.Errorf(
			"daily loss limit exceeded: current PnL %.2f is below limit -%.2f",
			todayPnL,
			r.lossLimit,
		)
	}

	return nil
}

// calculateDailyPnL calculates the profit/loss for today
func (r *DailyLossLimitRule) calculateDailyPnL(ctx context.Context, userID string) (float64, error) {
	// Get all filled orders for today
	// This is a simplified version - in production, you'd want to:
	// 1. Query from a dedicated PnL table
	// 2. Include unrealized PnL from open positions
	// 3. Use a materialized view for performance

	// For now, we'll just return 0 as this requires position tracking
	// which is typically handled by a separate Portfolio Service
	log.Debug().
		Str("user_id", userID).
		Msg("Daily PnL calculation not fully implemented, returning 0")
	return 0, nil
}

// DuplicateOrderRule prevents duplicate orders within a time window
type DuplicateOrderRule struct {
	redisClient     *redis.Client
	duplicateWindow int64 // seconds
	enabled         bool
}

// NewDuplicateOrderRule creates a new duplicate order rule
func NewDuplicateOrderRule(redisClient *redis.Client, duplicateWindow int64, enabled bool) *DuplicateOrderRule {
	return &DuplicateOrderRule{
		redisClient:     redisClient,
		duplicateWindow: duplicateWindow,
		enabled:         enabled,
	}
}

func (r *DuplicateOrderRule) Name() string {
	return "DuplicateOrder"
}

func (r *DuplicateOrderRule) Enabled() bool {
	return r.enabled
}

func (r *DuplicateOrderRule) Validate(ctx context.Context, order *repository.Order) error {
	// Generate order fingerprint
	fingerprint := r.generateFingerprint(order)
	fingerprintKey := fmt.Sprintf("order_fingerprint:%s", fingerprint)

	// Check if fingerprint exists in Redis
	exists, err := r.redisClient.Exists(ctx, fingerprintKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check duplicate order: %w", err)
	}

	if exists > 0 {
		return fmt.Errorf("duplicate order detected within %d seconds", r.duplicateWindow)
	}

	// Set fingerprint in Redis with TTL
	err = r.redisClient.Set(
		ctx,
		fingerprintKey,
		order.ID,
		time.Duration(r.duplicateWindow)*time.Second,
	).Err()
	if err != nil {
		log.Error().Err(err).Msg("Failed to set order fingerprint")
		// Don't block the order if Redis write fails
	}

	return nil
}

// generateFingerprint creates a unique fingerprint for an order
func (r *DuplicateOrderRule) generateFingerprint(order *repository.Order) string {
	var priceStr string
	if order.Price != nil {
		priceStr = fmt.Sprintf("%.2f", *order.Price)
	} else {
		priceStr = "market"
	}

	data := fmt.Sprintf(
		"%s:%s:%s:%s:%d",
		order.UserID,
		order.Symbol,
		order.Side,
		priceStr,
		order.Quantity,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// AllowedSymbolRule checks if symbol is in the allowed list
type AllowedSymbolRule struct {
	allowedSymbols []string
	enabled        bool
}

// NewAllowedSymbolRule creates a new allowed symbol rule
func NewAllowedSymbolRule(allowedSymbols []string, enabled bool) *AllowedSymbolRule {
	return &AllowedSymbolRule{
		allowedSymbols: allowedSymbols,
		enabled:        enabled,
	}
}

func (r *AllowedSymbolRule) Name() string {
	return "AllowedSymbol"
}

func (r *AllowedSymbolRule) Enabled() bool {
	return r.enabled
}

func (r *AllowedSymbolRule) Validate(ctx context.Context, order *repository.Order) error {
	// If no symbols are specified, allow all
	if len(r.allowedSymbols) == 0 {
		return nil
	}

	// Check if symbol is in allowed list (with wildcard support)
	for _, allowedSymbol := range r.allowedSymbols {
		if r.matchSymbol(order.Symbol, allowedSymbol) {
			return nil
		}
	}

	return fmt.Errorf(
		"symbol %s is not in the allowed list",
		order.Symbol,
	)
}

// matchSymbol checks if a symbol matches a pattern (with wildcard support)
func (r *AllowedSymbolRule) matchSymbol(symbol, pattern string) bool {
	// Exact match
	if symbol == pattern {
		return true
	}

	// Wildcard pattern matching (e.g., "005*" matches "005930")
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(symbol, prefix)
	}

	return false
}
