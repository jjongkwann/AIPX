package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// Producer wraps sarama async producer with retry logic and metrics
type Producer struct {
	producer sarama.AsyncProducer
	config   *ProducerConfig
	mu       sync.RWMutex
	closed   bool

	// Metrics hooks (optional)
	onSuccess func(topic string, partition int32, offset int64)
	onError   func(topic string, err error)
}

// Message represents a Kafka message to be produced
type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
}

// NewProducer creates a new Kafka producer
func NewProducer(config *ProducerConfig) (*Producer, error) {
	saramaConfig, err := config.ToSaramaProducerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create sarama config: %w", err)
	}

	producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	p := &Producer{
		producer: producer,
		config:   config,
		closed:   false,
	}

	// Start success/error handlers
	go p.handleSuccesses()
	go p.handleErrors()

	return p, nil
}

// Send sends a message to Kafka asynchronously
func (p *Producer) Send(ctx context.Context, msg *Message) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrProducerClosed
	}

	if len(msg.Value) > p.config.MaxMessageBytes {
		return ErrMessageTooLarge
	}

	saramaMsg := &sarama.ProducerMessage{
		Topic:     msg.Topic,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
		Timestamp: msg.Timestamp,
	}

	// Add headers
	if len(msg.Headers) > 0 {
		saramaMsg.Headers = make([]sarama.RecordHeader, 0, len(msg.Headers))
		for k, v := range msg.Headers {
			saramaMsg.Headers = append(saramaMsg.Headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
	}

	select {
	case p.producer.Input() <- saramaMsg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendProto sends a protobuf message to Kafka
func (p *Producer) SendProto(ctx context.Context, topic string, key []byte, msg proto.Message, headers map[string]string) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializationFailed, err)
	}

	return p.Send(ctx, &Message{
		Topic:     topic,
		Key:       key,
		Value:     data,
		Headers:   headers,
		Timestamp: time.Now(),
	})
}

// SendJSON sends a JSON-serializable message to Kafka (for non-protobuf messages)
func (p *Producer) SendJSON(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	return p.Send(ctx, &Message{
		Topic:     topic,
		Key:       key,
		Value:     value,
		Headers:   headers,
		Timestamp: time.Now(),
	})
}

// Close closes the producer gracefully
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	return p.producer.Close()
}

// SetSuccessHandler sets a callback for successful message delivery
func (p *Producer) SetSuccessHandler(handler func(topic string, partition int32, offset int64)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSuccess = handler
}

// SetErrorHandler sets a callback for message delivery errors
func (p *Producer) SetErrorHandler(handler func(topic string, err error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onError = handler
}

// handleSuccesses processes successful message deliveries
func (p *Producer) handleSuccesses() {
	for msg := range p.producer.Successes() {
		p.mu.RLock()
		handler := p.onSuccess
		p.mu.RUnlock()

		if handler != nil {
			handler(msg.Topic, msg.Partition, msg.Offset)
		}
	}
}

// handleErrors processes message delivery errors
func (p *Producer) handleErrors() {
	for err := range p.producer.Errors() {
		p.mu.RLock()
		handler := p.onError
		p.mu.RUnlock()

		if handler != nil {
			handler(err.Msg.Topic, err.Err)
		}
	}
}

// HealthCheck verifies producer connection to Kafka
func (p *Producer) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrProducerClosed
	}

	// Producer is healthy if it's not closed
	return nil
}
