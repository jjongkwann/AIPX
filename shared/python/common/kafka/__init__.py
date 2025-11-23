"""
Kafka messaging package for AIPX.

Provides producer and consumer interfaces for Apache Kafka messaging.
"""

from .config import KafkaConfig
from .consumer import KafkaConsumer
from .exceptions import (
    KafkaConnectionError,
    KafkaConsumerError,
    KafkaError,
    KafkaProducerError,
    KafkaSerializationError,
)
from .producer import KafkaProducer

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
