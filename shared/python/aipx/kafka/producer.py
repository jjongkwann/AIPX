"""
Kafka Producer wrapper using confluent-kafka
"""

import json
from typing import Optional, Any, Callable
from confluent_kafka import Producer, KafkaError
import structlog

logger = structlog.get_logger(__name__)


class KafkaProducer:
    """Kafka Producer wrapper"""

    def __init__(
        self,
        brokers: str,
        topic: str,
        client_id: Optional[str] = None,
        compression_type: str = "snappy",
        max_retries: int = 3,
    ):
        """
        Initialize Kafka Producer

        Args:
            brokers: Comma-separated list of Kafka brokers
            topic: Default topic name
            client_id: Client ID for this producer
            compression_type: Compression type (none, gzip, snappy, lz4, zstd)
            max_retries: Maximum number of retries
        """
        self.topic = topic
        self.config = {
            "bootstrap.servers": brokers,
            "compression.type": compression_type,
            "retries": max_retries,
            "acks": "1",
        }

        if client_id:
            self.config["client.id"] = client_id

        self.producer = Producer(self.config)

        logger.info(
            "kafka_producer_initialized",
            brokers=brokers,
            topic=topic,
            compression=compression_type,
        )

    def send(
        self,
        value: Any,
        key: Optional[bytes] = None,
        topic: Optional[str] = None,
        callback: Optional[Callable] = None,
    ):
        """
        Send message to Kafka

        Args:
            value: Message value (will be JSON serialized if dict)
            key: Message key (optional)
            topic: Topic name (uses default if not specified)
            callback: Delivery callback function
        """
        target_topic = topic or self.topic

        # Serialize value if dict
        if isinstance(value, dict):
            value = json.dumps(value).encode("utf-8")
        elif isinstance(value, str):
            value = value.encode("utf-8")

        try:
            self.producer.produce(
                topic=target_topic,
                value=value,
                key=key,
                callback=callback or self._delivery_callback,
            )
            self.producer.poll(0)

        except KafkaError as e:
            logger.error(
                "kafka_produce_error",
                topic=target_topic,
                error=str(e),
            )
            raise

    def _delivery_callback(self, err, msg):
        """Default delivery callback"""
        if err:
            logger.error(
                "kafka_delivery_failed",
                topic=msg.topic(),
                partition=msg.partition(),
                error=str(err),
            )
        else:
            logger.debug(
                "kafka_message_delivered",
                topic=msg.topic(),
                partition=msg.partition(),
                offset=msg.offset(),
            )

    def flush(self, timeout: float = 10.0):
        """
        Flush pending messages

        Args:
            timeout: Maximum time to wait (seconds)
        """
        remaining = self.producer.flush(timeout)
        if remaining > 0:
            logger.warning("kafka_flush_incomplete", remaining_messages=remaining)

    def close(self):
        """Close producer"""
        self.flush()
        logger.info("kafka_producer_closed")


def create_producer(
    brokers: str,
    topic: str,
    **kwargs,
) -> KafkaProducer:
    """
    Factory function to create Kafka producer

    Args:
        brokers: Kafka broker addresses
        topic: Default topic name
        **kwargs: Additional producer configuration

    Returns:
        KafkaProducer instance
    """
    return KafkaProducer(brokers, topic, **kwargs)
