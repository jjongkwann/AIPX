package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// MessageHandler processes consumed messages
type MessageHandler func(ctx context.Context, message *ConsumedMessage) error

// ConsumedMessage wraps a Kafka message
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
	return proto.Unmarshal(m.Value, msg)
}

// Consumer wraps sarama consumer group
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *ConsumerConfig
	handler       MessageHandler
	mu            sync.RWMutex
	closed        bool
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Brokers            []string
	GroupID            string
	Topics             []string
	EnableAutoCommit   bool
	AutoCommitInterval int
	SessionTimeout     int
	HeartbeatInterval  int
	OffsetInitial      string // "oldest" or "latest"
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config *ConsumerConfig, handler MessageHandler) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.EnableAutoCommit

	if config.OffsetInitial == "oldest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
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
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	consumerHandler := &consumerGroupHandler{
		handler:  c.handler,
		consumer: c,
	}

	go func() {
		for err := range c.consumerGroup.Errors() {
			if err != nil {
				// Log error
			}
		}
	}()

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

// Close closes the consumer
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.consumerGroup.Close()
}

type consumerGroupHandler struct {
	handler  MessageHandler
	consumer *Consumer
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
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

			consumedMsg := &ConsumedMessage{
				Topic:     message.Topic,
				Partition: message.Partition,
				Offset:    message.Offset,
				Key:       message.Key,
				Value:     message.Value,
				Timestamp: message.Timestamp.Unix(),
				Headers:   make(map[string]string),
			}

			for _, header := range message.Headers {
				consumedMsg.Headers[string(header.Key)] = string(header.Value)
			}

			err := h.handler(session.Context(), consumedMsg)

			if err == nil {
				session.MarkMessage(message, "")
			}

			if !h.consumer.config.EnableAutoCommit {
				session.Commit()
			}
		}
	}
}
