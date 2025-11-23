"""Tests for Position Monitor."""

from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock

import pytest

from src.monitor import Position, PositionMonitor


@pytest.mark.unit
class TestPositionMonitor:
    """Test PositionMonitor class."""

    @pytest.fixture
    def mock_repository(self):
        """Create mock repository."""
        repo = MagicMock()
        repo.update_positions = AsyncMock()
        repo.update_pnl = AsyncMock()
        repo.update_trade_stats = AsyncMock()
        repo.create_position_snapshot = AsyncMock()
        return repo

    @pytest.fixture
    def position_monitor(self, execution_id, mock_repository):
        """Create position monitor instance."""
        return PositionMonitor(
            execution_id=execution_id,
            repository=mock_repository,
            check_interval=10,
        )

    @pytest.mark.asyncio
    async def test_add_position(self, position_monitor):
        """Test adding a new position."""
        await position_monitor.add_position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
            side="LONG",
        )

        assert "AAPL" in position_monitor.positions
        position = position_monitor.positions["AAPL"]
        assert position.quantity == 100
        assert position.entry_price == Decimal("150.00")

    @pytest.mark.asyncio
    async def test_close_position(self, position_monitor):
        """Test closing a position."""
        # Add position
        await position_monitor.add_position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
        )

        # Close with profit
        pnl = await position_monitor.close_position(
            symbol="AAPL",
            exit_price=Decimal("160.00"),
        )

        assert pnl == Decimal("1000.00")  # $10 * 100 shares
        assert "AAPL" not in position_monitor.positions
        assert position_monitor.realized_pnl == Decimal("1000.00")
        assert position_monitor.total_trades == 1
        assert position_monitor.winning_trades == 1

    @pytest.mark.asyncio
    async def test_update_prices(self, position_monitor):
        """Test updating position prices."""
        await position_monitor.add_position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
        )

        await position_monitor.update_prices(
            {
                "AAPL": Decimal("155.00"),
            }
        )

        position = position_monitor.positions["AAPL"]
        assert position.current_price == Decimal("155.00")
        assert position_monitor.unrealized_pnl == Decimal("500.00")  # $5 * 100

    @pytest.mark.asyncio
    async def test_portfolio_value(self, position_monitor):
        """Test portfolio value calculation."""
        await position_monitor.add_position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
        )
        await position_monitor.add_position(
            symbol="MSFT",
            quantity=50,
            entry_price=Decimal("300.00"),
        )

        # Update prices
        await position_monitor.update_prices(
            {
                "AAPL": Decimal("155.00"),
                "MSFT": Decimal("310.00"),
            }
        )

        portfolio_value = await position_monitor.get_portfolio_value()
        expected = (Decimal("155.00") * 100) + (Decimal("310.00") * 50)
        assert portfolio_value == expected


@pytest.mark.unit
class TestPosition:
    """Test Position class."""

    def test_long_position_profit(self):
        """Test long position with profit."""
        position = Position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
            current_price=Decimal("160.00"),
            side="LONG",
        )

        assert position.unrealized_pnl == Decimal("1000.00")
        assert position.unrealized_pnl_pct == Decimal("6.666666666666666666666666667")

    def test_long_position_loss(self):
        """Test long position with loss."""
        position = Position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
            current_price=Decimal("140.00"),
            side="LONG",
        )

        assert position.unrealized_pnl == Decimal("-1000.00")

    def test_position_to_dict(self):
        """Test position serialization."""
        position = Position(
            symbol="AAPL",
            quantity=100,
            entry_price=Decimal("150.00"),
            current_price=Decimal("155.00"),
            side="LONG",
        )

        data = position.to_dict()
        assert data["symbol"] == "AAPL"
        assert data["quantity"] == 100
        assert data["unrealized_pnl"] == 500.0
