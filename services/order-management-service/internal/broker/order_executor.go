package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"

	"order-management-service/internal/repository"
)

// OrderExecutor orchestrates the order lifecycle
type OrderExecutor struct {
	kisClient  *KISClient
	repository repository.OrderRepository
	kafka      sarama.SyncProducer
	kafkaTopic string
}

// NewOrderExecutor creates a new order executor
func NewOrderExecutor(
	kisClient *KISClient,
	repo repository.OrderRepository,
	kafka sarama.SyncProducer,
	kafkaTopic string,
) *OrderExecutor {
	return &OrderExecutor{
		kisClient:  kisClient,
		repository: repo,
		kafka:      kafka,
		kafkaTopic: kafkaTopic,
	}
}

// Execute executes an order through its complete lifecycle
func (e *OrderExecutor) Execute(ctx context.Context, order *repository.Order) error {
	log.Info().
		Str("order_id", order.ID).
		Str("symbol", order.Symbol).
		Msg("Starting order execution")

	// Step 1: Create order in DB with PENDING status
	if err := e.repository.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to create order in database: %w", err)
	}

	log.Debug().Str("order_id", order.ID).Msg("Order created in database")

	// Step 2: Submit order to KIS broker
	kisReq := &OrderRequest{
		Symbol:   order.Symbol,
		Side:     string(order.Side),
		Type:     string(order.OrderType),
		Price:    0,
		Quantity: order.Quantity,
	}
	if order.Price != nil {
		kisReq.Price = *order.Price
	}

	kisResp, err := e.kisClient.SubmitOrder(ctx, kisReq)
	if err != nil {
		// Step 3a: Update order status to REJECTED
		reason := fmt.Sprintf("Failed to submit to broker: %v", err)
		if updateErr := e.repository.UpdateOrderStatus(ctx, order.ID, repository.OrderStatusRejected, reason); updateErr != nil {
			log.Error().Err(updateErr).Msg("Failed to update order status to REJECTED")
		}

		// Publish rejection event
		e.publishOrderEvent(ctx, order, repository.OrderStatusRejected, reason)

		return fmt.Errorf("order execution failed: %w", err)
	}

	// Step 3b: Update order with broker order ID and SENT status
	if err := e.repository.UpdateOrderExecution(
		ctx,
		order.ID,
		kisResp.OrderID,
		kisResp.FilledPrice,
		kisResp.FilledQuantity,
	); err != nil {
		log.Error().Err(err).Msg("Failed to update order execution details")
	}

	if err := e.repository.UpdateOrderStatus(ctx, order.ID, repository.OrderStatusSent, "Order sent to broker"); err != nil {
		log.Error().Err(err).Msg("Failed to update order status to SENT")
	}

	log.Info().
		Str("order_id", order.ID).
		Str("broker_order_id", kisResp.OrderID).
		Msg("Order sent to broker")

	// Step 4: Publish order event to Kafka
	e.publishOrderEvent(ctx, order, repository.OrderStatusSent, "Order submitted successfully")

	// Step 5: In production, you would start polling for order status updates
	// For now, we assume the order will be filled asynchronously

	return nil
}

// Cancel cancels an order
func (e *OrderExecutor) Cancel(ctx context.Context, orderID string, reason string) error {
	// Get order from database
	order, err := e.repository.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Check if order can be cancelled
	if order.Status == repository.OrderStatusFilled {
		return fmt.Errorf("cannot cancel filled order")
	}
	if order.Status == repository.OrderStatusCancelled {
		return fmt.Errorf("order already cancelled")
	}
	if order.Status == repository.OrderStatusRejected {
		return fmt.Errorf("cannot cancel rejected order")
	}

	// Cancel with broker if order was sent
	if order.BrokerOrderID != nil {
		_, err := e.kisClient.CancelOrder(ctx, *order.BrokerOrderID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("order_id", orderID).
				Msg("Failed to cancel order with broker, updating status anyway")
		}
	}

	// Update order status in database
	if err := e.repository.CancelOrder(ctx, orderID, reason); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publish cancellation event
	e.publishOrderEvent(ctx, order, repository.OrderStatusCancelled, reason)

	log.Info().
		Str("order_id", orderID).
		Str("reason", reason).
		Msg("Order cancelled")

	return nil
}

// publishOrderEvent publishes an order event to Kafka
func (e *OrderExecutor) publishOrderEvent(ctx context.Context, order *repository.Order, status repository.OrderStatus, message string) {
	event := OrderEvent{
		OrderID:    order.ID,
		UserID:     order.UserID,
		Symbol:     order.Symbol,
		Side:       string(order.Side),
		Type:       string(order.OrderType),
		Quantity:   order.Quantity,
		Status:     string(status),
		Message:    message,
		Timestamp:  order.CreatedAt.Unix(),
	}

	if order.Price != nil {
		event.Price = *order.Price
	}
	if order.BrokerOrderID != nil {
		event.BrokerID = *order.BrokerOrderID
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal order event")
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: e.kafkaTopic,
		Key:   sarama.StringEncoder(order.ID),
		Value: sarama.ByteEncoder(eventJSON),
	}

	partition, offset, err := e.kafka.SendMessage(msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", order.ID).
			Msg("Failed to publish order event to Kafka")
		return
	}

	log.Debug().
		Str("order_id", order.ID).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("Order event published to Kafka")
}

// OrderEvent represents an order event published to Kafka
type OrderEvent struct {
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Type      string  `json:"type"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
	BrokerID  string  `json:"broker_id,omitempty"`
	Message   string  `json:"message,omitempty"`
}
