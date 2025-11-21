"""Database service for PostgreSQL operations."""

import json
from datetime import datetime
from typing import Any
from uuid import UUID

import asyncpg
import structlog

from src.config import Settings
from src.schemas.strategy_config import StrategyConfig
from src.schemas.user_profile import UserProfile

logger = structlog.get_logger()


class DatabaseService:
    """PostgreSQL database service using asyncpg."""

    def __init__(self, settings: Settings):
        """Initialize database service."""
        self.settings = settings
        self._pool: asyncpg.Pool | None = None

    async def initialize(self) -> None:
        """Initialize database connection pool."""
        try:
            self._pool = await asyncpg.create_pool(
                host=self.settings.postgres_host,
                port=self.settings.postgres_port,
                database=self.settings.postgres_db,
                user=self.settings.postgres_user,
                password=self.settings.postgres_password,
                min_size=self.settings.postgres_pool_size,
                max_size=self.settings.postgres_pool_size + self.settings.postgres_max_overflow,
                command_timeout=60,
            )
            logger.info(
                "database_pool_created",
                host=self.settings.postgres_host,
                database=self.settings.postgres_db,
            )

        except Exception as e:
            logger.error("database_pool_creation_failed", error=str(e))
            raise

    async def cleanup(self) -> None:
        """Close database connection pool."""
        if self._pool:
            await self._pool.close()
            logger.info("database_pool_closed")

    @property
    def pool(self) -> asyncpg.Pool:
        """Get database connection pool."""
        if not self._pool:
            raise RuntimeError("Database pool not initialized")
        return self._pool

    # User Profile Operations

    async def get_user_profile(self, user_id: UUID) -> UserProfile | None:
        """Get user profile by ID."""
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                SELECT user_id, risk_tolerance, capital, preferred_sectors,
                       investment_horizon, created_at, updated_at
                FROM user_profiles
                WHERE user_id = $1
                """,
                user_id,
            )

            if not row:
                return None

            return UserProfile(
                user_id=row["user_id"],
                risk_tolerance=row["risk_tolerance"],
                capital=row["capital"],
                preferred_sectors=row["preferred_sectors"] or [],
                investment_horizon=row["investment_horizon"],
                created_at=row["created_at"],
                updated_at=row["updated_at"],
            )

    async def create_user_profile(self, profile: UserProfile) -> UserProfile:
        """Create a new user profile."""
        async with self.pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO user_profiles (
                    user_id, risk_tolerance, capital, preferred_sectors,
                    investment_horizon, created_at, updated_at
                )
                VALUES ($1, $2, $3, $4, $5, $6, $7)
                ON CONFLICT (user_id) DO UPDATE SET
                    risk_tolerance = EXCLUDED.risk_tolerance,
                    capital = EXCLUDED.capital,
                    preferred_sectors = EXCLUDED.preferred_sectors,
                    investment_horizon = EXCLUDED.investment_horizon,
                    updated_at = EXCLUDED.updated_at
                """,
                profile.user_id,
                profile.risk_tolerance,
                profile.capital,
                profile.preferred_sectors,
                profile.investment_horizon,
                profile.created_at,
                profile.updated_at,
            )

            logger.info("user_profile_created", user_id=str(profile.user_id))
            return profile

    async def update_user_profile(self, user_id: UUID, updates: dict[str, Any]) -> UserProfile | None:
        """Update user profile."""
        updates["updated_at"] = datetime.utcnow()

        set_clause = ", ".join([f"{key} = ${i + 2}" for i, key in enumerate(updates.keys())])
        values = [user_id] + list(updates.values())

        async with self.pool.acquire() as conn:
            await conn.execute(
                f"""
                UPDATE user_profiles
                SET {set_clause}
                WHERE user_id = $1
                """,
                *values,
            )

        return await self.get_user_profile(user_id)

    # Chat Session Operations

    async def create_session(self, user_id: UUID, session_id: UUID) -> dict[str, Any]:
        """Create a new chat session."""
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                INSERT INTO chat_sessions (session_id, user_id, started_at)
                VALUES ($1, $2, $3)
                RETURNING session_id, user_id, started_at, message_count
                """,
                session_id,
                user_id,
                datetime.utcnow(),
            )

            logger.info("session_created", session_id=str(session_id), user_id=str(user_id))
            return dict(row)

    async def get_session(self, session_id: UUID) -> dict[str, Any] | None:
        """Get session by ID."""
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                SELECT session_id, user_id, started_at, ended_at, message_count
                FROM chat_sessions
                WHERE session_id = $1
                """,
                session_id,
            )

            return dict(row) if row else None

    async def end_session(self, session_id: UUID) -> None:
        """End a chat session."""
        async with self.pool.acquire() as conn:
            await conn.execute(
                """
                UPDATE chat_sessions
                SET ended_at = $1
                WHERE session_id = $2
                """,
                datetime.utcnow(),
                session_id,
            )

            logger.info("session_ended", session_id=str(session_id))

    async def increment_message_count(self, session_id: UUID) -> None:
        """Increment message count for a session."""
        async with self.pool.acquire() as conn:
            await conn.execute(
                """
                UPDATE chat_sessions
                SET message_count = message_count + 1
                WHERE session_id = $1
                """,
                session_id,
            )

    # Strategy Operations

    async def create_strategy(self, strategy: StrategyConfig) -> StrategyConfig:
        """Create a new strategy configuration."""
        config_json = {
            "asset_allocation": strategy.asset_allocation,
            "max_position_size": strategy.max_position_size,
            "max_drawdown": strategy.max_drawdown,
            "stop_loss_pct": strategy.stop_loss_pct,
            "take_profit_pct": strategy.take_profit_pct,
            "entry_conditions": strategy.entry_conditions,
            "exit_conditions": strategy.exit_conditions,
            "indicators": strategy.indicators,
        }

        async with self.pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO strategies (
                    strategy_id, user_id, name, config, status, created_at
                )
                VALUES ($1, $2, $3, $4, $5, $6)
                """,
                strategy.strategy_id,
                strategy.user_id,
                strategy.name,
                json.dumps(config_json),
                strategy.status,
                strategy.created_at,
            )

            logger.info("strategy_created", strategy_id=str(strategy.strategy_id))
            return strategy

    async def get_strategy(self, strategy_id: UUID) -> StrategyConfig | None:
        """Get strategy by ID."""
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                SELECT strategy_id, user_id, name, config, status,
                       created_at, approved_at
                FROM strategies
                WHERE strategy_id = $1
                """,
                strategy_id,
            )

            if not row:
                return None

            config = json.loads(row["config"])

            return StrategyConfig(
                strategy_id=row["strategy_id"],
                user_id=row["user_id"],
                name=row["name"],
                description=config.get("description", ""),
                asset_allocation=config["asset_allocation"],
                max_position_size=config["max_position_size"],
                max_drawdown=config["max_drawdown"],
                stop_loss_pct=config["stop_loss_pct"],
                take_profit_pct=config.get("take_profit_pct", 0.2),
                entry_conditions=config.get("entry_conditions", []),
                exit_conditions=config.get("exit_conditions", []),
                indicators=config.get("indicators", {}),
                status=row["status"],
                created_at=row["created_at"],
                approved_at=row["approved_at"],
            )

    async def approve_strategy(self, strategy_id: UUID) -> None:
        """Approve a strategy."""
        async with self.pool.acquire() as conn:
            await conn.execute(
                """
                UPDATE strategies
                SET status = 'approved', approved_at = $1
                WHERE strategy_id = $2
                """,
                datetime.utcnow(),
                strategy_id,
            )

            logger.info("strategy_approved", strategy_id=str(strategy_id))

    async def list_user_strategies(self, user_id: UUID, limit: int = 50) -> list[StrategyConfig]:
        """List strategies for a user."""
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT strategy_id, user_id, name, config, status,
                       created_at, approved_at
                FROM strategies
                WHERE user_id = $1
                ORDER BY created_at DESC
                LIMIT $2
                """,
                user_id,
                limit,
            )

            strategies = []
            for row in rows:
                config = json.loads(row["config"])
                strategies.append(
                    StrategyConfig(
                        strategy_id=row["strategy_id"],
                        user_id=row["user_id"],
                        name=row["name"],
                        description=config.get("description", ""),
                        asset_allocation=config["asset_allocation"],
                        max_position_size=config["max_position_size"],
                        max_drawdown=config["max_drawdown"],
                        stop_loss_pct=config["stop_loss_pct"],
                        take_profit_pct=config.get("take_profit_pct", 0.2),
                        entry_conditions=config.get("entry_conditions", []),
                        exit_conditions=config.get("exit_conditions", []),
                        indicators=config.get("indicators", {}),
                        status=row["status"],
                        created_at=row["created_at"],
                        approved_at=row["approved_at"],
                    )
                )

            return strategies
