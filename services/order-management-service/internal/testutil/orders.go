package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/jjongkwann/aipx/services/order-management-service/internal/repository"
)

// CreateTestOrder creates a sample order for testing
func CreateTestOrder() *repository.Order {
	price := 50000.0
	return &repository.Order{
		ID:         uuid.New().String(),
		UserID:     "test_user_1",
		Symbol:     "005930",
		Side:       repository.OrderSideBuy,
		OrderType:  repository.OrderTypeLimit,
		Price:      &price,
		Quantity:   10,
		Status:     repository.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CreateTestMarketOrder creates a market order for testing
func CreateTestMarketOrder() *repository.Order {
	return &repository.Order{
		ID:         uuid.New().String(),
		UserID:     "test_user_1",
		Symbol:     "005930",
		Side:       repository.OrderSideBuy,
		OrderType:  repository.OrderTypeMarket,
		Quantity:   10,
		Status:     repository.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CreateTestSellOrder creates a sell order for testing
func CreateTestSellOrder() *repository.Order {
	price := 50000.0
	return &repository.Order{
		ID:         uuid.New().String(),
		UserID:     "test_user_1",
		Symbol:     "005930",
		Side:       repository.OrderSideSell,
		OrderType:  repository.OrderTypeLimit,
		Price:      &price,
		Quantity:   10,
		Status:     repository.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CreateHighValueOrder creates an order with high value for testing
func CreateHighValueOrder() *repository.Order {
	price := 200000.0
	return &repository.Order{
		ID:         uuid.New().String(),
		UserID:     "test_user_1",
		Symbol:     "005930",
		Side:       repository.OrderSideBuy,
		OrderType:  repository.OrderTypeLimit,
		Price:      &price,
		Quantity:   10,
		Status:     repository.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CreateDisallowedSymbolOrder creates an order with disallowed symbol
func CreateDisallowedSymbolOrder() *repository.Order {
	price := 10000.0
	return &repository.Order{
		ID:         uuid.New().String(),
		UserID:     "test_user_1",
		Symbol:     "999999",
		Side:       repository.OrderSideBuy,
		OrderType:  repository.OrderTypeLimit,
		Price:      &price,
		Quantity:   10,
		Status:     repository.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
