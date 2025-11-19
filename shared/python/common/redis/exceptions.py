"""Redis-related exceptions for AIPX."""


class RedisError(Exception):
    """Base exception for all Redis-related errors."""

    def __init__(self, message: str, original_error: Exception | None = None) -> None:
        """
        Initialize RedisError.

        Args:
            message: Error message
            original_error: Original exception that caused this error
        """
        super().__init__(message)
        self.original_error = original_error


class RedisConnectionError(RedisError):
    """Exception raised when Redis connection fails."""

    pass


class RedisLockError(RedisError):
    """Exception raised when distributed lock operations fail."""

    pass


class RedisCacheError(RedisError):
    """Exception raised when cache operations fail."""

    pass
