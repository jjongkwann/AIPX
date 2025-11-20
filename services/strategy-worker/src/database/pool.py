"""Database connection pool management."""

import asyncpg
import structlog

from ..config import DatabaseConfig

logger = structlog.get_logger()


class DatabasePool:
    """Manages PostgreSQL connection pool."""

    def __init__(self, config: DatabaseConfig) -> None:
        """Initialize database pool.

        Args:
            config: Database configuration
        """
        self.config = config
        self._pool: asyncpg.Pool | None = None

    async def connect(self) -> None:
        """Create connection pool."""
        if self._pool is not None:
            logger.warning("Database pool already connected")
            return

        try:
            self._pool = await asyncpg.create_pool(
                dsn=self.config.dsn,
                min_size=self.config.min_pool_size,
                max_size=self.config.max_pool_size,
                command_timeout=self.config.command_timeout,
            )
            logger.info(
                "Database pool connected",
                host=self.config.host,
                database=self.config.database,
                pool_size=f"{self.config.min_pool_size}-{self.config.max_pool_size}",
            )
        except Exception as e:
            logger.error("Failed to connect database pool", error=str(e))
            raise

    async def close(self) -> None:
        """Close connection pool."""
        if self._pool is None:
            return

        try:
            await self._pool.close()
            logger.info("Database pool closed")
        except Exception as e:
            logger.error("Error closing database pool", error=str(e))
        finally:
            self._pool = None

    @property
    def pool(self) -> asyncpg.Pool:
        """Get connection pool.

        Returns:
            Connection pool

        Raises:
            RuntimeError: If pool is not connected
        """
        if self._pool is None:
            raise RuntimeError("Database pool not connected. Call connect() first.")
        return self._pool

    async def execute(self, query: str, *args: any) -> str:
        """Execute a command and return status.

        Args:
            query: SQL query
            *args: Query parameters

        Returns:
            Command execution status
        """
        async with self._pool.acquire() as conn:
            return await conn.execute(query, *args)

    async def fetch(self, query: str, *args: any) -> list[asyncpg.Record]:
        """Fetch multiple rows.

        Args:
            query: SQL query
            *args: Query parameters

        Returns:
            List of records
        """
        async with self._pool.acquire() as conn:
            return await conn.fetch(query, *args)

    async def fetchrow(self, query: str, *args: any) -> asyncpg.Record | None:
        """Fetch a single row.

        Args:
            query: SQL query
            *args: Query parameters

        Returns:
            Single record or None
        """
        async with self._pool.acquire() as conn:
            return await conn.fetchrow(query, *args)

    async def fetchval(self, query: str, *args: any) -> any:
        """Fetch a single value.

        Args:
            query: SQL query
            *args: Query parameters

        Returns:
            Single value
        """
        async with self._pool.acquire() as conn:
            return await conn.fetchval(query, *args)
