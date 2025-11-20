"""Position monitoring and P&L tracking."""

import asyncio
from dataclasses import dataclass
from decimal import Decimal
from typing import Any
from uuid import UUID, uuid4

import structlog

from ..database import ExecutionRepository

logger = structlog.get_logger()


@dataclass
class Position:
    """Trading position."""

    symbol: str
    quantity: int
    entry_price: Decimal
    current_price: Decimal
    side: str  # "LONG" or "SHORT"

    @property
    def market_value(self) -> Decimal:
        """Calculate current market value."""
        return Decimal(self.quantity) * self.current_price

    @property
    def cost_basis(self) -> Decimal:
        """Calculate cost basis."""
        return Decimal(self.quantity) * self.entry_price

    @property
    def unrealized_pnl(self) -> Decimal:
        """Calculate unrealized P&L."""
        if self.side == "LONG":
            return self.market_value - self.cost_basis
        else:  # SHORT
            return self.cost_basis - self.market_value

    @property
    def unrealized_pnl_pct(self) -> Decimal:
        """Calculate unrealized P&L percentage."""
        if self.cost_basis == 0:
            return Decimal(0)
        return (self.unrealized_pnl / self.cost_basis) * Decimal(100)

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary."""
        return {
            "symbol": self.symbol,
            "quantity": self.quantity,
            "entry_price": float(self.entry_price),
            "current_price": float(self.current_price),
            "side": self.side,
            "market_value": float(self.market_value),
            "cost_basis": float(self.cost_basis),
            "unrealized_pnl": float(self.unrealized_pnl),
            "unrealized_pnl_pct": float(self.unrealized_pnl_pct),
        }


class PositionMonitor:
    """Monitors positions and tracks P&L."""

    def __init__(
        self,
        execution_id: UUID,
        repository: ExecutionRepository,
        check_interval: int = 60,
    ) -> None:
        """Initialize position monitor.

        Args:
            execution_id: Execution ID to monitor
            repository: Execution repository
            check_interval: Position check interval in seconds
        """
        self.execution_id = execution_id
        self.repository = repository
        self.check_interval = check_interval
        self.positions: dict[str, Position] = {}
        self._running = False
        self._task: asyncio.Task | None = None

        # P&L tracking
        self.total_pnl = Decimal(0)
        self.realized_pnl = Decimal(0)
        self.unrealized_pnl = Decimal(0)
        self.daily_pnl = Decimal(0)

        # Trade statistics
        self.total_trades = 0
        self.winning_trades = 0
        self.losing_trades = 0

    async def start(self) -> None:
        """Start monitoring positions."""
        if self._running:
            logger.warning("Position monitor already running")
            return

        self._running = True
        self._task = asyncio.create_task(self._monitor_loop())
        logger.info("Position monitor started", execution_id=str(self.execution_id))

    async def stop(self) -> None:
        """Stop monitoring positions."""
        if not self._running:
            return

        self._running = False

        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass

        logger.info("Position monitor stopped", execution_id=str(self.execution_id))

    async def add_position(
        self,
        symbol: str,
        quantity: int,
        entry_price: Decimal,
        side: str = "LONG",
    ) -> None:
        """Add or update a position.

        Args:
            symbol: Trading symbol
            quantity: Position quantity
            entry_price: Entry price
            side: Position side (LONG/SHORT)
        """
        if symbol in self.positions:
            # Update existing position (average price)
            existing = self.positions[symbol]
            total_quantity = existing.quantity + quantity
            if total_quantity == 0:
                # Position closed
                del self.positions[symbol]
                logger.info("Position closed", symbol=symbol)
                return

            avg_price = (
                (existing.entry_price * existing.quantity) + (entry_price * quantity)
            ) / total_quantity

            self.positions[symbol] = Position(
                symbol=symbol,
                quantity=total_quantity,
                entry_price=avg_price,
                current_price=entry_price,  # Will be updated by price feed
                side=side,
            )
            logger.info(
                "Position updated",
                symbol=symbol,
                quantity=total_quantity,
                avg_price=float(avg_price),
            )
        else:
            # New position
            self.positions[symbol] = Position(
                symbol=symbol,
                quantity=quantity,
                entry_price=entry_price,
                current_price=entry_price,
                side=side,
            )
            logger.info(
                "Position added",
                symbol=symbol,
                quantity=quantity,
                entry_price=float(entry_price),
            )

        await self._update_positions_in_db()

    async def close_position(self, symbol: str, exit_price: Decimal) -> Decimal:
        """Close a position and record realized P&L.

        Args:
            symbol: Trading symbol
            exit_price: Exit price

        Returns:
            Realized P&L
        """
        if symbol not in self.positions:
            logger.warning("Attempted to close non-existent position", symbol=symbol)
            return Decimal(0)

        position = self.positions[symbol]

        # Calculate realized P&L
        if position.side == "LONG":
            pnl = (exit_price - position.entry_price) * position.quantity
        else:  # SHORT
            pnl = (position.entry_price - exit_price) * position.quantity

        self.realized_pnl += pnl
        self.total_pnl += pnl
        self.daily_pnl += pnl

        # Update trade statistics
        self.total_trades += 1
        if pnl > 0:
            self.winning_trades += 1
        elif pnl < 0:
            self.losing_trades += 1

        logger.info(
            "Position closed",
            symbol=symbol,
            quantity=position.quantity,
            entry_price=float(position.entry_price),
            exit_price=float(exit_price),
            pnl=float(pnl),
        )

        # Remove position
        del self.positions[symbol]

        # Update database
        await self._update_pnl_in_db()
        await self._update_positions_in_db()

        return pnl

    async def update_prices(self, prices: dict[str, Decimal]) -> None:
        """Update current prices for positions.

        Args:
            prices: Map of symbol to current price
        """
        for symbol, price in prices.items():
            if symbol in self.positions:
                self.positions[symbol].current_price = price

        # Recalculate unrealized P&L
        self.unrealized_pnl = sum(pos.unrealized_pnl for pos in self.positions.values())
        self.total_pnl = self.realized_pnl + self.unrealized_pnl

        await self._update_pnl_in_db()

    async def get_portfolio_value(self) -> Decimal:
        """Get total portfolio value.

        Returns:
            Total portfolio value
        """
        return sum(pos.market_value for pos in self.positions.values())

    async def get_total_exposure(self) -> Decimal:
        """Get total exposure.

        Returns:
            Total exposure
        """
        return sum(pos.market_value for pos in self.positions.values())

    async def _monitor_loop(self) -> None:
        """Position monitoring loop."""
        logger.info("Position monitor loop started")

        try:
            while self._running:
                await self._check_positions()
                await asyncio.sleep(self.check_interval)

        except asyncio.CancelledError:
            logger.info("Position monitor loop cancelled")
        except Exception as e:
            logger.error("Error in position monitor loop", error=str(e))
            raise

    async def _check_positions(self) -> None:
        """Check positions and update metrics."""
        try:
            # Create position snapshot
            snapshot_id = uuid4()
            positions_dict = {
                symbol: pos.to_dict() for symbol, pos in self.positions.items()
            }
            total_value = await self.get_portfolio_value()

            await self.repository.create_position_snapshot(
                snapshot_id=snapshot_id,
                execution_id=self.execution_id,
                positions=positions_dict,
                total_value=total_value,
                total_pnl=self.total_pnl,
                unrealized_pnl=self.unrealized_pnl,
            )

            logger.debug(
                "Position snapshot created",
                execution_id=str(self.execution_id),
                positions=len(self.positions),
                total_value=float(total_value),
            )

        except Exception as e:
            logger.error("Error checking positions", error=str(e))

    async def _update_positions_in_db(self) -> None:
        """Update positions in database."""
        positions_dict = {symbol: pos.to_dict() for symbol, pos in self.positions.items()}
        total_exposure = await self.get_total_exposure()

        await self.repository.update_positions(
            execution_id=self.execution_id,
            positions=positions_dict,
            total_exposure=total_exposure,
        )

    async def _update_pnl_in_db(self) -> None:
        """Update P&L metrics in database."""
        await self.repository.update_pnl(
            execution_id=self.execution_id,
            total_pnl=self.total_pnl,
            realized_pnl=self.realized_pnl,
            unrealized_pnl=self.unrealized_pnl,
            daily_pnl=self.daily_pnl,
        )

        await self.repository.update_trade_stats(
            execution_id=self.execution_id,
            total_trades=self.total_trades,
            winning_trades=self.winning_trades,
            losing_trades=self.losing_trades,
        )
