"""Structured logging implementation for AIPX."""

import json
import logging
import sys
import traceback
from datetime import datetime
from typing import Any, Dict, Optional


class JSONFormatter(logging.Formatter):
    """
    JSON log formatter for structured logging.

    Outputs logs in JSON format with consistent fields for easy parsing
    by log aggregation systems (ELK, Splunk, etc.).
    """

    def format(self, record: logging.LogRecord) -> str:
        """
        Format log record as JSON.

        Args:
            record: Log record to format

        Returns:
            JSON formatted log string
        """
        log_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
            "module": record.module,
            "function": record.funcName,
            "line": record.lineno,
        }

        # Add extra fields
        if hasattr(record, "extra_data"):
            log_data.update(record.extra_data)

        # Add exception info if present
        if record.exc_info:
            log_data["exception"] = {
                "type": record.exc_info[0].__name__,
                "message": str(record.exc_info[1]),
                "traceback": "".join(traceback.format_exception(*record.exc_info)),
            }

        # Add context if present
        if hasattr(record, "context"):
            log_data["context"] = record.context

        return json.dumps(log_data)


class Logger:
    """
    Structured logger for AIPX.

    Provides a wrapper around Python's logging with structured JSON output,
    context management, and consistent formatting.

    Example:
        >>> logger = Logger("my-service")
        >>> logger.info("User logged in", user_id=123, ip="192.168.1.1")
        >>> logger.error("Database connection failed", error=str(e))
        >>>
        >>> with logger.context(request_id="abc-123"):
        ...     logger.info("Processing request")  # Includes request_id
    """

    def __init__(
        self,
        name: str,
        level: int = logging.INFO,
        use_json: bool = True,
    ) -> None:
        """
        Initialize Logger.

        Args:
            name: Logger name (usually service or module name)
            level: Logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
            use_json: Whether to use JSON formatting
        """
        self._logger = logging.getLogger(name)
        self._logger.setLevel(level)
        self._logger.propagate = False
        self._context: Dict[str, Any] = {}

        # Remove existing handlers
        self._logger.handlers.clear()

        # Add console handler
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(level)

        if use_json:
            formatter = JSONFormatter()
        else:
            formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")

        handler.setFormatter(formatter)
        self._logger.addHandler(handler)

    def _log(
        self,
        level: int,
        message: str,
        exc_info: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        Internal log method.

        Args:
            level: Log level
            message: Log message
            exc_info: Whether to include exception info
            **kwargs: Additional fields to log
        """
        # Merge context and kwargs
        extra_data = {**self._context, **kwargs}

        # Create log record with extra data
        extra = {"extra_data": extra_data}
        if self._context:
            extra["context"] = self._context

        self._logger.log(level, message, exc_info=exc_info, extra=extra)

    def debug(self, message: str, **kwargs: Any) -> None:
        """
        Log debug message.

        Args:
            message: Log message
            **kwargs: Additional fields
        """
        self._log(logging.DEBUG, message, **kwargs)

    def info(self, message: str, **kwargs: Any) -> None:
        """
        Log info message.

        Args:
            message: Log message
            **kwargs: Additional fields
        """
        self._log(logging.INFO, message, **kwargs)

    def warning(self, message: str, **kwargs: Any) -> None:
        """
        Log warning message.

        Args:
            message: Log message
            **kwargs: Additional fields
        """
        self._log(logging.WARNING, message, **kwargs)

    def error(
        self,
        message: str,
        exc_info: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        Log error message.

        Args:
            message: Log message
            exc_info: Whether to include exception info
            **kwargs: Additional fields
        """
        self._log(logging.ERROR, message, exc_info=exc_info, **kwargs)

    def critical(
        self,
        message: str,
        exc_info: bool = False,
        **kwargs: Any,
    ) -> None:
        """
        Log critical message.

        Args:
            message: Log message
            exc_info: Whether to include exception info
            **kwargs: Additional fields
        """
        self._log(logging.CRITICAL, message, exc_info=exc_info, **kwargs)

    def exception(self, message: str, **kwargs: Any) -> None:
        """
        Log exception with traceback.

        Args:
            message: Log message
            **kwargs: Additional fields
        """
        self._log(logging.ERROR, message, exc_info=True, **kwargs)

    def set_context(self, **kwargs: Any) -> None:
        """
        Set logging context.

        Context fields will be included in all subsequent log messages.

        Args:
            **kwargs: Context fields
        """
        self._context.update(kwargs)

    def clear_context(self) -> None:
        """Clear logging context."""
        self._context.clear()

    def context(self, **kwargs: Any):
        """
        Context manager for temporary logging context.

        Args:
            **kwargs: Context fields

        Example:
            >>> with logger.context(request_id="abc-123"):
            ...     logger.info("Processing")  # Includes request_id
        """
        from .context import log_context

        return log_context(self, **kwargs)


def get_logger(
    name: str,
    level: Optional[int] = None,
    use_json: bool = True,
) -> Logger:
    """
    Get or create a logger instance.

    Args:
        name: Logger name
        level: Logging level (uses INFO if not specified)
        use_json: Whether to use JSON formatting

    Returns:
        Logger instance

    Example:
        >>> logger = get_logger("my-service")
        >>> logger.info("Service started")
    """
    if level is None:
        import os

        level_name = os.getenv("LOG_LEVEL", "INFO").upper()
        level = getattr(logging, level_name, logging.INFO)

    return Logger(name, level=level, use_json=use_json)


def setup_logging(
    level: Optional[str] = None,
    use_json: bool = True,
) -> None:
    """
    Setup global logging configuration.

    Args:
        level: Logging level name (DEBUG, INFO, WARNING, ERROR, CRITICAL)
        use_json: Whether to use JSON formatting

    Example:
        >>> setup_logging(level="DEBUG", use_json=True)
    """
    import os

    if level is None:
        level = os.getenv("LOG_LEVEL", "INFO").upper()

    logging_level = getattr(logging, level, logging.INFO)

    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(logging_level)
    root_logger.handlers.clear()

    # Add console handler
    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(logging_level)

    if use_json:
        formatter = JSONFormatter()
    else:
        formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")

    handler.setFormatter(formatter)
    root_logger.addHandler(handler)

    # Suppress noisy loggers
    logging.getLogger("urllib3").setLevel(logging.WARNING)
    logging.getLogger("kafka").setLevel(logging.WARNING)
    logging.getLogger("aiokafka").setLevel(logging.WARNING)
