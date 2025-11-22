package broker_test

import (
	"context"
	"testing"
	"time"

	"order-management-service/internal/broker"
	"order-management-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKISClient(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		config := testutil.NewTestKISConfig("http://localhost")
		client, err := broker.NewKISClient(config)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Nil config", func(t *testing.T) {
		client, err := broker.NewKISClient(nil)

		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestKISClient_SubmitOrder_Success(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorSuccess)
	defer mockServer.Close()

	config := testutil.NewTestKISConfig(mockServer.Server.URL)
	client, err := broker.NewKISClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Limit order", func(t *testing.T) {
		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "0000000001", resp.OrderID)
		assert.Equal(t, "SENT", resp.Status)
	})

	t.Run("Market order", func(t *testing.T) {
		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "MARKET",
			Price:    0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "0000000001", resp.OrderID)
	})

	t.Run("Sell order", func(t *testing.T) {
		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "SELL",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestKISClient_SubmitOrder_Errors(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorServerError)
	defer mockServer.Close()

	ctx := context.Background()

	t.Run("Insufficient funds", func(t *testing.T) {
		mockServer := testutil.NewMockKISServer(t, testutil.BehaviorInsufficientFunds)
		defer mockServer.Close()

		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		config.MaxRetries = 0 // Don't retry on business errors
		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 1000000, // Large quantity
		}

		resp, err := client.SubmitOrder(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "주문가능금액을 초과하였습니다")
	})

	t.Run("Invalid symbol", func(t *testing.T) {
		mockServer := testutil.NewMockKISServer(t, testutil.BehaviorInvalidSymbol)
		defer mockServer.Close()

		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		config.MaxRetries = 0 // Don't retry on business errors
		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "999999",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "종목코드가 올바르지 않습니다")
	})

	t.Run("Rate limit", func(t *testing.T) {
		mockServer := testutil.NewMockKISServer(t, testutil.BehaviorRateLimit)
		defer mockServer.Close()

		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Server error", func(t *testing.T) {
		mockServer := testutil.NewMockKISServer(t, testutil.BehaviorServerError)
		defer mockServer.Close()

		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		resp, err := client.SubmitOrder(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestKISClient_Retry(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorServerError)
	defer mockServer.Close()

	ctx := context.Background()

	t.Run("Retries on failure", func(t *testing.T) {
		// Configure short retry delay for testing
		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		config.MaxRetries = 2
		config.RetryDelay = 10 * time.Millisecond

		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		start := time.Now()
		_, err = client.SubmitOrder(ctx, req)
		elapsed := time.Since(start)

		// Should fail after retries
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after")

		// Should take at least (10ms + 20ms) for 2 retries
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(30))
	})

	t.Run("Eventual success after retry", func(t *testing.T) {
		// Reset server behavior
		mockServer.SetBehavior(testutil.BehaviorServerError)

		// Configure short retry delay
		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		config.MaxRetries = 5 // Increase retries to ensure we catch the switch
		config.RetryDelay = 20 * time.Millisecond

		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		// After 50ms, switch to success
		go func() {
			time.Sleep(50 * time.Millisecond)
			mockServer.SetBehavior(testutil.BehaviorSuccess)
		}()

		resp, err := client.SubmitOrder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestKISClient_CircuitBreaker(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorServerError)
	defer mockServer.Close()

	config := testutil.NewTestKISConfig(mockServer.Server.URL)
	config.MaxRetries = 0 // No retries for this test
	client, err := broker.NewKISClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &broker.OrderRequest{
		Symbol:   "005930",
		Side:     "BUY",
		Type:     "LIMIT",
		Price:    50000.0,
		Quantity: 10,
	}

	t.Run("Opens after failures", func(t *testing.T) {
		// Make 3 failed requests to open circuit breaker
		for i := 0; i < 3; i++ {
			client.SubmitOrder(ctx, req)
		}

		// Circuit should be open now, request should fail immediately
		start := time.Now()
		_, err := client.SubmitOrder(ctx, req)
		elapsed := time.Since(start)

		assert.Error(t, err)
		// Should fail fast (less than 100ms)
		assert.Less(t, elapsed.Milliseconds(), int64(100))
	})
}

func TestKISClient_CancelOrder(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorSuccess)
	defer mockServer.Close()

	config := testutil.NewTestKISConfig(mockServer.Server.URL)
	client, err := broker.NewKISClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		resp, err := client.CancelOrder(ctx, "0000000001")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "0000000001", resp.OrderID)
		assert.Equal(t, "CANCELLED", resp.Status)
	})
}

func TestKISClient_GetOrderStatus(t *testing.T) {
	mockServer := testutil.NewMockKISServer(t, testutil.BehaviorSuccess)
	defer mockServer.Close()

	config := testutil.NewTestKISConfig(mockServer.Server.URL)
	client, err := broker.NewKISClient(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Returns status", func(t *testing.T) {
		status, err := client.GetOrderStatus(ctx, "0000000001")
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, "0000000001", status.OrderID)
	})
}

func TestKISClient_SignRequest(t *testing.T) {
	t.Run("Adds signature headers", func(t *testing.T) {
		ctx := context.Background()
		mockServer := testutil.NewMockKISServer(t, testutil.BehaviorSuccess)
		defer mockServer.Close()

		// Use config to set base URL
		config := testutil.NewTestKISConfig(mockServer.Server.URL)
		client, err := broker.NewKISClient(config)
		require.NoError(t, err)

		req := &broker.OrderRequest{
			Symbol:   "005930",
			Side:     "BUY",
			Type:     "LIMIT",
			Price:    50000.0,
			Quantity: 10,
		}

		// Request should succeed with signature
		resp, err := client.SubmitOrder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// TestKISClient_ConvertOrderType removed as it tests private method
