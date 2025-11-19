"""
Kafka Consumer wrapper using confluent-kafka
"""

import json
from typing import List, Callable, Optional, Any
from confluent_kafka import Consumer, KafkaError, KafkaException
import structlog

logger = structlog.get_logger(__name__)


class KafkaConsumer:
    """Kafka Consumer wrapper"""

    def __init__(
        self,
        brokers: str,
        topics: List[str],
        group_id: str,
        auto_offset_reset: str = "latest",
        enable_auto_commit: bool = True,
        auto_commit_interval_ms: int = 5000,
    ):
        """
        Initialize Kafka Consumer

        Args:
            brokers: Comma-separated list of Kafka brokers
            topics: List of topics to subscribe to
            group_id: Consumer group ID
            auto_offset_reset: Where to start reading (earliest, latest)
            enable_auto_commit: Enable automatic offset commit
            auto_commit_interval_ms: Auto commit interval in milliseconds
        """
        self.topics = topics
        self.config = {
            "bootstrap.servers": brokers,
            "group.id": group_id,
            "auto.offset.reset": auto_offset_reset,
            "enable.auto.commit": enable_auto_commit,
            "auto.commit.interval.ms": auto_commit_interval_ms,
        }

        self.consumer = Consumer(self.config)
        self.consumer.subscribe(topics)

        logger.info(
            "kafka_consumer_initialized",
            brokers=brokers,
            topics=topics,
            group_id=group_id,
        )

    def consume(
        self,
        handler: Callable[[Any], None],
        timeout: float = 1.0,
        max_messages: Optional[int] = None,
    ):
        """
        Start consuming messages

        Args:
            handler: Message handler function
            timeout: Poll timeout in seconds
            max_messages: Maximum number of messages to consume (None = infinite)
        """
        messages_consumed = 0

        try:
            while True:
                if max_messages and messages_consumed >= max_messages:
                    break

                msg = self.consumer.poll(timeout)

                if msg is None:
                    continue

                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        logger.debug(
                            "kafka_partition_eof",
                            topic=msg.topic(),
                            partition=msg.partition(),
                        )
                    else:
                        raise KafkaException(msg.error())
                else:
                    # Process message
                    try:
                        value = msg.value()
                        if value:
                            # Try to decode as JSON
                            try:
                                value = json.loads(value.decode("utf-8"))
                            except (json.JSONDecodeError, UnicodeDecodeError):
                                # Keep as bytes if not JSON
                                pass

                        handler(value)
                        messages_consumed += 1

                        logger.debug(
                            "kafka_message_consumed",
                            topic=msg.topic(),
                            partition=msg.partition(),
                            offset=msg.offset(),
                        )

                    except Exception as e:
                        logger.error(
                            "kafka_handler_error",
                            topic=msg.topic(),
                            partition=msg.partition(),
                            offset=msg.offset(),
                            error=str(e),
                        )
                        # Continue processing despite error

        except KeyboardInterrupt:
            logger.info("kafka_consumer_interrupted")

        finally:
            self.close()

    def close(self):
        """Close consumer"""
        self.consumer.close()
        logger.info("kafka_consumer_closed")


def create_consumer(
    brokers: str,
    topics: List[str],
    group_id: str,
    **kwargs,
) -> KafkaConsumer:
    """
    Factory function to create Kafka consumer

    Args:
        brokers: Kafka broker addresses
        topics: Topics to subscribe to
        group_id: Consumer group ID
        **kwargs: Additional consumer configuration

    Returns:
        KafkaConsumer instance
    """
    return KafkaConsumer(brokers, topics, group_id, **kwargs)
