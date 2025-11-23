"""Logging context management for AIPX."""

from contextlib import contextmanager
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from .logger import Logger


@contextmanager
def log_context(logger: "Logger", **kwargs: Any):
    """
    Context manager for temporary logging context.

    Adds context fields for the duration of the context, then removes them.

    Args:
        logger: Logger instance
        **kwargs: Context fields to add

    Example:
        >>> from common.logger import get_logger, log_context
        >>> logger = get_logger("my-service")
        >>>
        >>> with log_context(logger, request_id="abc-123", user_id=456):
        ...     logger.info("Processing request")  # Includes request_id and user_id
        ...     logger.info("Request completed")   # Still includes context
        >>> logger.info("Outside context")  # Context removed
    """
    # Save original context
    original_context = logger._context.copy()

    try:
        # Add new context
        logger.set_context(**kwargs)
        yield logger
    finally:
        # Restore original context
        logger._context = original_context


class LogContext:
    """
    Thread-safe logging context for AIPX.

    Provides a way to maintain logging context across async operations
    using contextvars.

    Example:
        >>> from common.logger import LogContext
        >>> context = LogContext()
        >>>
        >>> # In request handler
        >>> context.set(request_id="abc-123", user_id=456)
        >>> logger.info("Processing", **context.get())  # Includes context
        >>>
        >>> # Context is preserved across async calls
        >>> async def process():
        ...     logger.info("Processing", **context.get())  # Still has context
    """

    def __init__(self) -> None:
        """Initialize LogContext."""
        from contextvars import ContextVar

        self._context: ContextVar[dict] = ContextVar("log_context", default={})

    def set(self, **kwargs: Any) -> None:
        """
        Set context variables.

        Args:
            **kwargs: Context fields to set
        """
        current = self._context.get().copy()
        current.update(kwargs)
        self._context.set(current)

    def get(self) -> dict:
        """
        Get current context.

        Returns:
            Dictionary of context fields
        """
        return self._context.get().copy()

    def clear(self) -> None:
        """Clear context."""
        self._context.set({})

    @contextmanager
    def context(self, **kwargs: Any):
        """
        Context manager for temporary context.

        Args:
            **kwargs: Context fields

        Example:
            >>> with context.context(request_id="abc-123"):
            ...     logger.info("Processing", **context.get())
        """
        original = self._context.get().copy()

        try:
            self.set(**kwargs)
            yield self
        finally:
            self._context.set(original)
