package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// MessageHandler processes consumed messages
// Return error to skip committing the offset (message will be reprocessed)
type MessageHandler func(ctx context.Context, message *ConsumedMessage) error

// ConsumedMessage wraps a Kafka message with helper methods
type ConsumedMessage struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp int64
}

// UnmarshalProto unmarshals the message value into a protobuf message
func (m *ConsumedMessage) UnmarshalProto(msg proto.Message) error {
	if err := proto.Unmarshal(m.Value, msg); err != nil {
		return fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
	}
	return nil
}

// Consumer wraps sarama consumer group with graceful shutdown
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *ConsumerConfig
	handler       MessageHandler
	mu            sync.RWMutex
	closed        bool

	// Metrics hooks (optional)
	onMessageProcessed func(topic string, partition int32, offset int64, err error)
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config *ConsumerConfig, handler MessageHandler) (*Consumer, error) {
	saramaConfig, err := config.ToSaramaConsumerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create sarama config: %w", err)
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &Consumer{
		consumerGroup: consumerGroup,
		config:        config,
		handler:       handler,
		closed:        false,
	}, nil
}

// Start starts consuming messages
// This method blocks until context is cancelled
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConsumerClosed
	}
	c.mu.RUnlock()

	consumerHandler := &consumerGroupHandler{
		handler:  c.handler,
		consumer: c,
	}

	// Handle consumer group errors
	go func() {
		for err := range c.consumerGroup.Errors() {
			if err != nil {
				// Log error or call error handler
				if c.onMessageProcessed != nil {
					c.onMessageProcessed("", 0, 0, err)
				}
			}
		}
	}()

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := c.consumerGroup.Consume(ctx, c.config.Topics, consumerHandler); err != nil {
				c.mu.RLock()
				closed := c.closed
				c.mu.RUnlock()

				if closed {
					return nil
				}
				return fmt.Errorf("consume error: %w", err)
			}
		}
	}
}

// Close closes the consumer gracefully
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.consumerGroup.Close()
}

// SetMessageProcessedHandler sets a callback for message processing events
func (c *Consumer) SetMessageProcessedHandler(handler func(topic string, partition int32, offset int64, err error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessageProcessed = handler
}

// Pause pauses consumption from specified topic partitions
func (c *Consumer) Pause(topicPartitions map[string][]int32) {
	c.consumerGroup.Pause(topicPartitions)
}

// Resume resumes consumption from specified topic partitions
func (c *Consumer) Resume(topicPartitions map[string][]int32) {
	c.consumerGroup.Resume(topicPartitions)
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler  MessageHandler
	consumer *Consumer
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	// Called when a new session is created
	return nil
}

func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	// Called when session is ending (rebalance, shutdown)
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			// Convert to ConsumedMessage
			consumedMsg := &ConsumedMessage{
				Topic:     message.Topic,
				Partition: message.Partition,
				Offset:    message.Offset,
				Key:       message.Key,
				Value:     message.Value,
				Timestamp: message.Timestamp.Unix(),
				Headers:   make(map[string]string),
			}

			// Extract headers
			for _, header := range message.Headers {
				consumedMsg.Headers[string(header.Key)] = string(header.Value)
			}

			// Process message
			err := h.handler(session.Context(), consumedMsg)

			// Call metrics handler
			if h.consumer.onMessageProcessed != nil {
				h.consumer.onMessageProcessed(message.Topic, message.Partition, message.Offset, err)
			}

			// Mark message as processed if no error
			if err == nil {
				session.MarkMessage(message, "")
			} else {
				// Error occurred, message will be reprocessed
				// Optionally implement dead letter queue logic here
			}

			// Commit offsets periodically (if auto-commit is disabled)
			if !h.consumer.config.EnableAutoCommit {
				session.Commit()
			}
		}
	}
}
