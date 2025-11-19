package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrOrderNotFound is returned when an order is not found
	ErrOrderNotFound = errors.New("order not found")

	// ErrInvalidOrderStatus is returned when an invalid status is provided
	ErrInvalidOrderStatus = errors.New("invalid order status")

	// ErrDuplicateOrder is returned when attempting to create a duplicate order
	ErrDuplicateOrder = errors.New("duplicate order")
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusSent      OrderStatus = "SENT"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// Order represents a trading order
type Order struct {
	ID             string       `json:"id"`
	UserID         string       `json:"user_id"`
	StrategyID     *string      `json:"strategy_id,omitempty"`
	Symbol         string       `json:"symbol"`
	Side           OrderSide    `json:"side"`
	OrderType      OrderType    `json:"order_type"`
	Price          *float64     `json:"price,omitempty"`
	Quantity       int          `json:"quantity"`
	Status         OrderStatus  `json:"status"`
	BrokerOrderID  *string      `json:"broker_order_id,omitempty"`
	FilledPrice    *float64     `json:"filled_price,omitempty"`
	FilledQuantity *int         `json:"filled_quantity,omitempty"`
	RejectReason   *string      `json:"reject_reason,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// OrderAuditLog represents an audit log entry for an order
type OrderAuditLog struct {
	ID        string      `json:"id"`
	OrderID   string      `json:"order_id"`
	Status    OrderStatus `json:"status"`
	Reason    *string     `json:"reason,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	// CreateOrder creates a new order in the database
	CreateOrder(ctx context.Context, order *Order) error

	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, id string) (*Order, error)

	// GetUserOrders retrieves orders for a specific user with pagination
	GetUserOrders(ctx context.Context, userID string, limit int) ([]*Order, error)

	// UpdateOrderStatus updates the status of an order and creates an audit log entry
	UpdateOrderStatus(ctx context.Context, id string, status OrderStatus, reason string) error

	// UpdateOrderExecution updates order execution details (broker ID, filled price/quantity)
	UpdateOrderExecution(ctx context.Context, id string, brokerOrderID string, filledPrice float64, filledQuantity int) error

	// GetOrdersByStatus retrieves orders with a specific status
	GetOrdersByStatus(ctx context.Context, status OrderStatus, limit int) ([]*Order, error)

	// GetOrderAuditLog retrieves the audit log for a specific order
	GetOrderAuditLog(ctx context.Context, orderID string) ([]*OrderAuditLog, error)

	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, id string, reason string) error

	// GetPendingOrdersForUser retrieves all pending orders for a user
	GetPendingOrdersForUser(ctx context.Context, userID string) ([]*Order, error)
}

// postgresOrderRepository implements OrderRepository using PostgreSQL
type postgresOrderRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderRepository creates a new PostgreSQL-backed order repository
func NewPostgresOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &postgresOrderRepository{
		pool: pool,
	}
}

// CreateOrder creates a new order in the database
func (r *postgresOrderRepository) CreateOrder(ctx context.Context, order *Order) error {
	query := `
		INSERT INTO orders (
			id, user_id, strategy_id, symbol, side, order_type,
			price, quantity, status
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8
		)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		order.UserID, order.StrategyID, order.Symbol, order.Side, order.OrderType,
		order.Price, order.Quantity, order.Status,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by ID
func (r *postgresOrderRepository) GetOrder(ctx context.Context, id string) (*Order, error) {
	query := `
		SELECT id, user_id, strategy_id, symbol, side, order_type,
		       price, quantity, status, broker_order_id,
		       filled_price, filled_quantity, reject_reason,
		       created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	order := &Order{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.StrategyID, &order.Symbol,
		&order.Side, &order.OrderType, &order.Price, &order.Quantity,
		&order.Status, &order.BrokerOrderID, &order.FilledPrice,
		&order.FilledQuantity, &order.RejectReason,
		&order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// GetUserOrders retrieves orders for a specific user with pagination
func (r *postgresOrderRepository) GetUserOrders(ctx context.Context, userID string, limit int) ([]*Order, error) {
	query := `
		SELECT id, user_id, strategy_id, symbol, side, order_type,
		       price, quantity, status, broker_order_id,
		       filled_price, filled_quantity, reject_reason,
		       created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// UpdateOrderStatus updates the status of an order
func (r *postgresOrderRepository) UpdateOrderStatus(ctx context.Context, id string, status OrderStatus, reason string) error {
	query := `
		UPDATE orders
		SET status = $1, reject_reason = $2, updated_at = NOW()
		WHERE id = $3
	`

	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	result, err := r.pool.Exec(ctx, query, status, reasonPtr, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// UpdateOrderExecution updates order execution details
func (r *postgresOrderRepository) UpdateOrderExecution(ctx context.Context, id string, brokerOrderID string, filledPrice float64, filledQuantity int) error {
	query := `
		UPDATE orders
		SET broker_order_id = $1,
		    filled_price = $2,
		    filled_quantity = $3,
		    status = CASE
		        WHEN filled_quantity >= quantity THEN 'FILLED'
		        ELSE status
		    END,
		    updated_at = NOW()
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query, brokerOrderID, filledPrice, filledQuantity, id)
	if err != nil {
		return fmt.Errorf("failed to update order execution: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// GetOrdersByStatus retrieves orders with a specific status
func (r *postgresOrderRepository) GetOrdersByStatus(ctx context.Context, status OrderStatus, limit int) ([]*Order, error) {
	query := `
		SELECT id, user_id, strategy_id, symbol, side, order_type,
		       price, quantity, status, broker_order_id,
		       filled_price, filled_quantity, reject_reason,
		       created_at, updated_at
		FROM orders
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders by status: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// GetOrderAuditLog retrieves the audit log for a specific order
func (r *postgresOrderRepository) GetOrderAuditLog(ctx context.Context, orderID string) ([]*OrderAuditLog, error) {
	query := `
		SELECT id, order_id, status, reason, created_at
		FROM order_audit_log
		WHERE order_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	var logs []*OrderAuditLog
	for rows.Next() {
		log := &OrderAuditLog{}
		err := rows.Scan(&log.ID, &log.OrderID, &log.Status, &log.Reason, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return logs, nil
}

// CancelOrder cancels an order
func (r *postgresOrderRepository) CancelOrder(ctx context.Context, id string, reason string) error {
	return r.UpdateOrderStatus(ctx, id, OrderStatusCancelled, reason)
}

// GetPendingOrdersForUser retrieves all pending orders for a user
func (r *postgresOrderRepository) GetPendingOrdersForUser(ctx context.Context, userID string) ([]*Order, error) {
	query := `
		SELECT id, user_id, strategy_id, symbol, side, order_type,
		       price, quantity, status, broker_order_id,
		       filled_price, filled_quantity, reject_reason,
		       created_at, updated_at
		FROM orders
		WHERE user_id = $1 AND status IN ('PENDING', 'SENT')
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// scanOrders is a helper function to scan multiple order rows
func (r *postgresOrderRepository) scanOrders(rows pgx.Rows) ([]*Order, error) {
	var orders []*Order
	for rows.Next() {
		order := &Order{}
		err := rows.Scan(
			&order.ID, &order.UserID, &order.StrategyID, &order.Symbol,
			&order.Side, &order.OrderType, &order.Price, &order.Quantity,
			&order.Status, &order.BrokerOrderID, &order.FilledPrice,
			&order.FilledQuantity, &order.RejectReason,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	return orders, nil
}
