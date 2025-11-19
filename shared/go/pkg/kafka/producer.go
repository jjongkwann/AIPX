package kafka

import (
	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

// Producer wraps Sarama async producer
type Producer struct {
	producer sarama.AsyncProducer
	config   *ProducerConfig
}

// ProducerConfig holds producer configuration
type ProducerConfig struct {
	Brokers       []string
	Topic         string
	RequiredAcks  sarama.RequiredAcks
	Compression   sarama.CompressionCodec
	MaxRetry      int
	ReturnSuccess bool
	ReturnErrors  bool
}

// NewProducer creates a new Kafka producer
func NewProducer(config *ProducerConfig) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = config.RequiredAcks
	saramaConfig.Producer.Compression = config.Compression
	saramaConfig.Producer.Retry.Max = config.MaxRetry
	saramaConfig.Producer.Return.Successes = config.ReturnSuccess
	saramaConfig.Producer.Return.Errors = config.ReturnErrors

	producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}

	p := &Producer{
		producer: producer,
		config:   config,
	}

	// Handle success/error messages
	go p.handleMessages()

	return p, nil
}

// SendMessage sends a message to Kafka
func (p *Producer) SendMessage(key, value []byte) {
	msg := &sarama.ProducerMessage{
		Topic: p.config.Topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	p.producer.Input() <- msg
}

// handleMessages processes success and error messages
func (p *Producer) handleMessages() {
	for {
		select {
		case success := <-p.producer.Successes():
			if success != nil {
				log.Debug().
					Str("topic", success.Topic).
					Int32("partition", success.Partition).
					Int64("offset", success.Offset).
					Msg("Message sent successfully")
			}
		case err := <-p.producer.Errors():
			if err != nil {
				log.Error().
					Err(err.Err).
					Str("topic", err.Msg.Topic).
					Msg("Failed to send message")
			}
		}
	}
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// DefaultProducerConfig returns default producer configuration
func DefaultProducerConfig(brokers []string, topic string) *ProducerConfig {
	return &ProducerConfig{
		Brokers:       brokers,
		Topic:         topic,
		RequiredAcks:  sarama.WaitForLocal,
		Compression:   sarama.CompressionSnappy,
		MaxRetry:      3,
		ReturnSuccess: true,
		ReturnErrors:  true,
	}
}
