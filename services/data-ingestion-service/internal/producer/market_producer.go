package producer

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jjongkwann/aipx/shared/go/pkg/kafka"
	market_data "github.com/jjongkwann/aipx/shared/go/pkg/pb"
	"github.com/rs/zerolog/log"
)

const (
	// Kafka topic names
	TopicMarketTick      = "market.tick"
	TopicMarketOrderBook = "market.orderbook"
)

// MarketProducer wraps Kafka producer for market data
type MarketProducer struct {
	producer *kafka.Producer

	// Metrics
	tickCount      atomic.Uint64
	orderBookCount atomic.Uint64
	errorCount     atomic.Uint64
}

// NewMarketProducer creates a new market data producer
func NewMarketProducer(config *kafka.ProducerConfig) (*MarketProducer, error) {
	producer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	mp := &MarketProducer{
		producer: producer,
	}

	// Set up success and error handlers for metrics
	producer.SetSuccessHandler(mp.onSuccess)
	producer.SetErrorHandler(mp.onError)

	log.Info().Msg("market producer initialized")

	return mp, nil
}

// PublishTickData publishes tick data to Kafka
func (mp *MarketProducer) PublishTickData(ctx context.Context, data *market_data.TickData) error {
	if data == nil {
		return fmt.Errorf("nil tick data")
	}

	// Use symbol as partition key for ordered processing
	key := []byte(data.Symbol)

	// Add metadata headers
	headers := map[string]string{
		"symbol":    data.Symbol,
		"data_type": "tick",
		"version":   "1.0",
	}

	if err := mp.producer.SendProto(ctx, TopicMarketTick, key, data, headers); err != nil {
		mp.errorCount.Add(1)
		return fmt.Errorf("failed to publish tick data: %w", err)
	}

	log.Debug().
		Str("symbol", data.Symbol).
		Float64("price", data.Price).
		Int64("volume", data.Volume).
		Msg("tick data published")

	return nil
}

// PublishOrderBook publishes order book data to Kafka
func (mp *MarketProducer) PublishOrderBook(ctx context.Context, data *market_data.OrderBook) error {
	if data == nil {
		return fmt.Errorf("nil order book")
	}

	// Use symbol as partition key for ordered processing
	key := []byte(data.Symbol)

	// Add metadata headers
	headers := map[string]string{
		"symbol":    data.Symbol,
		"data_type": "orderbook",
		"version":   "1.0",
	}

	if err := mp.producer.SendProto(ctx, TopicMarketOrderBook, key, data, headers); err != nil {
		mp.errorCount.Add(1)
		return fmt.Errorf("failed to publish order book: %w", err)
	}

	log.Debug().
		Str("symbol", data.Symbol).
		Int("bid_levels", len(data.Bids)).
		Int("ask_levels", len(data.Asks)).
		Msg("order book published")

	return nil
}

// Close closes the producer gracefully
func (mp *MarketProducer) Close() error {
	log.Info().Msg("closing market producer")

	if err := mp.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}

	log.Info().
		Uint64("tick_count", mp.tickCount.Load()).
		Uint64("orderbook_count", mp.orderBookCount.Load()).
		Uint64("error_count", mp.errorCount.Load()).
		Msg("market producer closed")

	return nil
}

// GetMetrics returns current metrics
func (mp *MarketProducer) GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"tick_count":      mp.tickCount.Load(),
		"orderbook_count": mp.orderBookCount.Load(),
		"error_count":     mp.errorCount.Load(),
	}
}

// onSuccess handles successful message delivery
func (mp *MarketProducer) onSuccess(topic string, partition int32, offset int64) {
	switch topic {
	case TopicMarketTick:
		mp.tickCount.Add(1)
	case TopicMarketOrderBook:
		mp.orderBookCount.Add(1)
	}

	log.Debug().
		Str("topic", topic).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("message delivered successfully")
}

// onError handles message delivery errors
func (mp *MarketProducer) onError(topic string, err error) {
	mp.errorCount.Add(1)

	log.Error().
		Err(err).
		Str("topic", topic).
		Msg("message delivery failed")
}

// HealthCheck verifies producer health
func (mp *MarketProducer) HealthCheck(ctx context.Context) error {
	return mp.producer.HealthCheck(ctx)
}
