"""Kafka consumer for approved strategies."""

import asyncio
import json
from typing import Any, Callable

import structlog
from aiokafka import AIOKafkaConsumer, ConsumerRecord
from aiokafka.errors import KafkaError

from ..config import KafkaConfig

logger = structlog.get_logger()


class StrategyConsumer:
    """Consumes approved strategies from Kafka."""

    def __init__(self, config: KafkaConfig, message_handler: Callable[[dict[str, Any]], None]) -> None:
        """Initialize strategy consumer.

        Args:
            config: Kafka configuration
            message_handler: Async callback for handling messages
        """
        self.config = config
        self.message_handler = message_handler
        self._consumer: AIOKafkaConsumer | None = None
        self._running = False
        self._task: asyncio.Task | None = None

    async def start(self) -> None:
        """Start consuming messages."""
        if self._running:
            logger.warning("Consumer already running")
            return

        try:
            self._consumer = AIOKafkaConsumer(
                self.config.topics["approved"],
                bootstrap_servers=self.config.bootstrap_servers,
                group_id=self.config.consumer_group,
                auto_offset_reset=self.config.auto_offset_reset,
                enable_auto_commit=self.config.enable_auto_commit,
                max_poll_records=self.config.max_poll_records,
                session_timeout_ms=self.config.session_timeout_ms,
                value_deserializer=lambda m: json.loads(m.decode("utf-8")),
            )

            await self._consumer.start()
            logger.info(
                "Kafka consumer started",
                topic=self.config.topics["approved"],
                group=self.config.consumer_group,
            )

            self._running = True
            self._task = asyncio.create_task(self._consume_loop())

        except KafkaError as e:
            logger.error("Failed to start Kafka consumer", error=str(e))
            raise

    async def stop(self) -> None:
        """Stop consuming messages."""
        if not self._running:
            return

        self._running = False

        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass

        if self._consumer:
            await self._consumer.stop()
            logger.info("Kafka consumer stopped")

    async def _consume_loop(self) -> None:
        """Main consumption loop."""
        logger.info("Starting consumption loop")

        try:
            async for message in self._consumer:
                if not self._running:
                    break

                try:
                    await self._process_message(message)
                    # Manual commit after successful processing
                    if not self.config.enable_auto_commit:
                        await self._consumer.commit()
                except Exception as e:
                    logger.error(
                        "Error processing message",
                        error=str(e),
                        topic=message.topic,
                        partition=message.partition,
                        offset=message.offset,
                    )
                    # Continue processing other messages
                    continue

        except asyncio.CancelledError:
            logger.info("Consumption loop cancelled")
        except Exception as e:
            logger.error("Unexpected error in consumption loop", error=str(e))
            raise

    async def _process_message(self, message: ConsumerRecord) -> None:
        """Process a single Kafka message.

        Args:
            message: Kafka message record
        """
        logger.info(
            "Processing strategy message",
            topic=message.topic,
            partition=message.partition,
            offset=message.offset,
            timestamp=message.timestamp,
        )

        try:
            # Parse message value
            strategy_data = message.value

            # Validate required fields
            required_fields = ["strategy_id", "user_id", "config"]
            missing_fields = [field for field in required_fields if field not in strategy_data]

            if missing_fields:
                logger.error(
                    "Invalid strategy message: missing fields",
                    missing=missing_fields,
                    message=strategy_data,
                )
                return

            # Call message handler
            await self.message_handler(strategy_data)

            logger.info(
                "Strategy message processed successfully",
                strategy_id=strategy_data.get("strategy_id"),
            )

        except json.JSONDecodeError as e:
            logger.error("Failed to decode message JSON", error=str(e))
            raise
        except Exception as e:
            logger.error("Error in message handler", error=str(e))
            raise
