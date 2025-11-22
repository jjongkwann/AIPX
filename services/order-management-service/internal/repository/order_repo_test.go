package repository_test

import (
	"context"
	"testing"
	"time"

	"order-management-service/internal/repository"
	"order-management-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateOrder(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)

		require.NoError(t, err)
		assert.NotEmpty(t, order.ID)
		assert.False(t, order.CreatedAt.IsZero())
		assert.False(t, order.UpdatedAt.IsZero())
	})

	t.Run("With strategy ID", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		strategyID := "strategy_123"
		order.StrategyID = &strategyID

		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)
		assert.NotEmpty(t, order.ID)
	})
}

func TestRepository_GetOrder(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		// Create order
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		// Get order
		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, order.ID, retrieved.ID)
		assert.Equal(t, order.UserID, retrieved.UserID)
		assert.Equal(t, order.Symbol, retrieved.Symbol)
		assert.Equal(t, order.Side, retrieved.Side)
		assert.Equal(t, order.Quantity, retrieved.Quantity)
	})

	t.Run("Not found", func(t *testing.T) {
		retrieved, err := repo.GetOrder(ctx, "00000000-0000-0000-0000-000000000000")
		assert.Error(t, err)
		assert.Equal(t, repository.ErrOrderNotFound, err)
		assert.Nil(t, retrieved)
	})
}

func TestRepository_GetUserOrders(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Returns user orders", func(t *testing.T) {
		// Create multiple orders for user
		for i := 0; i < 3; i++ {
			order := testutil.CreateTestOrder()
			order.UserID = "test_user_1"
			err := repo.CreateOrder(ctx, order)
			require.NoError(t, err)
		}

		// Create order for different user
		otherOrder := testutil.CreateTestOrder()
		otherOrder.UserID = "test_user_2"
		repo.CreateOrder(ctx, otherOrder)

		// Get orders
		orders, err := repo.GetUserOrders(ctx, "test_user_1", 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 3)

		// Verify all orders belong to user
		for _, order := range orders {
			assert.Equal(t, "test_user_1", order.UserID)
		}
	})

	t.Run("Respects limit", func(t *testing.T) {
		orders, err := repo.GetUserOrders(ctx, "test_user_1", 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(orders), 2)
	})

	t.Run("Orders sorted by created_at DESC", func(t *testing.T) {
		orders, err := repo.GetUserOrders(ctx, "test_user_1", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(orders), 2)

		// Verify descending order
		for i := 0; i < len(orders)-1; i++ {
			assert.True(t, orders[i].CreatedAt.After(orders[i+1].CreatedAt) ||
				orders[i].CreatedAt.Equal(orders[i+1].CreatedAt))
		}
	})
}

func TestRepository_UpdateOrderStatus(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		err = repo.UpdateOrderStatus(ctx, order.ID, repository.OrderStatusSent, "Order sent to broker")
		require.NoError(t, err)

		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, repository.OrderStatusSent, retrieved.Status)
	})

	t.Run("Not found", func(t *testing.T) {
		err := repo.UpdateOrderStatus(ctx, "00000000-0000-0000-0000-000000000000", repository.OrderStatusSent, "")
		assert.Error(t, err)
		assert.Equal(t, repository.ErrOrderNotFound, err)
	})

	t.Run("Updates reject reason", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		reason := "Insufficient funds"
		err = repo.UpdateOrderStatus(ctx, order.ID, repository.OrderStatusRejected, reason)
		require.NoError(t, err)

		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, repository.OrderStatusRejected, retrieved.Status)
		require.NotNil(t, retrieved.RejectReason)
		assert.Equal(t, reason, *retrieved.RejectReason)
	})
}

func TestRepository_UpdateOrderExecution(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		brokerOrderID := "BROKER_001"
		filledPrice := 50000.0
		filledQuantity := 10

		err = repo.UpdateOrderExecution(ctx, order.ID, brokerOrderID, filledPrice, filledQuantity)
		require.NoError(t, err)

		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.BrokerOrderID)
		assert.Equal(t, brokerOrderID, *retrieved.BrokerOrderID)
		require.NotNil(t, retrieved.FilledPrice)
		assert.Equal(t, filledPrice, *retrieved.FilledPrice)
		require.NotNil(t, retrieved.FilledQuantity)
		assert.Equal(t, filledQuantity, *retrieved.FilledQuantity)
	})

	t.Run("Auto fills when quantity matches", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		order.Quantity = 10
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		err = repo.UpdateOrderExecution(ctx, order.ID, "BROKER_002", 50000.0, 10)
		require.NoError(t, err)

		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, repository.OrderStatusFilled, retrieved.Status)
	})
}

func TestRepository_GetOrdersByStatus(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Returns orders with status", func(t *testing.T) {
		// Create orders with different statuses
		pendingOrder := testutil.CreateTestOrder()
		repo.CreateOrder(ctx, pendingOrder)

		sentOrder := testutil.CreateTestOrder()
		repo.CreateOrder(ctx, sentOrder)
		repo.UpdateOrderStatus(ctx, sentOrder.ID, repository.OrderStatusSent, "")

		// Query pending orders
		pendingOrders, err := repo.GetOrdersByStatus(ctx, repository.OrderStatusPending, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pendingOrders), 1)

		// Query sent orders
		sentOrders, err := repo.GetOrdersByStatus(ctx, repository.OrderStatusSent, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(sentOrders), 1)
	})
}

func TestRepository_CancelOrder(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		order := testutil.CreateTestOrder()
		err := repo.CreateOrder(ctx, order)
		require.NoError(t, err)

		reason := "User requested cancellation"
		err = repo.CancelOrder(ctx, order.ID, reason)
		require.NoError(t, err)

		retrieved, err := repo.GetOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, repository.OrderStatusCancelled, retrieved.Status)
		require.NotNil(t, retrieved.RejectReason)
		assert.Equal(t, reason, *retrieved.RejectReason)
	})
}

func TestRepository_GetPendingOrdersForUser(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Returns pending and sent orders", func(t *testing.T) {
		userID := "test_user_pending"

		// Create pending order
		pendingOrder := testutil.CreateTestOrder()
		pendingOrder.UserID = userID
		repo.CreateOrder(ctx, pendingOrder)

		// Create sent order
		sentOrder := testutil.CreateTestOrder()
		sentOrder.UserID = userID
		repo.CreateOrder(ctx, sentOrder)
		repo.UpdateOrderStatus(ctx, sentOrder.ID, repository.OrderStatusSent, "")

		// Create filled order (should not be returned)
		filledOrder := testutil.CreateTestOrder()
		filledOrder.UserID = userID
		repo.CreateOrder(ctx, filledOrder)
		repo.UpdateOrderStatus(ctx, filledOrder.ID, repository.OrderStatusFilled, "")

		orders, err := repo.GetPendingOrdersForUser(ctx, userID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orders), 2)

		// Verify only pending/sent orders
		for _, order := range orders {
			assert.Contains(t, []repository.OrderStatus{repository.OrderStatusPending, repository.OrderStatusSent}, order.Status)
		}
	})
}

func TestRepository_Concurrency(t *testing.T) {
	pgContainer := testutil.NewTestDB(t)
	defer pgContainer.Close(t)

	repo := repository.NewPostgresOrderRepository(pgContainer.Pool)
	ctx := context.Background()

	t.Run("Concurrent creates", func(t *testing.T) {
		const numOrders = 10

		done := make(chan bool, numOrders)
		for i := 0; i < numOrders; i++ {
			go func() {
				order := testutil.CreateTestOrder()
				err := repo.CreateOrder(ctx, order)
				assert.NoError(t, err)
				done <- true
			}()
		}

		timeout := time.After(5 * time.Second)
		for i := 0; i < numOrders; i++ {
			select {
			case <-done:
			case <-timeout:
				t.Fatal("Timeout waiting for concurrent creates")
			}
		}
	})
}
