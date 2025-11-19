package kafka

import (
	"crypto/tls"
	"time"

	"github.com/IBM/sarama"
)

// ProducerConfig holds configuration for Kafka producer
type ProducerConfig struct {
	// Brokers is the list of Kafka broker addresses
	Brokers []string

	// ClientID identifies the client
	ClientID string

	// Compression codec for messages (gzip, snappy, lz4, zstd)
	Compression sarama.CompressionCodec

	// MaxMessageBytes is the maximum size of a message
	MaxMessageBytes int

	// RequiredAcks specifies the number of acks required from broker
	// 0 = NoResponse, 1 = WaitForLocal, -1 = WaitForAll
	RequiredAcks sarama.RequiredAcks

	// Timeout for broker acknowledgements
	Timeout time.Duration

	// RetryMax is the maximum number of retry attempts
	RetryMax int

	// RetryBackoff is the time to wait before retrying
	RetryBackoff time.Duration

	// EnableIdempotence ensures exactly-once semantics
	EnableIdempotence bool

	// TLS configuration for secure connections
	TLSConfig *tls.Config

	// SASL configuration (if needed)
	SASLConfig *SASLConfig

	// Metadata refresh frequency
	MetadataRefreshFrequency time.Duration
}

// ConsumerConfig holds configuration for Kafka consumer
type ConsumerConfig struct {
	// Brokers is the list of Kafka broker addresses
	Brokers []string

	// GroupID is the consumer group identifier
	GroupID string

	// Topics to subscribe to
	Topics []string

	// ClientID identifies the client
	ClientID string

	// InitialOffset specifies where to start consuming (oldest or newest)
	InitialOffset int64

	// SessionTimeout is the timeout used to detect consumer failures
	SessionTimeout time.Duration

	// RebalanceTimeout is the maximum allowed time for rebalancing
	RebalanceTimeout time.Duration

	// HeartbeatInterval is the expected time between heartbeats
	HeartbeatInterval time.Duration

	// MaxProcessingTime is the maximum time a message can be processed
	MaxProcessingTime time.Duration

	// EnableAutoCommit determines if offsets are committed automatically
	EnableAutoCommit bool

	// AutoCommitInterval is how often to commit when auto-commit is enabled
	AutoCommitInterval time.Duration

	// TLS configuration for secure connections
	TLSConfig *tls.Config

	// SASL configuration (if needed)
	SASLConfig *SASLConfig

	// Metadata refresh frequency
	MetadataRefreshFrequency time.Duration
}

// SASLConfig holds SASL authentication configuration
type SASLConfig struct {
	// Enable SASL authentication
	Enable bool

	// Mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
	Mechanism sarama.SASLMechanism

	// Username for authentication
	Username string

	// Password for authentication
	Password string

	// Handshake determines if SASL handshake should be performed
	Handshake bool
}

// DefaultProducerConfig returns a producer configuration with sensible defaults
func DefaultProducerConfig() *ProducerConfig {
	return &ProducerConfig{
		ClientID:                 "aipx-producer",
		Compression:              sarama.CompressionSnappy,
		MaxMessageBytes:          1000000, // 1MB
		RequiredAcks:             sarama.WaitForAll,
		Timeout:                  10 * time.Second,
		RetryMax:                 3,
		RetryBackoff:             100 * time.Millisecond,
		EnableIdempotence:        true,
		MetadataRefreshFrequency: 10 * time.Minute,
	}
}

// DefaultConsumerConfig returns a consumer configuration with sensible defaults
func DefaultConsumerConfig(groupID string, topics []string) *ConsumerConfig {
	return &ConsumerConfig{
		GroupID:                  groupID,
		Topics:                   topics,
		ClientID:                 "aipx-consumer",
		InitialOffset:            sarama.OffsetNewest,
		SessionTimeout:           20 * time.Second,
		RebalanceTimeout:         60 * time.Second,
		HeartbeatInterval:        3 * time.Second,
		MaxProcessingTime:        30 * time.Second,
		EnableAutoCommit:         false, // Manual commit for better control
		AutoCommitInterval:       1 * time.Second,
		MetadataRefreshFrequency: 10 * time.Minute,
	}
}

// ToSaramaProducerConfig converts ProducerConfig to sarama.Config
func (c *ProducerConfig) ToSaramaProducerConfig() (*sarama.Config, error) {
	if len(c.Brokers) == 0 {
		return nil, ErrInvalidConfig
	}

	config := sarama.NewConfig()
	config.ClientID = c.ClientID
	config.Producer.Compression = c.Compression
	config.Producer.MaxMessageBytes = c.MaxMessageBytes
	config.Producer.RequiredAcks = c.RequiredAcks
	config.Producer.Timeout = c.Timeout
	config.Producer.Retry.Max = c.RetryMax
	config.Producer.Retry.Backoff = c.RetryBackoff
	config.Producer.Idempotent = c.EnableIdempotence
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Metadata.RefreshFrequency = c.MetadataRefreshFrequency

	// Enable idempotence requires specific settings
	if c.EnableIdempotence {
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Retry.Max = 5
		config.Net.MaxOpenRequests = 1
	}

	// TLS configuration
	if c.TLSConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = c.TLSConfig
	}

	// SASL configuration
	if c.SASLConfig != nil && c.SASLConfig.Enable {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = c.SASLConfig.Mechanism
		config.Net.SASL.User = c.SASLConfig.Username
		config.Net.SASL.Password = c.SASLConfig.Password
		config.Net.SASL.Handshake = c.SASLConfig.Handshake
	}

	return config, nil
}

// ToSaramaConsumerConfig converts ConsumerConfig to sarama.Config
func (c *ConsumerConfig) ToSaramaConsumerConfig() (*sarama.Config, error) {
	if len(c.Brokers) == 0 {
		return nil, ErrInvalidConfig
	}
	if c.GroupID == "" {
		return nil, ErrInvalidConfig
	}
	if len(c.Topics) == 0 {
		return nil, ErrInvalidConfig
	}

	config := sarama.NewConfig()
	config.ClientID = c.ClientID
	config.Consumer.Group.Session.Timeout = c.SessionTimeout
	config.Consumer.Group.Rebalance.Timeout = c.RebalanceTimeout
	config.Consumer.Group.Heartbeat.Interval = c.HeartbeatInterval
	config.Consumer.MaxProcessingTime = c.MaxProcessingTime
	config.Consumer.Offsets.Initial = c.InitialOffset
	config.Consumer.Return.Errors = true
	config.Metadata.RefreshFrequency = c.MetadataRefreshFrequency

	// Auto-commit settings
	if c.EnableAutoCommit {
		config.Consumer.Offsets.AutoCommit.Enable = true
		config.Consumer.Offsets.AutoCommit.Interval = c.AutoCommitInterval
	} else {
		config.Consumer.Offsets.AutoCommit.Enable = false
	}

	// TLS configuration
	if c.TLSConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = c.TLSConfig
	}

	// SASL configuration
	if c.SASLConfig != nil && c.SASLConfig.Enable {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = c.SASLConfig.Mechanism
		config.Net.SASL.User = c.SASLConfig.Username
		config.Net.SASL.Password = c.SASLConfig.Password
		config.Net.SASL.Handshake = c.SASLConfig.Handshake
	}

	return config, nil
}
