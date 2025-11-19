"""
Kafka messaging package for AIPX.

Provides producer and consumer interfaces for Apache Kafka messaging.
"""

from .producer import KafkaProducer
from .consumer import KafkaConsumer
from .config import KafkaConfig
from .exceptions import (
    KafkaError,
    KafkaProducerError,
    KafkaConsumerError,
    KafkaConnectionError,
    KafkaSerializationError,
)

__all__ = [
    "KafkaProducer",
    "KafkaConsumer",
    "KafkaConfig",
    "KafkaError",
    "KafkaProducerError",
    "KafkaConsumerError",
    "KafkaConnectionError",
    "KafkaSerializationError",
]
