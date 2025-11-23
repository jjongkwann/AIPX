"""
Redis client wrapper
"""

import json
from typing import Any, Dict, List, Optional

import redis
import structlog

logger = structlog.get_logger(__name__)


class RedisClient:
    """Redis client wrapper"""

    def __init__(
        self,
        host: str = "localhost",
        port: int = 6379,
        password: Optional[str] = None,
        db: int = 0,
        max_connections: int = 10,
        decode_responses: bool = True,
    ):
        """
        Initialize Redis client

        Args:
            host: Redis host
            port: Redis port
            password: Redis password (optional)
            db: Redis database number
            max_connections: Maximum number of connections in pool
            decode_responses: Decode responses to strings
        """
        pool = redis.ConnectionPool(
            host=host,
            port=port,
            password=password,
            db=db,
            max_connections=max_connections,
            decode_responses=decode_responses,
        )

        self.client = redis.Redis(connection_pool=pool)

        # Test connection
        try:
            self.client.ping()
            logger.info(
                "redis_connected",
                host=host,
                port=port,
                db=db,
            )
        except redis.ConnectionError as e:
            logger.error("redis_connection_failed", error=str(e))
            raise

    def set(
        self,
        key: str,
        value: Any,
        ex: Optional[int] = None,
        px: Optional[int] = None,
    ) -> bool:
        """
        Set key-value pair

        Args:
            key: Key name
            value: Value (will be JSON serialized if dict)
            ex: Expiration in seconds
            px: Expiration in milliseconds

        Returns:
            True if successful
        """
        if isinstance(value, (dict, list)):
            value = json.dumps(value)

        return self.client.set(key, value, ex=ex, px=px)

    def get(self, key: str) -> Optional[Any]:
        """
        Get value by key

        Args:
            key: Key name

        Returns:
            Value or None if not found
        """
        value = self.client.get(key)

        if value is None:
            return None

        # Try to parse as JSON
        try:
            return json.loads(value)
        except (json.JSONDecodeError, TypeError):
            return value

    def delete(self, *keys: str) -> int:
        """
        Delete one or more keys

        Args:
            *keys: Key names

        Returns:
            Number of keys deleted
        """
        return self.client.delete(*keys)

    def exists(self, *keys: str) -> int:
        """
        Check if keys exist

        Args:
            *keys: Key names

        Returns:
            Number of keys that exist
        """
        return self.client.exists(*keys)

    def expire(self, key: str, seconds: int) -> bool:
        """
        Set expiration on key

        Args:
            key: Key name
            seconds: Expiration in seconds

        Returns:
            True if successful
        """
        return self.client.expire(key, seconds)

    def hset(self, name: str, mapping: Dict[str, Any]) -> int:
        """
        Set multiple hash fields

        Args:
            name: Hash name
            mapping: Field-value dictionary

        Returns:
            Number of fields added
        """
        # Serialize dict values to JSON
        serialized = {}
        for k, v in mapping.items():
            if isinstance(v, (dict, list)):
                serialized[k] = json.dumps(v)
            else:
                serialized[k] = v

        return self.client.hset(name, mapping=serialized)

    def hget(self, name: str, key: str) -> Optional[Any]:
        """
        Get hash field value

        Args:
            name: Hash name
            key: Field name

        Returns:
            Value or None
        """
        value = self.client.hget(name, key)

        if value is None:
            return None

        try:
            return json.loads(value)
        except (json.JSONDecodeError, TypeError):
            return value

    def hgetall(self, name: str) -> Dict[str, Any]:
        """
        Get all hash fields

        Args:
            name: Hash name

        Returns:
            Dictionary of field-value pairs
        """
        data = self.client.hgetall(name)

        # Try to parse JSON values
        result = {}
        for k, v in data.items():
            try:
                result[k] = json.loads(v)
            except (json.JSONDecodeError, TypeError):
                result[k] = v

        return result

    def hdel(self, name: str, *keys: str) -> int:
        """
        Delete hash fields

        Args:
            name: Hash name
            *keys: Field names

        Returns:
            Number of fields deleted
        """
        return self.client.hdel(name, *keys)

    def lpush(self, name: str, *values: Any) -> int:
        """
        Prepend values to list

        Args:
            name: List name
            *values: Values to prepend

        Returns:
            Length of list after push
        """
        serialized = [json.dumps(v) if isinstance(v, (dict, list)) else v for v in values]
        return self.client.lpush(name, *serialized)

    def rpush(self, name: str, *values: Any) -> int:
        """
        Append values to list

        Args:
            name: List name
            *values: Values to append

        Returns:
            Length of list after push
        """
        serialized = [json.dumps(v) if isinstance(v, (dict, list)) else v for v in values]
        return self.client.rpush(name, *serialized)

    def lrange(self, name: str, start: int, end: int) -> List[Any]:
        """
        Get range of list elements

        Args:
            name: List name
            start: Start index
            end: End index

        Returns:
            List of elements
        """
        values = self.client.lrange(name, start, end)

        result = []
        for v in values:
            try:
                result.append(json.loads(v))
            except (json.JSONDecodeError, TypeError):
                result.append(v)

        return result

    def publish(self, channel: str, message: Any) -> int:
        """
        Publish message to channel

        Args:
            channel: Channel name
            message: Message (will be JSON serialized if dict)

        Returns:
            Number of subscribers that received the message
        """
        if isinstance(message, (dict, list)):
            message = json.dumps(message)

        return self.client.publish(channel, message)

    def close(self):
        """Close Redis connection"""
        self.client.close()
        logger.info("redis_connection_closed")


def create_client(
    host: str = "localhost",
    port: int = 6379,
    **kwargs,
) -> RedisClient:
    """
    Factory function to create Redis client

    Args:
        host: Redis host
        port: Redis port
        **kwargs: Additional client configuration

    Returns:
        RedisClient instance
    """
    return RedisClient(host, port, **kwargs)
