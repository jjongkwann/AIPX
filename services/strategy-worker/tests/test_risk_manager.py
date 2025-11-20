"""Tests for Risk Manager."""

from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from src.risk import RiskManager, RiskCheckResult


@pytest.mark.unit
class TestRiskManager:
    """Test RiskManager class."""

    @pytest.fixture
    def mock_repository(self):
        """Create mock repository."""
        repo = MagicMock()
        repo.get_daily_pnl = AsyncMock(return_value=Decimal("0"))
        repo.create_risk_event = AsyncMock()
        return repo

    @pytest.fixture
    def risk_manager(self, test_config, mock_repository):
        """Create risk manager instance."""
        return RiskManager(test_config.risk, mock_repository)

    @pytest.mark.asyncio
    async def test_order_below_min_size(self, risk_manager, execution_id):
        """Test order rejection for below minimum size."""
        result = await risk_manager.check_order(
            execution_id=execution_id,
            symbol="AAPL",
            side="BUY",
            quantity=1,
            price=Decimal("50.00"),  # $50 order
            current_positions={},
            portfolio_value=Decimal("10000.00"),
        )

        assert not result.passed
        assert len(result.violations) > 0
        assert result.violations[0].violation_type == "MIN_ORDER_SIZE"

    @pytest.mark.asyncio
    async def test_order_above_max_size(self, risk_manager, execution_id):
        """Test order rejection for above maximum size."""
        result = await risk_manager.check_order(
            execution_id=execution_id,
            symbol="AAPL",
            side="BUY",
            quantity=1000,
            price=Decimal("200.00"),  # $200k order
            current_positions={},
            portfolio_value=Decimal("10000.00"),
        )

        assert not result.passed
        assert any(v.violation_type == "MAX_ORDER_SIZE" for v in result.violations)

    @pytest.mark.asyncio
    async def test_position_size_limit(self, risk_manager, execution_id):
        """Test position size limit violation."""
        result = await risk_manager.check_order(
            execution_id=execution_id,
            symbol="AAPL",
            side="BUY",
            quantity=40,
            price=Decimal("100.00"),  # $4000 order
            current_positions={},
            portfolio_value=Decimal("10000.00"),  # 40% of portfolio
        )

        assert not result.passed
        assert any(v.violation_type == "MAX_POSITION_SIZE" for v in result.violations)

    @pytest.mark.asyncio
    async def test_valid_order(self, risk_manager, execution_id):
        """Test valid order passes all checks."""
        result = await risk_manager.check_order(
            execution_id=execution_id,
            symbol="AAPL",
            side="BUY",
            quantity=10,
            price=Decimal("150.00"),  # $1500 order (15% of portfolio)
            current_positions={},
            portfolio_value=Decimal("10000.00"),
        )

        assert result.passed
        assert len(result.violations) == 0

    @pytest.mark.asyncio
    async def test_daily_loss_limit(self, risk_manager, execution_id, mock_repository):
        """Test daily loss limit violation."""
        # Set daily loss to -$600 (6% of $10k portfolio)
        mock_repository.get_daily_pnl = AsyncMock(return_value=Decimal("-600.00"))

        result = await risk_manager.check_order(
            execution_id=execution_id,
            symbol="AAPL",
            side="BUY",
            quantity=10,
            price=Decimal("150.00"),
            current_positions={},
            portfolio_value=Decimal("10000.00"),
        )

        assert not result.passed
        assert any(v.violation_type == "MAX_DAILY_LOSS_PCT" for v in result.violations)

    @pytest.mark.asyncio
    async def test_stop_loss_triggered(self, risk_manager, execution_id):
        """Test stop loss trigger detection."""
        triggered = await risk_manager.check_stop_loss(
            execution_id=execution_id,
            symbol="AAPL",
            entry_price=Decimal("100.00"),
            current_price=Decimal("92.00"),  # -8% loss
            stop_loss_pct=Decimal("0.05"),  # 5% stop loss
        )

        assert triggered

    @pytest.mark.asyncio
    async def test_stop_loss_not_triggered(self, risk_manager, execution_id):
        """Test stop loss not triggered."""
        triggered = await risk_manager.check_stop_loss(
            execution_id=execution_id,
            symbol="AAPL",
            entry_price=Decimal("100.00"),
            current_price=Decimal("97.00"),  # -3% loss
            stop_loss_pct=Decimal("0.05"),  # 5% stop loss
        )

        assert not triggered
