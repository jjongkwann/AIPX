"""Redis client implementation for AIPX."""

import logging
from typing import Any, Optional, Dict, List
import json

import redis.asyncio as aioredis
from redis.exceptions import RedisError as BaseRedisError

from .config import RedisConfig
from .exceptions import RedisConnectionError, RedisError

logger = logging.getLogger(__name__)


class RedisClient:
    """
    Async Redis client wrapper for AIPX.

    Provides a high-level interface for Redis operations with automatic
    connection management, JSON serialization, and error handling.

    Example:
        >>> config = RedisConfig.from_env()
        >>> client = RedisClient(config)
        >>> await client.connect()
        >>> await client.set("key", {"value": 123})
        >>> data = await client.get("key")
        >>> await client.close()

    Or using context manager:
        >>> async with RedisClient(config) as client:
        ...     await client.set("key", "value")
    """

    def __init__(self, config: RedisConfig) -> None:
        """
        Initialize RedisClient.

        Args:
            config: Redis configuration
        """
        self._config = config
        self._client: Optional[aioredis.Redis] = None
        self._pool: Optional[aioredis.ConnectionPool] = None

    async def connect(self) -> None:
        """
        Establish connection to Redis.

        Raises:
            RedisConnectionError: If connection fails
        """
        try:
            self._pool = aioredis.ConnectionPool(**self._config.to_dict())
            self._client = aioredis.Redis(connection_pool=self._pool)

            # Test connection
            await self._client.ping()

            logger.info(
                "Redis client connected",
                extra={"host": self._config.host, "port": self._config.port, "db": self._config.db},
            )
        except BaseRedisError as e:
            logger.error(
                "Failed to connect to Redis",
                extra={"error": str(e)},
                exc_info=True,
            )
            raise RedisConnectionError(
                f"Failed to connect to Redis: {str(e)}", original_error=e
            ) from e

    async def close(self) -> None:
        """Close the Redis connection."""
        if self._client:
            try:
                await self._client.close()
                logger.info("Redis client closed")
            except BaseRedisError as e:
                logger.warning(f"Error closing Redis client: {e}", exc_info=True)
            finally:
                self._client = None

        if self._pool:
            try:
                await self._pool.disconnect()
            except BaseRedisError as e:
                logger.warning(f"Error closing Redis pool: {e}", exc_info=True)
            finally:
                self._pool = None

    def _ensure_connected(self) -> None:
        """Ensure client is connected."""
        if not self._client:
            raise RedisConnectionError("Redis client not connected")

    async def get(self, key: str, default: Any = None) -> Any:
        """
        Get value by key.

        Args:
            key: Redis key
            default: Default value if key doesn't exist

        Returns:
            Value or default

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            value = await self._client.get(key)
            if value is None:
                return default

            # Try to parse as JSON
            try:
                return json.loads(value)
            except (json.JSONDecodeError, TypeError):
                return value

        except BaseRedisError as e:
            logger.error(
                "Failed to get key",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to get key {key}: {str(e)}", original_error=e) from e

    async def set(
        self,
        key: str,
        value: Any,
        ttl: Optional[int] = None,
        nx: bool = False,
        xx: bool = False,
    ) -> bool:
        """
        Set key to value.

        Args:
            key: Redis key
            value: Value to set (will be JSON serialized if dict/list)
            ttl: Time to live in seconds
            nx: Only set if key doesn't exist
            xx: Only set if key exists

        Returns:
            True if set successfully, False otherwise

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            # Serialize value if necessary
            if isinstance(value, (dict, list)):
                value = json.dumps(value)

            result = await self._client.set(key, value, ex=ttl, nx=nx, xx=xx)
            return bool(result)

        except BaseRedisError as e:
            logger.error(
                "Failed to set key",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to set key {key}: {str(e)}", original_error=e) from e

    async def delete(self, *keys: str) -> int:
        """
        Delete one or more keys.

        Args:
            *keys: Keys to delete

        Returns:
            Number of keys deleted

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            return await self._client.delete(*keys)
        except BaseRedisError as e:
            logger.error(
                "Failed to delete keys",
                extra={"keys": keys, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to delete keys: {str(e)}", original_error=e) from e

    async def exists(self, *keys: str) -> int:
        """
        Check if keys exist.

        Args:
            *keys: Keys to check

        Returns:
            Number of existing keys

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            return await self._client.exists(*keys)
        except BaseRedisError as e:
            logger.error(
                "Failed to check key existence",
                extra={"keys": keys, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to check existence: {str(e)}", original_error=e) from e

    async def expire(self, key: str, seconds: int) -> bool:
        """
        Set key expiration time.

        Args:
            key: Redis key
            seconds: Expiration time in seconds

        Returns:
            True if expiration was set, False if key doesn't exist

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            return await self._client.expire(key, seconds)
        except BaseRedisError as e:
            logger.error(
                "Failed to set expiration",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to set expiration: {str(e)}", original_error=e) from e

    async def incr(self, key: str, amount: int = 1) -> int:
        """
        Increment key value.

        Args:
            key: Redis key
            amount: Amount to increment

        Returns:
            New value after increment

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            return await self._client.incrby(key, amount)
        except BaseRedisError as e:
            logger.error(
                "Failed to increment key",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to increment key: {str(e)}", original_error=e) from e

    async def hset(self, name: str, mapping: Dict[str, Any]) -> int:
        """
        Set hash field values.

        Args:
            name: Hash name
            mapping: Field-value mapping

        Returns:
            Number of fields that were added

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            # Serialize dict/list values
            serialized_mapping = {}
            for k, v in mapping.items():
                if isinstance(v, (dict, list)):
                    serialized_mapping[k] = json.dumps(v)
                else:
                    serialized_mapping[k] = v

            return await self._client.hset(name, mapping=serialized_mapping)
        except BaseRedisError as e:
            logger.error(
                "Failed to set hash",
                extra={"name": name, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to set hash: {str(e)}", original_error=e) from e

    async def hgetall(self, name: str) -> Dict[str, Any]:
        """
        Get all hash field values.

        Args:
            name: Hash name

        Returns:
            Dictionary of field-value pairs

        Raises:
            RedisError: If operation fails
        """
        self._ensure_connected()

        try:
            data = await self._client.hgetall(name)

            # Try to deserialize JSON values
            result = {}
            for k, v in data.items():
                try:
                    result[k] = json.loads(v)
                except (json.JSONDecodeError, TypeError):
                    result[k] = v

            return result
        except BaseRedisError as e:
            logger.error(
                "Failed to get hash",
                extra={"name": name, "error": str(e)},
                exc_info=True,
            )
            raise RedisError(f"Failed to get hash: {str(e)}", original_error=e) from e

    async def __aenter__(self) -> "RedisClient":
        """Context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        await self.close()
