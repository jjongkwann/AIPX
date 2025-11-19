package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

// MessageHandler processes consumed messages
type MessageHandler func(message *sarama.ConsumerMessage) error

// Consumer wraps Sarama consumer group
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *ConsumerConfig
	handler       MessageHandler
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Brokers       []string
	Topics        []string
	GroupID       string
	OffsetInitial int64
	Handler       MessageHandler
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config *ConsumerConfig) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.Consumer.Offsets.Initial = config.OffsetInitial
	saramaConfig.Version = sarama.V3_3_0_0

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		consumerGroup: consumerGroup,
		config:        config,
		handler:       config.Handler,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	consumerHandler := &consumerGroupHandler{
		handler: c.handler,
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Consumer context cancelled, stopping...")
			return nil
		default:
			if err := c.consumerGroup.Consume(ctx, c.config.Topics, consumerHandler); err != nil {
				log.Error().Err(err).Msg("Error from consumer")
				return err
			}
		}
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.consumerGroup.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler MessageHandler
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("Consumer group session setup")
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("Consumer group session cleanup")
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.handler(message); err != nil {
			log.Error().
				Err(err).
				Str("topic", message.Topic).
				Int32("partition", message.Partition).
				Int64("offset", message.Offset).
				Msg("Error processing message")
			// Continue processing despite error
		}
		session.MarkMessage(message, "")
	}
	return nil
}

// DefaultConsumerConfig returns default consumer configuration
func DefaultConsumerConfig(brokers []string, topics []string, groupID string, handler MessageHandler) *ConsumerConfig {
	return &ConsumerConfig{
		Brokers:       brokers,
		Topics:        topics,
		GroupID:       groupID,
		OffsetInitial: sarama.OffsetNewest,
		Handler:       handler,
	}
}
