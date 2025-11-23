"""Redis cache implementation for AIPX."""

import hashlib
import json
import logging
from functools import wraps
from typing import Any, Callable, Optional

from .client import RedisClient
from .config import RedisConfig
from .exceptions import RedisCacheError

logger = logging.getLogger(__name__)


class RedisCache:
    """
    Redis-based caching implementation for AIPX.

    Provides decorator and direct caching with automatic serialization,
    TTL management, and cache key generation.

    Example:
        >>> config = RedisConfig.from_env()
        >>> cache = RedisCache(config, default_ttl=300)
        >>> await cache.connect()
        >>>
        >>> # Direct caching
        >>> await cache.set("user:123", {"name": "John"})
        >>> user = await cache.get("user:123")
        >>>
        >>> # Decorator usage
        >>> @cache.cached(ttl=600, key_prefix="user")
        >>> async def get_user(user_id: int):
        ...     return {"id": user_id, "name": "John"}
    """

    def __init__(
        self,
        config: RedisConfig,
        default_ttl: int = 300,
        key_prefix: str = "cache",
    ) -> None:
        """
        Initialize RedisCache.

        Args:
            config: Redis configuration
            default_ttl: Default TTL in seconds
            key_prefix: Prefix for all cache keys
        """
        self._client = RedisClient(config)
        self._default_ttl = default_ttl
        self._key_prefix = key_prefix

    async def connect(self) -> None:
        """Connect to Redis."""
        await self._client.connect()

    async def close(self) -> None:
        """Close Redis connection."""
        await self._client.close()

    def _make_key(self, key: str) -> str:
        """
        Create full cache key with prefix.

        Args:
            key: Original key

        Returns:
            Prefixed cache key
        """
        return f"{self._key_prefix}:{key}"

    def _hash_args(self, *args, **kwargs) -> str:
        """
        Create hash from function arguments.

        Args:
            *args: Positional arguments
            **kwargs: Keyword arguments

        Returns:
            Hash string
        """
        key_data = json.dumps({"args": args, "kwargs": kwargs}, sort_keys=True)
        return hashlib.md5(key_data.encode()).hexdigest()

    async def get(self, key: str, default: Any = None) -> Any:
        """
        Get cached value.

        Args:
            key: Cache key
            default: Default value if not found

        Returns:
            Cached value or default

        Raises:
            RedisCacheError: If cache operation fails
        """
        try:
            full_key = self._make_key(key)
            value = await self._client.get(full_key, default)

            if value is not None and value != default:
                logger.debug("Cache hit", extra={"key": key})
            else:
                logger.debug("Cache miss", extra={"key": key})

            return value

        except Exception as e:
            logger.error(
                "Failed to get cached value",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisCacheError(f"Failed to get cached value: {str(e)}", original_error=e) from e

    async def set(
        self,
        key: str,
        value: Any,
        ttl: Optional[int] = None,
    ) -> bool:
        """
        Set cached value.

        Args:
            key: Cache key
            value: Value to cache
            ttl: Time to live in seconds (uses default if not specified)

        Returns:
            True if set successfully

        Raises:
            RedisCacheError: If cache operation fails
        """
        try:
            full_key = self._make_key(key)
            cache_ttl = ttl if ttl is not None else self._default_ttl

            result = await self._client.set(full_key, value, ttl=cache_ttl)

            logger.debug(
                "Value cached",
                extra={"key": key, "ttl": cache_ttl},
            )

            return result

        except Exception as e:
            logger.error(
                "Failed to cache value",
                extra={"key": key, "error": str(e)},
                exc_info=True,
            )
            raise RedisCacheError(f"Failed to cache value: {str(e)}", original_error=e) from e

    async def delete(self, *keys: str) -> int:
        """
        Delete cached values.

        Args:
            *keys: Cache keys to delete

        Returns:
            Number of keys deleted

        Raises:
            RedisCacheError: If cache operation fails
        """
        try:
            full_keys = [self._make_key(k) for k in keys]
            count = await self._client.delete(*full_keys)

            logger.debug(
                "Cache keys deleted",
                extra={"keys": keys, "count": count},
            )

            return count

        except Exception as e:
            logger.error(
                "Failed to delete cached values",
                extra={"keys": keys, "error": str(e)},
                exc_info=True,
            )
            raise RedisCacheError(
                f"Failed to delete cached values: {str(e)}", original_error=e
            ) from e

    async def clear_pattern(self, pattern: str) -> int:
        """
        Delete all keys matching pattern.

        Args:
            pattern: Key pattern (e.g., "user:*")

        Returns:
            Number of keys deleted

        Raises:
            RedisCacheError: If cache operation fails
        """
        try:
            full_pattern = self._make_key(pattern)
            # Note: This uses SCAN which is safe for production
            # but may not delete all keys if there are many matches
            count = 0
            async for key in self._client._client.scan_iter(match=full_pattern):
                await self._client.delete(key)
                count += 1

            logger.info(
                "Cache pattern cleared",
                extra={"pattern": pattern, "count": count},
            )

            return count

        except Exception as e:
            logger.error(
                "Failed to clear cache pattern",
                extra={"pattern": pattern, "error": str(e)},
                exc_info=True,
            )
            raise RedisCacheError(
                f"Failed to clear cache pattern: {str(e)}", original_error=e
            ) from e

    def cached(
        self,
        ttl: Optional[int] = None,
        key_prefix: Optional[str] = None,
        key_builder: Optional[Callable] = None,
    ):
        """
        Decorator to cache function results.

        Args:
            ttl: Cache TTL in seconds
            key_prefix: Prefix for cache key
            key_builder: Custom function to build cache key from args

        Returns:
            Decorated function

        Example:
            >>> @cache.cached(ttl=600, key_prefix="user")
            >>> async def get_user(user_id: int):
            ...     return {"id": user_id, "name": "John"}
        """

        def decorator(func: Callable):
            @wraps(func)
            async def wrapper(*args, **kwargs):
                # Build cache key
                if key_builder:
                    cache_key = key_builder(*args, **kwargs)
                else:
                    prefix = key_prefix or func.__name__
                    arg_hash = self._hash_args(*args, **kwargs)
                    cache_key = f"{prefix}:{arg_hash}"

                # Try to get from cache
                cached_value = await self.get(cache_key)
                if cached_value is not None:
                    return cached_value

                # Call function and cache result
                result = await func(*args, **kwargs)
                await self.set(cache_key, result, ttl=ttl)

                return result

            return wrapper

        return decorator

    async def __aenter__(self) -> "RedisCache":
        """Context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        await self.close()
