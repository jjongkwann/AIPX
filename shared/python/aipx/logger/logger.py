"""
Structured logging configuration
"""

import sys
import logging
from typing import Optional
import structlog
from structlog.stdlib import LoggerFactory


def configure_logging(
    level: str = "INFO",
    format_type: str = "json",
    logger_name: Optional[str] = None,
) -> None:
    """
    Configure structured logging

    Args:
        level: Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
        format_type: Output format (json or console)
        logger_name: Optional logger name
    """
    # Convert level string to logging level
    log_level = getattr(logging, level.upper(), logging.INFO)

    # Configure standard library logging
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=log_level,
    )

    # Processors for structlog
    processors = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
    ]

    # Add appropriate renderer based on format type
    if format_type == "console":
        processors.append(
            structlog.dev.ConsoleRenderer(
                colors=True,
                exception_formatter=structlog.dev.plain_traceback,
            )
        )
    else:  # json
        processors.append(structlog.processors.JSONRenderer())

    # Configure structlog
    structlog.configure(
        processors=processors,
        logger_factory=LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        cache_logger_on_first_use=True,
    )

    logger = structlog.get_logger(logger_name)
    logger.info(
        "logging_configured",
        level=level,
        format=format_type,
    )


def get_logger(name: Optional[str] = None):
    """
    Get a logger instance

    Args:
        name: Logger name (usually __name__)

    Returns:
        Structlog logger instance
    """
    return structlog.get_logger(name)
