"""Redis distributed lock implementation for AIPX."""

import asyncio
import logging
import uuid
from typing import Optional

from .client import RedisClient
from .exceptions import RedisLockError

logger = logging.getLogger(__name__)


class RedisLock:
    """
    Redis-based distributed lock for AIPX.

    Provides distributed locking mechanism using Redis with automatic
    expiration and deadlock prevention.

    Example:
        >>> config = RedisConfig.from_env()
        >>> client = RedisClient(config)
        >>> await client.connect()
        >>>
        >>> lock = RedisLock(client, "resource:123", ttl=30)
        >>> async with lock:
        ...     # Critical section - only one process can execute this
        ...     await process_resource()

    Or manual acquire/release:
        >>> lock = RedisLock(client, "resource:123")
        >>> if await lock.acquire(timeout=10):
        ...     try:
        ...         await process_resource()
        ...     finally:
        ...         await lock.release()
    """

    def __init__(
        self,
        client: RedisClient,
        resource: str,
        ttl: int = 30,
        key_prefix: str = "lock",
    ) -> None:
        """
        Initialize RedisLock.

        Args:
            client: Redis client instance
            resource: Resource identifier to lock
            ttl: Lock time-to-live in seconds (prevents deadlocks)
            key_prefix: Prefix for lock keys
        """
        self._client = client
        self._resource = resource
        self._ttl = ttl
        self._key = f"{key_prefix}:{resource}"
        self._identifier = str(uuid.uuid4())
        self._locked = False

    async def acquire(
        self,
        timeout: Optional[float] = None,
        retry_interval: float = 0.1,
    ) -> bool:
        """
        Acquire the lock.

        Args:
            timeout: Maximum time to wait for lock in seconds (None = wait forever)
            retry_interval: Time between retry attempts in seconds

        Returns:
            True if lock acquired, False if timeout

        Raises:
            RedisLockError: If lock operation fails
        """
        start_time = asyncio.get_event_loop().time()

        try:
            while True:
                # Try to acquire lock (SET NX with expiration)
                acquired = await self._client.set(
                    self._key,
                    self._identifier,
                    ttl=self._ttl,
                    nx=True,  # Only set if doesn't exist
                )

                if acquired:
                    self._locked = True
                    logger.debug(
                        "Lock acquired",
                        extra={
                            "resource": self._resource,
                            "identifier": self._identifier,
                            "ttl": self._ttl,
                        },
                    )
                    return True

                # Check timeout
                if timeout is not None:
                    elapsed = asyncio.get_event_loop().time() - start_time
                    if elapsed >= timeout:
                        logger.debug(
                            "Lock acquisition timeout",
                            extra={
                                "resource": self._resource,
                                "timeout": timeout,
                            },
                        )
                        return False

                # Wait before retry
                await asyncio.sleep(retry_interval)

        except Exception as e:
            logger.error(
                "Failed to acquire lock",
                extra={"resource": self._resource, "error": str(e)},
                exc_info=True,
            )
            raise RedisLockError(f"Failed to acquire lock: {str(e)}", original_error=e) from e

    async def release(self) -> bool:
        """
        Release the lock.

        Returns:
            True if lock was released, False if lock wasn't held

        Raises:
            RedisLockError: If lock operation fails
        """
        if not self._locked:
            logger.warning(
                "Attempted to release lock that wasn't acquired",
                extra={"resource": self._resource},
            )
            return False

        try:
            # Use Lua script to ensure we only delete our own lock
            # This prevents race conditions
            lua_script = """
            if redis.call("get", KEYS[1]) == ARGV[1] then
                return redis.call("del", KEYS[1])
            else
                return 0
            end
            """

            # Execute Lua script
            result = await self._client._client.eval(
                lua_script,
                1,
                self._key,
                self._identifier,
            )

            released = bool(result)

            if released:
                self._locked = False
                logger.debug(
                    "Lock released",
                    extra={
                        "resource": self._resource,
                        "identifier": self._identifier,
                    },
                )
            else:
                logger.warning(
                    "Lock was not released (may have expired)",
                    extra={"resource": self._resource},
                )

            return released

        except Exception as e:
            logger.error(
                "Failed to release lock",
                extra={"resource": self._resource, "error": str(e)},
                exc_info=True,
            )
            raise RedisLockError(f"Failed to release lock: {str(e)}", original_error=e) from e

    async def extend(self, additional_ttl: int) -> bool:
        """
        Extend lock expiration time.

        Args:
            additional_ttl: Additional time in seconds

        Returns:
            True if lock was extended, False if lock wasn't held

        Raises:
            RedisLockError: If lock operation fails
        """
        if not self._locked:
            return False

        try:
            # Use Lua script to extend only our own lock
            lua_script = """
            if redis.call("get", KEYS[1]) == ARGV[1] then
                return redis.call("expire", KEYS[1], ARGV[2])
            else
                return 0
            end
            """

            result = await self._client._client.eval(
                lua_script,
                1,
                self._key,
                self._identifier,
                additional_ttl,
            )

            extended = bool(result)

            if extended:
                logger.debug(
                    "Lock extended",
                    extra={
                        "resource": self._resource,
                        "additional_ttl": additional_ttl,
                    },
                )
            else:
                logger.warning(
                    "Lock extension failed (may have expired)",
                    extra={"resource": self._resource},
                )

            return extended

        except Exception as e:
            logger.error(
                "Failed to extend lock",
                extra={"resource": self._resource, "error": str(e)},
                exc_info=True,
            )
            raise RedisLockError(f"Failed to extend lock: {str(e)}", original_error=e) from e

    async def is_locked(self) -> bool:
        """
        Check if resource is currently locked.

        Returns:
            True if resource is locked (by anyone)

        Raises:
            RedisLockError: If lock operation fails
        """
        try:
            exists = await self._client.exists(self._key)
            return exists > 0
        except Exception as e:
            logger.error(
                "Failed to check lock status",
                extra={"resource": self._resource, "error": str(e)},
                exc_info=True,
            )
            raise RedisLockError(f"Failed to check lock status: {str(e)}", original_error=e) from e

    async def __aenter__(self) -> "RedisLock":
        """Context manager entry."""
        await self.acquire()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        await self.release()
