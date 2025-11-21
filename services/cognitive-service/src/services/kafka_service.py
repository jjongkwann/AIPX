"""Kafka service for event publishing."""

import json
from typing import Any
from uuid import UUID

import structlog
from aiokafka import AIOKafkaProducer

from src.config import Settings

logger = structlog.get_logger()


class KafkaService:
    """Kafka producer service for publishing events."""

    def __init__(self, settings: Settings):
        """Initialize Kafka service."""
        self.settings = settings
        self._producer: AIOKafkaProducer | None = None

    async def initialize(self) -> None:
        """Initialize Kafka producer."""
        try:
            self._producer = AIOKafkaProducer(
                bootstrap_servers=self.settings.kafka_broker_list,
                value_serializer=lambda v: json.dumps(v).encode("utf-8"),
                key_serializer=lambda k: str(k).encode("utf-8") if k else None,
                compression_type="gzip",
                acks="all",
                max_request_size=1048576,  # 1MB
            )

            await self._producer.start()
            logger.info("kafka_producer_started", brokers=self.settings.kafka_brokers)

        except Exception as e:
            logger.error("kafka_producer_start_failed", error=str(e))
            raise

    async def cleanup(self) -> None:
        """Stop Kafka producer."""
        if self._producer:
            await self._producer.stop()
            logger.info("kafka_producer_stopped")

    @property
    def producer(self) -> AIOKafkaProducer:
        """Get Kafka producer instance."""
        if not self._producer:
            raise RuntimeError("Kafka producer not initialized")
        return self._producer

    async def publish_event(self, topic: str, event: dict[str, Any], key: str | UUID | None = None) -> None:
        """
        Publish an event to Kafka.

        Args:
            topic: Kafka topic
            event: Event data
            key: Optional partition key
        """
        try:
            # Convert UUID to string for serialization
            if isinstance(key, UUID):
                key = str(key)

            await self.producer.send_and_wait(topic, value=event, key=key)

            logger.info(
                "event_published",
                topic=topic,
                event_type=event.get("type"),
                key=key,
            )

        except Exception as e:
            logger.error(
                "event_publish_failed",
                topic=topic,
                error=str(e),
            )
            raise

    async def publish_strategy_created(self, strategy_id: UUID, user_id: UUID, strategy_data: dict[str, Any]) -> None:
        """Publish strategy created event."""
        event = {
            "type": "strategy.created",
            "strategy_id": str(strategy_id),
            "user_id": str(user_id),
            "timestamp": strategy_data.get("created_at"),
            "data": strategy_data,
        }

        await self.publish_event(
            topic=self.settings.kafka_topic_strategy_created,
            event=event,
            key=strategy_id,
        )

    async def publish_strategy_approved(self, strategy_id: UUID, user_id: UUID, approved_at: str) -> None:
        """Publish strategy approved event."""
        event = {
            "type": "strategy.approved",
            "strategy_id": str(strategy_id),
            "user_id": str(user_id),
            "timestamp": approved_at,
        }

        await self.publish_event(
            topic=self.settings.kafka_topic_strategy_approved,
            event=event,
            key=strategy_id,
        )

    async def publish_strategy_rejected(self, strategy_id: UUID, user_id: UUID, reason: str | None = None) -> None:
        """Publish strategy rejected event."""
        event = {
            "type": "strategy.rejected",
            "strategy_id": str(strategy_id),
            "user_id": str(user_id),
            "reason": reason,
            "timestamp": "",  # Will be set by timestamp
        }

        await self.publish_event(
            topic=self.settings.kafka_topic_strategy_rejected,
            event=event,
            key=strategy_id,
        )
