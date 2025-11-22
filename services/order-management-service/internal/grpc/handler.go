package grpc

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "shared/pkg/pb"
	"order-management-service/internal/broker"
	"order-management-service/internal/ratelimit"
	"order-management-service/internal/repository"
	"order-management-service/internal/risk"
)

// OrderServiceHandler implements the OrderService gRPC service
type OrderServiceHandler struct {
	pb.UnimplementedOrderServiceServer
	riskEngine    *risk.Engine
	rateLimiter   *ratelimit.Limiter
	orderExecutor *broker.OrderExecutor
	repository    repository.OrderRepository
	streams       sync.Map // map[string]pb.OrderService_StreamOrdersServer
}

// HandlerConfig holds the handler configuration
type HandlerConfig struct {
	RiskEngine    *risk.Engine
	RateLimiter   *ratelimit.Limiter
	OrderExecutor *broker.OrderExecutor
	Repository    repository.OrderRepository
}

// NewOrderServiceHandler creates a new order service handler
func NewOrderServiceHandler(config *HandlerConfig) (*OrderServiceHandler, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.RiskEngine == nil {
		return nil, fmt.Errorf("risk engine cannot be nil")
	}
	if config.RateLimiter == nil {
		return nil, fmt.Errorf("rate limiter cannot be nil")
	}
	if config.OrderExecutor == nil {
		return nil, fmt.Errorf("order executor cannot be nil")
	}
	if config.Repository == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}

	handler := &OrderServiceHandler{
		riskEngine:    config.RiskEngine,
		rateLimiter:   config.RateLimiter,
		orderExecutor: config.OrderExecutor,
		repository:    config.Repository,
	}

	log.Info().Msg("Order service handler created")
	return handler, nil
}

// StreamOrders implements bidirectional streaming for high-frequency orders
func (h *OrderServiceHandler) StreamOrders(stream pb.OrderService_StreamOrdersServer) error {
	ctx := stream.Context()
	requestID := GetRequestID(ctx)
	userID := GetUserID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("user_id", userID).
		Msg("Client connected to order stream")

	// Store stream for this user
	h.streams.Store(userID, stream)
	defer h.streams.Delete(userID)

	// Handle incoming orders
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Str("request_id", requestID).
				Str("user_id", userID).
				Msg("Client disconnected from order stream")
			return nil

		default:
			// Receive order request
			req, err := stream.Recv()
			if err == io.EOF {
				log.Info().
					Str("request_id", requestID).
					Str("user_id", userID).
					Msg("Stream closed by client")
				return nil
			}
			if err != nil {
				log.Error().
					Err(err).
					Str("request_id", requestID).
					Msg("Error receiving order")
				return status.Error(codes.Internal, "failed to receive order")
			}

			// Process order asynchronously
			go h.processOrder(stream, req, userID)
		}
	}
}

// processOrder processes a single order request
func (h *OrderServiceHandler) processOrder(stream pb.OrderService_StreamOrdersServer, req *pb.OrderRequest, userID string) {
	ctx := stream.Context()
	requestID := GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("order_id", req.Id).
		Str("symbol", req.Symbol).
		Msg("Processing order")

	// Step 1: Validate request
	if err := h.validateRequest(req); err != nil {
		h.sendResponse(stream, req.Id, pb.OrderStatus_REJECTED, err.Error())
		return
	}

	// Step 2: Check rate limit
	allowed, err := h.rateLimiter.Allow(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("Rate limit check failed")
		h.sendResponse(stream, req.Id, pb.OrderStatus_REJECTED, "rate limit check failed")
		return
	}
	if !allowed {
		log.Warn().Str("user_id", userID).Msg("Rate limit exceeded")
		h.sendResponse(stream, req.Id, pb.OrderStatus_REJECTED, "rate limit exceeded")
		return
	}

	// Step 3: Convert protobuf request to domain order
	order := h.protoToOrder(req, userID)

	// Step 4: Risk validation
	if err := h.riskEngine.Validate(ctx, order); err != nil {
		log.Warn().
			Err(err).
			Str("order_id", req.Id).
			Msg("Risk validation failed")
		h.sendResponse(stream, req.Id, pb.OrderStatus_REJECTED, fmt.Sprintf("risk check failed: %v", err))
		return
	}

	// Step 5: Execute order
	if err := h.orderExecutor.Execute(ctx, order); err != nil {
		log.Error().
			Err(err).
			Str("order_id", req.Id).
			Msg("Order execution failed")
		h.sendResponse(stream, req.Id, pb.OrderStatus_REJECTED, fmt.Sprintf("execution failed: %v", err))
		return
	}

	// Step 6: Send success response
	h.sendResponse(stream, req.Id, pb.OrderStatus_ACCEPTED, "order submitted successfully")
}

// validateRequest validates an order request
func (h *OrderServiceHandler) validateRequest(req *pb.OrderRequest) error {
	if req.Id == "" {
		return fmt.Errorf("order ID is required")
	}
	if req.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if req.Type == pb.OrderType_LIMIT && req.Price <= 0 {
		return fmt.Errorf("price is required for limit orders")
	}
	return nil
}

// protoToOrder converts protobuf order request to domain order
func (h *OrderServiceHandler) protoToOrder(req *pb.OrderRequest, userID string) *repository.Order {
	order := &repository.Order{
		ID:         req.Id,
		UserID:     userID,
		Symbol:     req.Symbol,
		Side:       h.convertSide(req.Side),
		OrderType:  h.convertOrderType(req.Type),
		Quantity:   int(req.Quantity),
		Status:     repository.OrderStatusPending,
	}

	if req.StrategyId != "" {
		order.StrategyID = &req.StrategyId
	}

	if req.Type == pb.OrderType_LIMIT {
		order.Price = &req.Price
	}

	return order
}

// orderToProto converts domain order to protobuf order response
func (h *OrderServiceHandler) orderToProto(order *repository.Order) *pb.OrderResponse {
	resp := &pb.OrderResponse{
		OrderId:   order.ID,
		RequestId: order.ID,
		Status:    h.convertStatus(order.Status),
		Timestamp: time.Now().Unix(),
	}

	if order.FilledPrice != nil {
		resp.FilledPrice = *order.FilledPrice
	}
	if order.FilledQuantity != nil {
		resp.FilledQuantity = int64(*order.FilledQuantity)
	}
	if order.RejectReason != nil {
		resp.Message = *order.RejectReason
	}

	return resp
}

// sendResponse sends a response to the client
func (h *OrderServiceHandler) sendResponse(stream pb.OrderService_StreamOrdersServer, orderID string, status pb.OrderStatus, message string) {
	resp := &pb.OrderResponse{
		OrderId:   orderID,
		RequestId: orderID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	if err := stream.Send(resp); err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to send response")
	}
}

// convertSide converts protobuf side to domain side
func (h *OrderServiceHandler) convertSide(side pb.Side) repository.OrderSide {
	switch side {
	case pb.Side_BUY:
		return repository.OrderSideBuy
	case pb.Side_SELL:
		return repository.OrderSideSell
	default:
		return repository.OrderSideBuy
	}
}

// convertOrderType converts protobuf order type to domain order type
func (h *OrderServiceHandler) convertOrderType(orderType pb.OrderType) repository.OrderType {
	switch orderType {
	case pb.OrderType_LIMIT:
		return repository.OrderTypeLimit
	case pb.OrderType_MARKET:
		return repository.OrderTypeMarket
	default:
		return repository.OrderTypeLimit
	}
}

// convertStatus converts domain status to protobuf status
func (h *OrderServiceHandler) convertStatus(status repository.OrderStatus) pb.OrderStatus {
	switch status {
	case repository.OrderStatusPending:
		return pb.OrderStatus_PENDING
	case repository.OrderStatusSent:
		return pb.OrderStatus_ACCEPTED
	case repository.OrderStatusFilled:
		return pb.OrderStatus_FILLED
	case repository.OrderStatusRejected:
		return pb.OrderStatus_REJECTED
	case repository.OrderStatusCancelled:
		return pb.OrderStatus_CANCELLED
	default:
		return pb.OrderStatus_PENDING
	}
}
