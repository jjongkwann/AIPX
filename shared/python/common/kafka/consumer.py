"""Kafka consumer implementation for AIPX."""

import json
import logging
from typing import Any, Callable, Dict, Optional, List
import asyncio

from kafka import KafkaConsumer as ApacheKafkaConsumer
from kafka.errors import KafkaError as ApacheKafkaError

from .config import KafkaConfig
from .exceptions import KafkaConsumerError, KafkaConnectionError, KafkaSerializationError

logger = logging.getLogger(__name__)


class KafkaConsumer:
    """
    Kafka consumer wrapper for AIPX.

    Provides a high-level interface for consuming messages from Kafka topics
    with JSON deserialization, error handling, and automatic offset management.

    Example:
        >>> config = KafkaConfig.from_env()
        >>> consumer = KafkaConsumer(config, "my-service-group")
        >>>
        >>> async def handle_message(message):
        ...     print(f"Received: {message}")
        >>>
        >>> await consumer.consume("user-events", handle_message)

    Or using context manager:
        >>> async with KafkaConsumer(config, "my-group") as consumer:
        ...     await consumer.consume("events", handler)
    """

    def __init__(self, config: KafkaConfig, group_id: str) -> None:
        """
        Initialize KafkaConsumer.

        Args:
            config: Kafka configuration
            group_id: Consumer group ID

        Raises:
            KafkaConnectionError: If connection to Kafka fails
        """
        self._config = config
        self._group_id = group_id
        self._consumer: Optional[ApacheKafkaConsumer] = None
        self._running = False
        self._connect()

    def _connect(self) -> None:
        """
        Establish connection to Kafka.

        Raises:
            KafkaConnectionError: If connection fails
        """
        try:
            consumer_config = self._config.get_consumer_config(self._group_id)
            # Override deserializers for JSON
            consumer_config["value_deserializer"] = lambda v: json.loads(v.decode("utf-8"))
            consumer_config["key_deserializer"] = (
                lambda k: k.decode("utf-8") if k else None
            )

            self._consumer = ApacheKafkaConsumer(**consumer_config)
            logger.info(
                "Kafka consumer connected",
                extra={
                    "bootstrap_servers": self._config.bootstrap_servers,
                    "group_id": self._group_id,
                },
            )
        except ApacheKafkaError as e:
            logger.error(
                "Failed to connect Kafka consumer",
                extra={"error": str(e)},
                exc_info=True,
            )
            raise KafkaConnectionError(
                f"Failed to connect to Kafka: {str(e)}", original_error=e
            ) from e

    def subscribe(self, topics: List[str]) -> None:
        """
        Subscribe to topics.

        Args:
            topics: List of topic names to subscribe to

        Raises:
            KafkaConsumerError: If subscription fails
        """
        if not self._consumer:
            raise KafkaConsumerError("Consumer not connected")

        try:
            self._consumer.subscribe(topics)
            logger.info(
                "Subscribed to topics",
                extra={"topics": topics, "group_id": self._group_id},
            )
        except ApacheKafkaError as e:
            logger.error(
                "Failed to subscribe to topics",
                extra={"topics": topics, "error": str(e)},
                exc_info=True,
            )
            raise KafkaConsumerError(
                f"Failed to subscribe to topics: {str(e)}", original_error=e
            ) from e

    async def consume(
        self,
        topics: List[str] | str,
        handler: Callable[[Dict[str, Any]], None],
        error_handler: Optional[Callable[[Exception, Dict[str, Any]], None]] = None,
    ) -> None:
        """
        Start consuming messages from topics.

        Args:
            topics: Topic name or list of topic names
            handler: Async function to handle each message
            error_handler: Optional async function to handle errors

        Raises:
            KafkaConsumerError: If consumer fails
        """
        if isinstance(topics, str):
            topics = [topics]

        self.subscribe(topics)
        self._running = True

        logger.info(
            "Starting message consumption",
            extra={"topics": topics, "group_id": self._group_id},
        )

        try:
            while self._running:
                # Poll for messages with timeout
                messages = self._consumer.poll(timeout_ms=1000)

                for topic_partition, records in messages.items():
                    for record in records:
                        try:
                            message_data = {
                                "topic": record.topic,
                                "partition": record.partition,
                                "offset": record.offset,
                                "key": record.key,
                                "value": record.value,
                                "timestamp": record.timestamp,
                                "headers": dict(record.headers) if record.headers else {},
                            }

                            logger.debug(
                                "Processing message",
                                extra={
                                    "topic": record.topic,
                                    "partition": record.partition,
                                    "offset": record.offset,
                                },
                            )

                            # Call handler
                            if asyncio.iscoroutinefunction(handler):
                                await handler(message_data)
                            else:
                                handler(message_data)

                            # Commit offset after successful processing
                            self._consumer.commit()

                        except KafkaSerializationError as e:
                            logger.error(
                                "Failed to deserialize message",
                                extra={"error": str(e)},
                                exc_info=True,
                            )
                            if error_handler:
                                if asyncio.iscoroutinefunction(error_handler):
                                    await error_handler(e, message_data)
                                else:
                                    error_handler(e, message_data)

                        except Exception as e:
                            logger.error(
                                "Error handling message",
                                extra={"error": str(e)},
                                exc_info=True,
                            )
                            if error_handler:
                                if asyncio.iscoroutinefunction(error_handler):
                                    await error_handler(e, message_data)
                                else:
                                    error_handler(e, message_data)
                            else:
                                raise

                # Allow other async tasks to run
                await asyncio.sleep(0.01)

        except ApacheKafkaError as e:
            logger.error(
                "Kafka consumer error",
                extra={"error": str(e)},
                exc_info=True,
            )
            raise KafkaConsumerError(
                f"Consumer error: {str(e)}", original_error=e
            ) from e

    def stop(self) -> None:
        """Stop consuming messages."""
        self._running = False
        logger.info("Stopping message consumption")

    async def close(self) -> None:
        """Close the consumer connection."""
        self._running = False
        if self._consumer:
            try:
                self._consumer.close()
                logger.info("Kafka consumer closed")
            except ApacheKafkaError as e:
                logger.warning(f"Error closing consumer: {e}", exc_info=True)
            finally:
                self._consumer = None

    async def __aenter__(self) -> "KafkaConsumer":
        """Context manager entry."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        await self.close()
