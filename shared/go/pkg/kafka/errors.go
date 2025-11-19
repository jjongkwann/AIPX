package kafka

import (
	"errors"
	"fmt"
)

var (
	// ErrProducerClosed is returned when attempting to use a closed producer
	ErrProducerClosed = errors.New("producer is closed")

	// ErrConsumerClosed is returned when attempting to use a closed consumer
	ErrConsumerClosed = errors.New("consumer is closed")

	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrMessageTooLarge is returned when message exceeds size limit
	ErrMessageTooLarge = errors.New("message too large")

	// ErrSerializationFailed is returned when message serialization fails
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrDeserializationFailed is returned when message deserialization fails
	ErrDeserializationFailed = errors.New("deserialization failed")
)

// ProducerError wraps producer errors with additional context
type ProducerError struct {
	Topic   string
	Key     string
	Err     error
	Retries int
}

func (e *ProducerError) Error() string {
	return fmt.Sprintf("producer error (topic=%s, key=%s, retries=%d): %v",
		e.Topic, e.Key, e.Retries, e.Err)
}

func (e *ProducerError) Unwrap() error {
	return e.Err
}

// ConsumerError wraps consumer errors with additional context
type ConsumerError struct {
	Topic     string
	Partition int32
	Offset    int64
	Err       error
}

func (e *ConsumerError) Error() string {
	return fmt.Sprintf("consumer error (topic=%s, partition=%d, offset=%d): %v",
		e.Topic, e.Partition, e.Offset, e.Err)
}

func (e *ConsumerError) Unwrap() error {
	return e.Err
}
