package risk

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/jjongkwann/aipx/services/order-management-service/internal/repository"
	"github.com/jjongkwann/aipx/services/order-management-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)

	t.Run("Success with all dependencies", func(t *testing.T) {
		config := &Config{
			MaxOrderValue:    1000000.0,
			PriceDeviation:   0.05,
			DailyLossLimit:   100000.0,
			AllowedSymbols:   []string{"005930", "035720", "000660"},
			DuplicateWindow:  10,
			EnableRiskChecks: true,
		}
		deps := &Dependencies{
			Repository:  repo,
			RedisClient: redis.Client,
		}

		engine, err := NewEngine(config, deps)
		require.NoError(t, err)
		assert.NotNil(t, engine)
		assert.True(t, engine.IsEnabled())
		assert.Len(t, engine.GetRules(), 5)
	})

	t.Run("Nil config error", func(t *testing.T) {
		deps := &Dependencies{
			Repository:  repo,
			RedisClient: redis.Client,
		}

		engine, err := NewEngine(nil, deps)
		assert.Error(t, err)
		assert.Nil(t, engine)
	})

	t.Run("Engine with disabled checks", func(t *testing.T) {
		config := &Config{
			MaxOrderValue:    1000000.0,
			PriceDeviation:   0.05,
			DailyLossLimit:   100000.0,
			AllowedSymbols:   []string{"005930", "035720", "000660"},
			DuplicateWindow:  10,
			EnableRiskChecks: false,
		}
		deps := &Dependencies{
			Repository:  repo,
			RedisClient: redis.Client,
		}

		engine, err := NewEngine(config, deps)
		require.NoError(t, err)
		assert.False(t, engine.IsEnabled())
	})
}

func TestEngine_Validate(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	config := &Config{
		MaxOrderValue:    1000000.0,
		PriceDeviation:   0.05,
		DailyLossLimit:   100000.0,
		AllowedSymbols:   []string{"005930", "035720", "000660"},
		DuplicateWindow:  10,
		EnableRiskChecks: true,
	}
	deps := &Dependencies{
		Repository:  repo,
		RedisClient: redis.Client,
	}

	ctx := context.Background()

	t.Run("All rules pass", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		order := testutil.CreateTestOrder()
		price := 50000.0
		order.Price = &price
		order.Quantity = 10

		// Set market price
		marketKey := fmt.Sprintf("market_price:%s", order.Symbol)
		redis.Client.Set(ctx, marketKey, strconv.FormatFloat(price, 'f', 2, 64), 0)

		err = engine.Validate(ctx, order)
		assert.NoError(t, err)
	})

	t.Run("First rule fails", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		order := testutil.CreateHighValueOrder()

		err = engine.Validate(ctx, order)
		assert.Error(t, err)
		assert.IsType(t, &RiskError{}, err)

		riskErr := err.(*RiskError)
		assert.Equal(t, "MaxOrderValue", riskErr.Rule)
	})

	t.Run("Price deviation rule fails", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		order := testutil.CreateTestOrder()
		price := 60000.0
		order.Price = &price
		order.Quantity = 10

		// Set market price significantly lower
		marketKey := fmt.Sprintf("market_price:%s", order.Symbol)
		redis.Client.Set(ctx, marketKey, "50000.0", 0)

		err = engine.Validate(ctx, order)
		assert.Error(t, err)
		assert.IsType(t, &RiskError{}, err)

		riskErr := err.(*RiskError)
		assert.Equal(t, "PriceDeviation", riskErr.Rule)
	})

	t.Run("Duplicate order fails", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		order := testutil.CreateTestOrder()

		// Set market price
		marketKey := fmt.Sprintf("market_price:%s", order.Symbol)
		redis.Client.Set(ctx, marketKey, "50000.0", 0)

		// First order should pass
		err = engine.Validate(ctx, order)
		assert.NoError(t, err)

		// Duplicate order should fail
		err = engine.Validate(ctx, order)
		assert.Error(t, err)
		assert.IsType(t, &RiskError{}, err)

		riskErr := err.(*RiskError)
		assert.Equal(t, "DuplicateOrder", riskErr.Rule)
	})

	t.Run("Disallowed symbol fails", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		order := testutil.CreateDisallowedSymbolOrder()

		err = engine.Validate(ctx, order)
		assert.Error(t, err)
		assert.IsType(t, &RiskError{}, err)

		riskErr := err.(*RiskError)
		assert.Equal(t, "AllowedSymbol", riskErr.Rule)
	})

	t.Run("Disabled engine skips validation", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		engine.Disable()
		assert.False(t, engine.IsEnabled())

		// Even invalid order should pass
		order := testutil.CreateHighValueOrder()
		err = engine.Validate(ctx, order)
		assert.NoError(t, err)
	})
}

func TestEngine_ConcurrentValidation(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	config := &Config{
		MaxOrderValue:    1000000.0,
		PriceDeviation:   0.05,
		DailyLossLimit:   100000.0,
		AllowedSymbols:   []string{"005930", "035720", "000660"},
		DuplicateWindow:  10,
		EnableRiskChecks: true,
	}
	deps := &Dependencies{
		Repository:  repo,
		RedisClient: redis.Client,
	}

	ctx := context.Background()

	t.Run("Concurrent validations", func(t *testing.T) {
		redis.Reset()
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		// Set market prices
		for _, symbol := range []string{"005930", "035720", "000660"} {
			marketKey := fmt.Sprintf("market_price:%s", symbol)
			redis.Client.Set(ctx, marketKey, "50000.0", 0)
		}

		var wg sync.WaitGroup
		errors := make([]error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				order := testutil.CreateTestOrder()
				order.Symbol = config.AllowedSymbols[idx%len(config.AllowedSymbols)]

				errors[idx] = engine.Validate(ctx, order)
			}(i)
		}

		wg.Wait()

		// Count successful validations
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}

		// At least some should succeed
		assert.Greater(t, successCount, 0)
	})
}

func TestEngine_RuleManagement(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	redis := testutil.NewTestRedis(t)
	defer redis.Close()

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	config := &Config{
		MaxOrderValue:    1000000.0,
		PriceDeviation:   0.05,
		DailyLossLimit:   100000.0,
		AllowedSymbols:   []string{"005930", "035720", "000660"},
		DuplicateWindow:  10,
		EnableRiskChecks: true,
	}
	deps := &Dependencies{
		Repository:  repo,
		RedisClient: redis.Client,
	}

	t.Run("Add custom rule", func(t *testing.T) {
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		initialCount := len(engine.GetRules())

		customRule := &mockRule{name: "CustomRule"}
		engine.AddRule(customRule)

		assert.Len(t, engine.GetRules(), initialCount+1)
	})

	t.Run("Remove rule", func(t *testing.T) {
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		initialCount := len(engine.GetRules())

		engine.RemoveRule("MaxOrderValue")

		assert.Len(t, engine.GetRules(), initialCount-1)
	})

	t.Run("Enable/Disable engine", func(t *testing.T) {
		engine, err := NewEngine(config, deps)
		require.NoError(t, err)

		assert.True(t, engine.IsEnabled())

		engine.Disable()
		assert.False(t, engine.IsEnabled())

		engine.Enable()
		assert.True(t, engine.IsEnabled())
	})
}

func TestRiskError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := &RiskError{
			Rule:    "TestRule",
			Message: "test error message",
			Order:   order,
		}

		assert.Equal(t, "risk check failed [TestRule]: test error message", err.Error())
	})
}

// mockRule is a mock implementation of RiskRule for testing
type mockRule struct {
	name    string
	enabled bool
	err     error
}

func (m *mockRule) Name() string {
	return m.name
}

func (m *mockRule) Enabled() bool {
	return m.enabled
}

func (m *mockRule) Validate(ctx context.Context, order *repository.Order) error {
	return m.err
}
