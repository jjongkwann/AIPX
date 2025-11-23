"""
AIPX Common Shared Package

This package provides common utilities for all AIPX microservices including:
- Kafka messaging (producer/consumer)
- Redis caching and distributed locking
- Structured logging with context
- Configuration management
"""

__version__ = "0.1.0"

from .config import EnvConfig, Settings
from .kafka import KafkaConfig, KafkaConsumer, KafkaProducer
from .logger import LogContext, Logger, get_logger
from .redis import RedisCache, RedisClient, RedisConfig, RedisLock

__all__ = [
    # Kafka
    "KafkaProducer",
    "KafkaConsumer",
    "KafkaConfig",
    # Redis
    "RedisClient",
    "RedisCache",
    "RedisLock",
    "RedisConfig",
    # Logger
    "Logger",
    "LogContext",
    "get_logger",
    # Config
    "Settings",
    "EnvConfig",
]
