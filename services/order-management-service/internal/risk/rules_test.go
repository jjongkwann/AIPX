package risk

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jjongkwann/aipx/services/order-management-service/internal/repository"
	"github.com/jjongkwann/aipx/services/order-management-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxOrderValueRule(t *testing.T) {
	tests := []struct {
		name        string
		maxValue    float64
		order       *repository.Order
		expectError bool
	}{
		{
			name:     "Order below limit",
			maxValue: 1000000.0,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 50000.0
				order.Price = &price
				order.Quantity = 10 // 500,000
				return order
			}(),
			expectError: false,
		},
		{
			name:     "Order at limit",
			maxValue: 500000.0,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 50000.0
				order.Price = &price
				order.Quantity = 10 // 500,000
				return order
			}(),
			expectError: false,
		},
		{
			name:     "Order above limit",
			maxValue: 100000.0,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 50000.0
				order.Price = &price
				order.Quantity = 10 // 500,000
				return order
			}(),
			expectError: true,
		},
		{
			name:     "Market order skipped",
			maxValue: 100000.0,
			order: func() *repository.Order {
				order := testutil.CreateTestMarketOrder()
				order.Quantity = 100
				return order
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewMaxOrderValueRule(tt.maxValue, true)
			err := rule.Validate(context.Background(), tt.order)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPriceDeviationRule(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	tests := []struct {
		name           string
		maxDeviation   float64
		order          *repository.Order
		marketPrice    float64
		setMarketPrice bool
		expectError    bool
	}{
		{
			name:         "Price within deviation",
			maxDeviation: 0.05,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 52000.0
				order.Price = &price
				return order
			}(),
			marketPrice:    50000.0,
			setMarketPrice: true,
			expectError:    false,
		},
		{
			name:         "Price at max deviation",
			maxDeviation: 0.05,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 52500.0
				order.Price = &price
				return order
			}(),
			marketPrice:    50000.0,
			setMarketPrice: true,
			expectError:    false,
		},
		{
			name:         "Price exceeds deviation",
			maxDeviation: 0.05,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 60000.0
				order.Price = &price
				return order
			}(),
			marketPrice:    50000.0,
			setMarketPrice: true,
			expectError:    true,
		},
		{
			name:         "No market price available",
			maxDeviation: 0.05,
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				price := 60000.0
				order.Price = &price
				return order
			}(),
			setMarketPrice: false,
			expectError:    false,
		},
		{
			name:         "Market order skipped",
			maxDeviation: 0.05,
			order:        testutil.CreateTestMarketOrder(),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redis.Reset()

			if tt.setMarketPrice {
				key := fmt.Sprintf("market_price:%s", tt.order.Symbol)
				err := redis.Client.Set(ctx, key, strconv.FormatFloat(tt.marketPrice, 'f', 2, 64), 0).Err()
				require.NoError(t, err)
			}

			rule := NewPriceDeviationRule(redis.Client, tt.maxDeviation, true)
			err := rule.Validate(ctx, tt.order)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDuplicateOrderRule(t *testing.T) {
	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	t.Run("First order allowed", func(t *testing.T) {
		redis.Reset()
		rule := NewDuplicateOrderRule(redis.Client, 10, true)
		order := testutil.CreateTestOrder()

		err := rule.Validate(ctx, order)
		assert.NoError(t, err)
	})

	t.Run("Duplicate order rejected", func(t *testing.T) {
		redis.Reset()
		rule := NewDuplicateOrderRule(redis.Client, 10, true)
		order := testutil.CreateTestOrder()

		// First order
		err := rule.Validate(ctx, order)
		assert.NoError(t, err)

		// Duplicate order
		err = rule.Validate(ctx, order)
		assert.Error(t, err)
	})

	t.Run("Order allowed after window expires", func(t *testing.T) {
		redis.Reset()
		rule := NewDuplicateOrderRule(redis.Client, 1, true) // 1 second window
		order := testutil.CreateTestOrder()

		// First order
		err := rule.Validate(ctx, order)
		assert.NoError(t, err)

		// Wait for window to expire
		time.Sleep(1100 * time.Millisecond)

		// Second order should be allowed
		err = rule.Validate(ctx, order)
		assert.NoError(t, err)
	})

	t.Run("Different orders allowed", func(t *testing.T) {
		redis.Reset()
		rule := NewDuplicateOrderRule(redis.Client, 10, true)

		order1 := testutil.CreateTestOrder()
		order2 := testutil.CreateTestOrder()

		err := rule.Validate(ctx, order1)
		assert.NoError(t, err)

		err = rule.Validate(ctx, order2)
		assert.NoError(t, err)
	})
}

func TestAllowedSymbolRule(t *testing.T) {
	tests := []struct {
		name           string
		allowedSymbols []string
		order          *repository.Order
		expectError    bool
	}{
		{
			name:           "Allowed symbol",
			allowedSymbols: []string{"005930", "035720"},
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				order.Symbol = "005930"
				return order
			}(),
			expectError: false,
		},
		{
			name:           "Disallowed symbol",
			allowedSymbols: []string{"005930", "035720"},
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				order.Symbol = "999999"
				return order
			}(),
			expectError: true,
		},
		{
			name:           "Wildcard match",
			allowedSymbols: []string{"005*"},
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				order.Symbol = "005930"
				return order
			}(),
			expectError: false,
		},
		{
			name:           "Wildcard no match",
			allowedSymbols: []string{"005*"},
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				order.Symbol = "035720"
				return order
			}(),
			expectError: true,
		},
		{
			name:           "Empty allowed list allows all",
			allowedSymbols: []string{},
			order: func() *repository.Order {
				order := testutil.CreateTestOrder()
				order.Symbol = "999999"
				return order
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewAllowedSymbolRule(tt.allowedSymbols, true)
			err := rule.Validate(context.Background(), tt.order)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDailyLossLimitRule(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	ctx := context.Background()

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)

	t.Run("Loss within limit", func(t *testing.T) {
		redis.Reset()
		rule := NewDailyLossLimitRule(repo, redis.Client, 100000.0, true)
		order := testutil.CreateTestOrder()

		err := rule.Validate(ctx, order)
		assert.NoError(t, err)
	})

	t.Run("Caching works", func(t *testing.T) {
		redis.Reset()
		rule := NewDailyLossLimitRule(repo, redis.Client, 100000.0, true)
		order := testutil.CreateTestOrder()

		// First call
		err := rule.Validate(ctx, order)
		assert.NoError(t, err)

		// Second call should hit cache
		err = rule.Validate(ctx, order)
		assert.NoError(t, err)

		// Verify cache exists
		cacheKey := fmt.Sprintf("daily_pnl:%s:%s", order.UserID, time.Now().Format("2006-01-02"))
		exists := redis.Client.Exists(ctx, cacheKey).Val()
		assert.Equal(t, int64(1), exists)
	})
}
