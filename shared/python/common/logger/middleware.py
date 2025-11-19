"""Logging middleware for AIPX web frameworks."""

import time
import uuid
from typing import Callable, Optional
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response

from .logger import get_logger


class LoggingMiddleware(BaseHTTPMiddleware):
    """
    FastAPI/Starlette middleware for request/response logging.

    Automatically logs all HTTP requests and responses with timing,
    status codes, and request IDs.

    Example:
        >>> from fastapi import FastAPI
        >>> from common.logger import LoggingMiddleware
        >>>
        >>> app = FastAPI()
        >>> app.add_middleware(LoggingMiddleware)
        >>>
        >>> @app.get("/users")
        >>> async def get_users():
        ...     return {"users": []}
    """

    def __init__(
        self,
        app,
        logger_name: str = "http",
        skip_paths: Optional[list[str]] = None,
    ) -> None:
        """
        Initialize LoggingMiddleware.

        Args:
            app: ASGI application
            logger_name: Name for the logger
            skip_paths: List of paths to skip logging (e.g., ["/health"])
        """
        super().__init__(app)
        self.logger = get_logger(logger_name)
        self.skip_paths = skip_paths or ["/health", "/metrics"]

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process request and response.

        Args:
            request: HTTP request
            call_next: Next middleware/handler

        Returns:
            HTTP response
        """
        # Skip logging for certain paths
        if request.url.path in self.skip_paths:
            return await call_next(request)

        # Generate request ID
        request_id = str(uuid.uuid4())
        request.state.request_id = request_id

        # Start timing
        start_time = time.time()

        # Log request
        self.logger.info(
            "HTTP request",
            request_id=request_id,
            method=request.method,
            path=request.url.path,
            query_params=str(request.query_params),
            client_host=request.client.host if request.client else None,
            user_agent=request.headers.get("user-agent"),
        )

        try:
            # Process request
            response = await call_next(request)

            # Calculate duration
            duration_ms = (time.time() - start_time) * 1000

            # Log response
            self.logger.info(
                "HTTP response",
                request_id=request_id,
                method=request.method,
                path=request.url.path,
                status_code=response.status_code,
                duration_ms=round(duration_ms, 2),
            )

            # Add request ID to response headers
            response.headers["X-Request-ID"] = request_id

            return response

        except Exception as e:
            # Calculate duration
            duration_ms = (time.time() - start_time) * 1000

            # Log error
            self.logger.error(
                "HTTP request failed",
                request_id=request_id,
                method=request.method,
                path=request.url.path,
                error=str(e),
                error_type=type(e).__name__,
                duration_ms=round(duration_ms, 2),
                exc_info=True,
            )

            raise
