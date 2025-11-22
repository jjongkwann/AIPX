package consumer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"data-recorder-service/internal/buffer"
	"data-recorder-service/internal/pkg/kafka"
	"data-recorder-service/internal/pkg/pb"
	"github.com/rs/zerolog/log"
)

// MarketConsumer consumes market data from Kafka and routes to buffers
type MarketConsumer struct {
	consumer       *kafka.Consumer
	tickBuffer     *buffer.BatchBuffer
	orderbookBuffer *buffer.BatchBuffer
	config         *ConsumerConfig

	// Metrics
	totalMessages   atomic.Int64
	totalErrors     atomic.Int64
	tickCount       atomic.Int64
	orderbookCount  atomic.Int64
	lastMessageAt   atomic.Int64

	// Control
	mu      sync.RWMutex
	running atomic.Bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Brokers        []string
	GroupID        string
	Topics         []string
	OffsetOldest   bool
	SessionTimeout time.Duration
	HeartbeatInterval time.Duration
}

// NewMarketConsumer creates a new market data consumer
func NewMarketConsumer(
	cfg *ConsumerConfig,
	tickBuffer *buffer.BatchBuffer,
	orderbookBuffer *buffer.BatchBuffer,
) (*MarketConsumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if tickBuffer == nil || orderbookBuffer == nil {
		return nil, fmt.Errorf("buffers cannot be nil")
	}

	mc := &MarketConsumer{
		tickBuffer:      tickBuffer,
		orderbookBuffer: orderbookBuffer,
		config:          cfg,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}

	// Create Kafka consumer config
	kafkaConfig := &kafka.ConsumerConfig{
		Brokers:            cfg.Brokers,
		GroupID:            cfg.GroupID,
		Topics:             cfg.Topics,
		EnableAutoCommit:   true,
		AutoCommitInterval: 5,
		SessionTimeout:     int(cfg.SessionTimeout.Seconds()),
		HeartbeatInterval:  int(cfg.HeartbeatInterval.Seconds()),
	}

	if cfg.OffsetOldest {
		kafkaConfig.OffsetInitial = "oldest"
	} else {
		kafkaConfig.OffsetInitial = "latest"
	}

	// Create consumer with message handler
	consumer, err := kafka.NewConsumer(kafkaConfig, mc.handleMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	mc.consumer = consumer
	mc.running.Store(true)

	return mc, nil
}

// Start starts consuming messages
func (mc *MarketConsumer) Start(ctx context.Context) error {
	if !mc.running.Load() {
		return fmt.Errorf("consumer is not running")
	}

	log.Info().
		Strs("topics", mc.config.Topics).
		Str("group_id", mc.config.GroupID).
		Msg("Starting market data consumer")

	// Start consumer in goroutine
	go func() {
		defer close(mc.doneCh)

		if err := mc.consumer.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Consumer stopped with error")
		}
	}()

	return nil
}

// handleMessage processes a consumed message
func (mc *MarketConsumer) handleMessage(ctx context.Context, msg *kafka.ConsumedMessage) error {
	mc.totalMessages.Add(1)
	mc.lastMessageAt.Store(time.Now().Unix())

	// Route based on topic
	switch msg.Topic {
	case "market.tick":
		return mc.handleTickMessage(msg)
	case "market.orderbook":
		return mc.handleOrderBookMessage(msg)
	default:
		log.Warn().
			Str("topic", msg.Topic).
			Msg("Unknown topic")
		return nil
	}
}

// handleTickMessage processes a tick data message
func (mc *MarketConsumer) handleTickMessage(msg *kafka.ConsumedMessage) error {
	// Deserialize protobuf
	tick := &pb.TickData{}
	if err := msg.UnmarshalProto(tick); err != nil {
		mc.totalErrors.Add(1)
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int64("offset", msg.Offset).
			Msg("Failed to unmarshal tick data")
		return nil // Don't retry, skip message
	}

	// Validate
	if err := mc.validateTickData(tick); err != nil {
		mc.totalErrors.Add(1)
		log.Warn().
			Err(err).
			Str("symbol", tick.Symbol).
			Msg("Invalid tick data")
		return nil // Skip invalid message
	}

	// Add to buffer
	if err := mc.tickBuffer.AddTick(tick); err != nil {
		mc.totalErrors.Add(1)
		log.Error().
			Err(err).
			Str("symbol", tick.Symbol).
			Msg("Failed to add tick to buffer")
		// Return error to retry
		return err
	}

	mc.tickCount.Add(1)

	log.Debug().
		Str("symbol", tick.Symbol).
		Float64("price", tick.Price).
		Int64("volume", tick.Volume).
		Msg("Processed tick data")

	return nil
}

// handleOrderBookMessage processes an orderbook message
func (mc *MarketConsumer) handleOrderBookMessage(msg *kafka.ConsumedMessage) error {
	// Deserialize protobuf
	orderbook := &pb.OrderBook{}
	if err := msg.UnmarshalProto(orderbook); err != nil {
		mc.totalErrors.Add(1)
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int64("offset", msg.Offset).
			Msg("Failed to unmarshal orderbook")
		return nil // Skip message
	}

	// Validate
	if err := mc.validateOrderBook(orderbook); err != nil {
		mc.totalErrors.Add(1)
		log.Warn().
			Err(err).
			Str("symbol", orderbook.Symbol).
			Msg("Invalid orderbook")
		return nil // Skip invalid message
	}

	// Add to buffer
	if err := mc.orderbookBuffer.AddOrderBook(orderbook); err != nil {
		mc.totalErrors.Add(1)
		log.Error().
			Err(err).
			Str("symbol", orderbook.Symbol).
			Msg("Failed to add orderbook to buffer")
		return err
	}

	mc.orderbookCount.Add(1)

	log.Debug().
		Str("symbol", orderbook.Symbol).
		Int("bids", len(orderbook.Bids)).
		Int("asks", len(orderbook.Asks)).
		Msg("Processed orderbook")

	return nil
}

// validateTickData validates tick data
func (mc *MarketConsumer) validateTickData(tick *pb.TickData) error {
	if tick.Symbol == "" {
		return fmt.Errorf("symbol is empty")
	}
	if tick.Price <= 0 {
		return fmt.Errorf("invalid price: %f", tick.Price)
	}
	if tick.Volume < 0 {
		return fmt.Errorf("invalid volume: %d", tick.Volume)
	}
	if tick.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", tick.Timestamp)
	}
	return nil
}

// validateOrderBook validates orderbook data
func (mc *MarketConsumer) validateOrderBook(ob *pb.OrderBook) error {
	if ob.Symbol == "" {
		return fmt.Errorf("symbol is empty")
	}
	if len(ob.Bids) == 0 && len(ob.Asks) == 0 {
		return fmt.Errorf("both bids and asks are empty")
	}
	if ob.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", ob.Timestamp)
	}

	// Validate bid levels
	for i, bid := range ob.Bids {
		if bid.Price <= 0 {
			return fmt.Errorf("invalid bid price at index %d: %f", i, bid.Price)
		}
		if bid.Quantity < 0 {
			return fmt.Errorf("invalid bid quantity at index %d: %d", i, bid.Quantity)
		}
	}

	// Validate ask levels
	for i, ask := range ob.Asks {
		if ask.Price <= 0 {
			return fmt.Errorf("invalid ask price at index %d: %f", i, ask.Price)
		}
		if ask.Quantity < 0 {
			return fmt.Errorf("invalid ask quantity at index %d: %d", i, ask.Quantity)
		}
	}

	return nil
}

// Stop stops the consumer gracefully
func (mc *MarketConsumer) Stop() error {
	if !mc.running.CompareAndSwap(true, false) {
		return nil
	}

	log.Info().Msg("Stopping market data consumer")

	// Close consumer
	if err := mc.consumer.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing consumer")
	}

	// Wait for consumer goroutine to finish
	<-mc.doneCh

	log.Info().Msg("Market data consumer stopped")
	return nil
}

// Metrics returns consumer metrics
func (mc *MarketConsumer) Metrics() ConsumerMetrics {
	return ConsumerMetrics{
		TotalMessages:  mc.totalMessages.Load(),
		TotalErrors:    mc.totalErrors.Load(),
		TickCount:      mc.tickCount.Load(),
		OrderBookCount: mc.orderbookCount.Load(),
		LastMessageAt:  time.Unix(mc.lastMessageAt.Load(), 0),
	}
}

// HealthCheck checks consumer health
func (mc *MarketConsumer) HealthCheck() error {
	if !mc.running.Load() {
		return fmt.Errorf("consumer is not running")
	}

	// Check if messages are being processed
	lastMsg := time.Unix(mc.lastMessageAt.Load(), 0)
	if !lastMsg.IsZero() && time.Since(lastMsg) > 5*time.Minute {
		return fmt.Errorf("no messages processed in last 5 minutes")
	}

	return nil
}

// ConsumerMetrics contains consumer statistics
type ConsumerMetrics struct {
	TotalMessages  int64
	TotalErrors    int64
	TickCount      int64
	OrderBookCount int64
	LastMessageAt  time.Time
}
