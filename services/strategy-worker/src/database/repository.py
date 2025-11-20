"""Repository for strategy execution data."""

import json
from datetime import datetime
from decimal import Decimal
from typing import Any
from uuid import UUID

import asyncpg
import structlog

from .pool import DatabasePool

logger = structlog.get_logger()


class ExecutionRepository:
    """Repository for strategy execution operations."""

    def __init__(self, db_pool: DatabasePool) -> None:
        """Initialize repository.

        Args:
            db_pool: Database connection pool
        """
        self.db_pool = db_pool

    async def create_execution(
        self,
        execution_id: UUID,
        strategy_id: UUID,
        user_id: UUID,
        config: dict[str, Any],
    ) -> None:
        """Create new strategy execution record.

        Args:
            execution_id: Execution ID
            strategy_id: Strategy ID
            user_id: User ID
            config: Strategy configuration
        """
        query = """
            INSERT INTO strategy_executions (
                execution_id, strategy_id, user_id, config, status
            ) VALUES ($1, $2, $3, $4, 'active')
        """
        await self.db_pool.execute(query, execution_id, strategy_id, user_id, json.dumps(config))
        logger.info(
            "Created strategy execution",
            execution_id=str(execution_id),
            strategy_id=str(strategy_id),
        )

    async def update_execution_status(
        self, execution_id: UUID, status: str, stopped_at: datetime | None = None
    ) -> None:
        """Update execution status.

        Args:
            execution_id: Execution ID
            status: New status
            stopped_at: Stop timestamp (optional)
        """
        query = """
            UPDATE strategy_executions
            SET status = $2, stopped_at = $3
            WHERE execution_id = $1
        """
        await self.db_pool.execute(query, execution_id, status, stopped_at)
        logger.info("Updated execution status", execution_id=str(execution_id), status=status)

    async def update_positions(
        self, execution_id: UUID, positions: dict[str, Any], total_exposure: Decimal
    ) -> None:
        """Update current positions.

        Args:
            execution_id: Execution ID
            positions: Position data
            total_exposure: Total exposure amount
        """
        query = """
            UPDATE strategy_executions
            SET positions = $2, total_exposure = $3, last_checked_at = NOW()
            WHERE execution_id = $1
        """
        await self.db_pool.execute(query, execution_id, json.dumps(positions), total_exposure)

    async def update_pnl(
        self,
        execution_id: UUID,
        total_pnl: Decimal,
        realized_pnl: Decimal,
        unrealized_pnl: Decimal,
        daily_pnl: Decimal,
    ) -> None:
        """Update P&L metrics.

        Args:
            execution_id: Execution ID
            total_pnl: Total P&L
            realized_pnl: Realized P&L
            unrealized_pnl: Unrealized P&L
            daily_pnl: Daily P&L
        """
        query = """
            UPDATE strategy_executions
            SET total_pnl = $2, realized_pnl = $3, unrealized_pnl = $4, daily_pnl = $5
            WHERE execution_id = $1
        """
        await self.db_pool.execute(
            query, execution_id, total_pnl, realized_pnl, unrealized_pnl, daily_pnl
        )

    async def update_trade_stats(
        self, execution_id: UUID, total_trades: int, winning_trades: int, losing_trades: int
    ) -> None:
        """Update trade statistics.

        Args:
            execution_id: Execution ID
            total_trades: Total number of trades
            winning_trades: Number of winning trades
            losing_trades: Number of losing trades
        """
        query = """
            UPDATE strategy_executions
            SET total_trades = $2, winning_trades = $3, losing_trades = $4
            WHERE execution_id = $1
        """
        await self.db_pool.execute(query, execution_id, total_trades, winning_trades, losing_trades)

    async def get_execution(self, execution_id: UUID) -> asyncpg.Record | None:
        """Get execution by ID.

        Args:
            execution_id: Execution ID

        Returns:
            Execution record or None
        """
        query = """
            SELECT * FROM strategy_executions WHERE execution_id = $1
        """
        return await self.db_pool.fetchrow(query, execution_id)

    async def get_active_executions(self) -> list[asyncpg.Record]:
        """Get all active executions.

        Returns:
            List of active execution records
        """
        query = """
            SELECT * FROM strategy_executions WHERE status = 'active'
        """
        return await self.db_pool.fetch(query)

    async def create_order(
        self,
        order_id: UUID,
        execution_id: UUID,
        symbol: str,
        side: str,
        order_type: str,
        quantity: int,
        price: Decimal | None = None,
    ) -> None:
        """Create order record.

        Args:
            order_id: Order ID
            execution_id: Execution ID
            symbol: Trading symbol
            side: Order side (BUY/SELL)
            order_type: Order type (LIMIT/MARKET)
            quantity: Order quantity
            price: Order price (optional for market orders)
        """
        query = """
            INSERT INTO execution_orders (
                order_id, execution_id, symbol, side, order_type, quantity, price, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING')
        """
        await self.db_pool.execute(
            query, order_id, execution_id, symbol, side, order_type, quantity, price
        )
        logger.info(
            "Created order",
            order_id=str(order_id),
            execution_id=str(execution_id),
            symbol=symbol,
            side=side,
        )

    async def update_order_status(
        self,
        order_id: UUID,
        status: str,
        oms_order_id: str | None = None,
        filled_quantity: int | None = None,
        filled_price: Decimal | None = None,
        error_message: str | None = None,
    ) -> None:
        """Update order status.

        Args:
            order_id: Order ID
            status: New status
            oms_order_id: OMS order ID (optional)
            filled_quantity: Filled quantity (optional)
            filled_price: Filled price (optional)
            error_message: Error message for rejected orders (optional)
        """
        query = """
            UPDATE execution_orders
            SET status = $2, oms_order_id = $3, filled_quantity = $4,
                filled_price = $5, error_message = $6,
                submitted_at = CASE WHEN $2 = 'ACCEPTED' THEN NOW() ELSE submitted_at END,
                filled_at = CASE WHEN $2 = 'FILLED' THEN NOW() ELSE filled_at END,
                cancelled_at = CASE WHEN $2 = 'CANCELLED' THEN NOW() ELSE cancelled_at END
            WHERE order_id = $1
        """
        await self.db_pool.execute(
            query, order_id, status, oms_order_id, filled_quantity, filled_price, error_message
        )
        logger.info("Updated order status", order_id=str(order_id), status=status)

    async def get_order(self, order_id: UUID) -> asyncpg.Record | None:
        """Get order by ID.

        Args:
            order_id: Order ID

        Returns:
            Order record or None
        """
        query = """
            SELECT * FROM execution_orders WHERE order_id = $1
        """
        return await self.db_pool.fetchrow(query, order_id)

    async def get_execution_orders(self, execution_id: UUID) -> list[asyncpg.Record]:
        """Get all orders for an execution.

        Args:
            execution_id: Execution ID

        Returns:
            List of order records
        """
        query = """
            SELECT * FROM execution_orders
            WHERE execution_id = $1
            ORDER BY created_at DESC
        """
        return await self.db_pool.fetch(query, execution_id)

    async def create_position_snapshot(
        self,
        snapshot_id: UUID,
        execution_id: UUID,
        positions: dict[str, Any],
        total_value: Decimal,
        total_pnl: Decimal,
        unrealized_pnl: Decimal,
    ) -> None:
        """Create position snapshot.

        Args:
            snapshot_id: Snapshot ID
            execution_id: Execution ID
            positions: Position data
            total_value: Total position value
            total_pnl: Total P&L
            unrealized_pnl: Unrealized P&L
        """
        query = """
            INSERT INTO position_snapshots (
                snapshot_id, execution_id, positions, total_value, total_pnl, unrealized_pnl
            ) VALUES ($1, $2, $3, $4, $5, $6)
        """
        await self.db_pool.execute(
            query, snapshot_id, execution_id, json.dumps(positions), total_value, total_pnl, unrealized_pnl
        )

    async def create_risk_event(
        self,
        event_id: UUID,
        execution_id: UUID,
        event_type: str,
        severity: str,
        description: str,
        data: dict[str, Any] | None = None,
        action_taken: str | None = None,
    ) -> None:
        """Log risk event.

        Args:
            event_id: Event ID
            execution_id: Execution ID
            event_type: Event type
            severity: Severity level (INFO/WARNING/CRITICAL)
            description: Event description
            data: Additional event data (optional)
            action_taken: Action taken in response (optional)
        """
        query = """
            INSERT INTO risk_events (
                event_id, execution_id, event_type, severity, description, data, action_taken
            ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        """
        await self.db_pool.execute(
            query,
            event_id,
            execution_id,
            event_type,
            severity,
            description,
            json.dumps(data) if data else None,
            action_taken,
        )
        logger.warning(
            "Risk event logged",
            execution_id=str(execution_id),
            event_type=event_type,
            severity=severity,
        )

    async def get_daily_pnl(self, execution_id: UUID) -> Decimal:
        """Get today's P&L for an execution.

        Args:
            execution_id: Execution ID

        Returns:
            Daily P&L
        """
        query = """
            SELECT daily_pnl FROM strategy_executions WHERE execution_id = $1
        """
        result = await self.db_pool.fetchval(query, execution_id)
        return result or Decimal("0.00")

    async def reset_daily_pnl(self, execution_id: UUID) -> None:
        """Reset daily P&L to zero (called at start of trading day).

        Args:
            execution_id: Execution ID
        """
        query = """
            UPDATE strategy_executions SET daily_pnl = 0.00 WHERE execution_id = $1
        """
        await self.db_pool.execute(query, execution_id)
