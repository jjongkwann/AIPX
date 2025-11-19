"""
AIPX Common Shared Package

This package provides common utilities for all AIPX microservices including:
- Kafka messaging (producer/consumer)
- Redis caching and distributed locking
- Structured logging with context
- Configuration management
"""

__version__ = "0.1.0"

from .kafka import KafkaProducer, KafkaConsumer, KafkaConfig
from .redis import RedisClient, RedisCache, RedisLock, RedisConfig
from .logger import Logger, LogContext, get_logger
from .config import Settings, EnvConfig

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
