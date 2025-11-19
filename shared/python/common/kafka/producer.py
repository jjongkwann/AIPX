"""Kafka producer implementation for AIPX."""

import json
import logging
from typing import Any, Dict, Optional

from kafka import KafkaProducer as ApacheKafkaProducer
from kafka.errors import KafkaError as ApacheKafkaError

from .config import KafkaConfig
from .exceptions import KafkaProducerError, KafkaConnectionError, KafkaSerializationError

logger = logging.getLogger(__name__)


class KafkaProducer:
    """
    Kafka producer wrapper for AIPX.

    Provides a high-level interface for producing messages to Kafka topics
    with JSON serialization, error handling, and logging.

    Example:
        >>> config = KafkaConfig.from_env()
        >>> producer = KafkaProducer(config)
        >>> await producer.send("user-events", {"user_id": 123, "action": "login"})
        >>> await producer.close()

    Or using context manager:
        >>> async with KafkaProducer(config) as producer:
        ...     await producer.send("user-events", {"user_id": 123})
    """

    def __init__(self, config: KafkaConfig) -> None:
        """
        Initialize KafkaProducer.

        Args:
            config: Kafka configuration

        Raises:
            KafkaConnectionError: If connection to Kafka fails
        """
        self._config = config
        self._producer: Optional[ApacheKafkaProducer] = None
        self._connect()

    def _connect(self) -> None:
        """
        Establish connection to Kafka.

        Raises:
            KafkaConnectionError: If connection fails
        """
        try:
            producer_config = self._config.get_producer_config()
            # Override serializers for JSON
            producer_config["value_serializer"] = lambda v: json.dumps(v).encode("utf-8")
            producer_config["key_serializer"] = (
                lambda k: k.encode("utf-8") if isinstance(k, str) else json.dumps(k).encode("utf-8")
            )

            self._producer = ApacheKafkaProducer(**producer_config)
            logger.info(
                "Kafka producer connected",
                extra={"bootstrap_servers": self._config.bootstrap_servers},
            )
        except ApacheKafkaError as e:
            logger.error(
                "Failed to connect Kafka producer",
                extra={"error": str(e)},
                exc_info=True,
            )
            raise KafkaConnectionError(
                f"Failed to connect to Kafka: {str(e)}", original_error=e
            ) from e

    async def send(
        self,
        topic: str,
        value: Dict[str, Any],
        key: Optional[str] = None,
        partition: Optional[int] = None,
        headers: Optional[Dict[str, str]] = None,
    ) -> None:
        """
        Send a message to Kafka topic.

        Args:
            topic: Topic name
            value: Message value (will be JSON serialized)
            key: Optional message key for partitioning
            partition: Optional partition number
            headers: Optional message headers

        Raises:
            KafkaProducerError: If send operation fails
            KafkaSerializationError: If message serialization fails
        """
        if not self._producer:
            raise KafkaProducerError("Producer not connected")

        try:
            # Convert headers to bytes if provided
            kafka_headers = None
            if headers:
                kafka_headers = [(k, v.encode("utf-8")) for k, v in headers.items()]

            future = self._producer.send(
                topic,
                value=value,
                key=key,
                partition=partition,
                headers=kafka_headers,
            )

            # Wait for the send to complete
            record_metadata = future.get(timeout=10)

            logger.debug(
                "Message sent successfully",
                extra={
                    "topic": topic,
                    "partition": record_metadata.partition,
                    "offset": record_metadata.offset,
                },
            )

        except (TypeError, ValueError) as e:
            logger.error(
                "Failed to serialize message",
                extra={"topic": topic, "error": str(e)},
                exc_info=True,
            )
            raise KafkaSerializationError(
                f"Failed to serialize message: {str(e)}", original_error=e
            ) from e

        except ApacheKafkaError as e:
            logger.error(
                "Failed to send message",
                extra={"topic": topic, "error": str(e)},
                exc_info=True,
            )
            raise KafkaProducerError(
                f"Failed to send message to topic {topic}: {str(e)}", original_error=e
            ) from e

    async def send_batch(
        self,
        topic: str,
        messages: list[Dict[str, Any]],
        keys: Optional[list[Optional[str]]] = None,
    ) -> None:
        """
        Send multiple messages to Kafka topic.

        Args:
            topic: Topic name
            messages: List of message values
            keys: Optional list of message keys (must match messages length)

        Raises:
            KafkaProducerError: If batch send fails
            ValueError: If keys length doesn't match messages length
        """
        if keys and len(keys) != len(messages):
            raise ValueError("Keys list must have same length as messages list")

        if not keys:
            keys = [None] * len(messages)

        for msg, key in zip(messages, keys):
            await self.send(topic, msg, key)

    def flush(self, timeout: Optional[float] = None) -> None:
        """
        Flush pending messages.

        Args:
            timeout: Maximum time to wait in seconds

        Raises:
            KafkaProducerError: If flush fails
        """
        if not self._producer:
            raise KafkaProducerError("Producer not connected")

        try:
            self._producer.flush(timeout=timeout)
            logger.debug("Producer flushed successfully")
        except ApacheKafkaError as e:
            logger.error("Failed to flush producer", extra={"error": str(e)}, exc_info=True)
            raise KafkaProducerError(f"Failed to flush producer: {str(e)}", original_error=e) from e

    async def close(self) -> None:
        """Close the producer connection."""
        if self._producer:
            try:
                self._producer.close()
                logger.info("Kafka producer closed")
            except ApacheKafkaError as e:
                logger.warning(f"Error closing producer: {e}", exc_info=True)
            finally:
                self._producer = None

    async def __aenter__(self) -> "KafkaProducer":
        """Context manager entry."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        await self.close()
