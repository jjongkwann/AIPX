"""Unit tests for Portfolio Manager"""

from datetime import datetime

import pytest
from src.engine.portfolio import PortfolioManager


class TestPortfolioManager:
    """Test PortfolioManager class"""

    def test_initialization(self):
        """Test portfolio initialization"""
        pm = PortfolioManager(initial_cash=10000000)

        assert pm.initial_cash == 10000000
        assert pm.cash == 10000000
        assert pm.commission_rate == 0.0003
        assert pm.tax_rate == 0.0023
        assert len(pm.positions) == 0
        assert len(pm.trade_history) == 0
        assert pm.realized_pnl == 0.0
        assert pm.total_commission == 0.0

    def test_initialization_invalid_cash(self):
        """Test that zero or negative cash raises error"""
        with pytest.raises(ValueError):
            PortfolioManager(initial_cash=0)

        with pytest.raises(ValueError):
            PortfolioManager(initial_cash=-1000)

    def test_execute_buy_success(self):
        """Test successful buy execution"""
        pm = PortfolioManager(initial_cash=10000000)

        success = pm.execute_buy(symbol="005930", quantity=10, price=71000, timestamp=datetime(2024, 1, 1, 9, 0))

        assert success is True
        assert pm.cash < 10000000  # Cash decreased
        assert "005930" in pm.positions
        assert pm.positions["005930"].quantity == 10
        assert pm.positions["005930"].avg_price == 71000
        assert len(pm.trade_history) == 1
        assert pm.total_commission > 0

    def test_execute_buy_insufficient_funds(self):
        """Test buy with insufficient funds"""
        pm = PortfolioManager(initial_cash=100000)  # Not enough

        success = pm.execute_buy(
            symbol="005930",
            quantity=100,  # Would cost 7,100,000
            price=71000,
            timestamp=datetime(2024, 1, 1, 9, 0),
        )

        assert success is False
        assert pm.cash == 100000  # No change
        assert "005930" not in pm.positions
        assert len(pm.trade_history) == 0

    def test_execute_buy_invalid_quantity(self):
        """Test buy with invalid quantity"""
        pm = PortfolioManager(initial_cash=10000000)

        with pytest.raises(ValueError):
            pm.execute_buy("005930", 0, 71000, datetime(2024, 1, 1))

        with pytest.raises(ValueError):
            pm.execute_buy("005930", -10, 71000, datetime(2024, 1, 1))

    def test_execute_buy_invalid_price(self):
        """Test buy with invalid price"""
        pm = PortfolioManager(initial_cash=10000000)

        with pytest.raises(ValueError):
            pm.execute_buy("005930", 10, 0, datetime(2024, 1, 1))

        with pytest.raises(ValueError):
            pm.execute_buy("005930", 10, -71000, datetime(2024, 1, 1))

    def test_execute_buy_multiple_times_same_symbol(self):
        """Test buying same symbol multiple times (averaging)"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))
        pm.execute_buy("005930", 10, 72000, datetime(2024, 1, 2))

        position = pm.positions["005930"]
        assert position.quantity == 20
        assert position.avg_price == 71000  # Average of 70000 and 72000

    def test_execute_sell_success(self):
        """Test successful sell execution"""
        pm = PortfolioManager(initial_cash=10000000)

        # First buy
        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))
        cash_after_buy = pm.cash

        # Then sell
        success = pm.execute_sell("005930", 10, 72000, datetime(2024, 1, 2))

        assert success is True
        assert pm.cash > cash_after_buy  # Cash increased
        assert "005930" not in pm.positions  # Position closed
        assert len(pm.trade_history) == 2

        # Check P&L was recorded
        sell_trade = pm.trade_history[-1]
        assert sell_trade.pnl is not None

    def test_execute_sell_no_position(self):
        """Test selling without position"""
        pm = PortfolioManager(initial_cash=10000000)

        success = pm.execute_sell("005930", 10, 71000, datetime(2024, 1, 1))

        assert success is False
        assert len(pm.trade_history) == 0

    def test_execute_sell_insufficient_quantity(self):
        """Test selling more than owned"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 71000, datetime(2024, 1, 1))

        success = pm.execute_sell("005930", 20, 71000, datetime(2024, 1, 2))

        assert success is False
        assert pm.positions["005930"].quantity == 10  # Unchanged

    def test_execute_sell_invalid_quantity(self):
        """Test sell with invalid quantity"""
        pm = PortfolioManager(initial_cash=10000000)
        pm.execute_buy("005930", 10, 71000, datetime(2024, 1, 1))

        with pytest.raises(ValueError):
            pm.execute_sell("005930", 0, 71000, datetime(2024, 1, 2))

        with pytest.raises(ValueError):
            pm.execute_sell("005930", -10, 71000, datetime(2024, 1, 2))

    def test_execute_sell_invalid_price(self):
        """Test sell with invalid price"""
        pm = PortfolioManager(initial_cash=10000000)
        pm.execute_buy("005930", 10, 71000, datetime(2024, 1, 1))

        with pytest.raises(ValueError):
            pm.execute_sell("005930", 10, 0, datetime(2024, 1, 2))

        with pytest.raises(ValueError):
            pm.execute_sell("005930", 10, -71000, datetime(2024, 1, 2))

    def test_execute_sell_partial(self):
        """Test partial sell"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 20, 71000, datetime(2024, 1, 1))
        pm.execute_sell("005930", 10, 72000, datetime(2024, 1, 2))

        assert "005930" in pm.positions
        assert pm.positions["005930"].quantity == 10
        assert pm.positions["005930"].avg_price == 71000  # Unchanged

    def test_realized_pnl_calculation(self):
        """Test realized P&L calculation"""
        pm = PortfolioManager(initial_cash=10000000)

        # Buy at 70000
        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        # Sell at 72000 (profit)
        pm.execute_sell("005930", 10, 72000, datetime(2024, 1, 2))

        # Calculate expected P&L
        cost = 10 * 70000
        proceeds = 10 * 72000
        commission_buy = cost * 0.0003
        commission_sell = proceeds * 0.0003
        tax = proceeds * 0.0023
        expected_pnl = (proceeds - commission_sell - tax) - cost

        assert abs(pm.realized_pnl - expected_pnl) < 0.01

    def test_get_equity_with_positions(self):
        """Test equity calculation with positions"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        current_prices = {"005930": 72000}
        equity = pm.get_equity(current_prices)

        expected_equity = pm.cash + (10 * 72000)
        assert abs(equity - expected_equity) < 0.01

    def test_get_equity_no_positions(self):
        """Test equity equals cash when no positions"""
        pm = PortfolioManager(initial_cash=10000000)

        equity = pm.get_equity({})

        assert equity == 10000000

    def test_get_equity_multiple_positions(self):
        """Test equity with multiple positions"""
        pm = PortfolioManager(initial_cash=20000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))
        pm.execute_buy("000660", 5, 100000, datetime(2024, 1, 1))

        current_prices = {"005930": 72000, "000660": 105000}

        equity = pm.get_equity(current_prices)

        expected_equity = pm.cash + (10 * 72000) + (5 * 105000)
        assert abs(equity - expected_equity) < 0.01

    def test_get_unrealized_pnl(self):
        """Test unrealized P&L calculation"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        current_prices = {"005930": 72000}
        unrealized_pnl = pm.get_unrealized_pnl(current_prices)

        expected_pnl = (72000 - 70000) * 10
        assert abs(unrealized_pnl - expected_pnl) < 0.01

    def test_get_unrealized_pnl_loss(self):
        """Test unrealized P&L with loss"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        current_prices = {"005930": 68000}
        unrealized_pnl = pm.get_unrealized_pnl(current_prices)

        expected_pnl = (68000 - 70000) * 10
        assert abs(unrealized_pnl - expected_pnl) < 0.01
        assert unrealized_pnl < 0

    def test_get_total_pnl(self):
        """Test total P&L (realized + unrealized)"""
        pm = PortfolioManager(initial_cash=10000000)

        # Buy and sell some
        pm.execute_buy("005930", 20, 70000, datetime(2024, 1, 1))
        pm.execute_sell("005930", 10, 72000, datetime(2024, 1, 2))

        # Calculate with current price
        current_prices = {"005930": 73000}
        total_pnl = pm.get_total_pnl(current_prices)

        # Should be realized (from 10 sold) + unrealized (from 10 held)
        assert total_pnl > pm.realized_pnl

    def test_update_equity_curve(self):
        """Test equity curve recording"""
        pm = PortfolioManager(initial_cash=10000000)

        timestamp = datetime(2024, 1, 1, 9, 0)
        current_prices = {}

        pm.update_equity_curve(timestamp, current_prices)

        assert len(pm.equity_curve) == 1
        assert pm.equity_curve[0]["timestamp"] == timestamp
        assert pm.equity_curve[0]["equity"] == 10000000
        assert pm.equity_curve[0]["cash"] == 10000000
        assert pm.equity_curve[0]["position_value"] == 0

    def test_equity_curve_with_positions(self):
        """Test equity curve with positions"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        timestamp = datetime(2024, 1, 2)
        current_prices = {"005930": 72000}

        pm.update_equity_curve(timestamp, current_prices)

        assert len(pm.equity_curve) == 1
        point = pm.equity_curve[0]
        assert point["equity"] == pm.cash + (10 * 72000)
        assert point["position_value"] == 10 * 72000

    def test_get_position(self):
        """Test getting position for symbol"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        position = pm.get_position("005930")

        assert position is not None
        assert position.symbol == "005930"
        assert position.quantity == 10
        assert position.avg_price == 70000

    def test_get_position_not_exist(self):
        """Test getting non-existent position returns None"""
        pm = PortfolioManager(initial_cash=10000000)

        position = pm.get_position("005930")

        assert position is None

    def test_get_all_positions(self):
        """Test getting all positions"""
        pm = PortfolioManager(initial_cash=20000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))
        pm.execute_buy("000660", 5, 100000, datetime(2024, 1, 1))

        positions = pm.get_all_positions()

        assert len(positions) == 2
        symbols = [p.symbol for p in positions]
        assert "005930" in symbols
        assert "000660" in symbols

    def test_commission_accumulation(self):
        """Test that commission accumulates correctly"""
        pm = PortfolioManager(initial_cash=10000000)

        # Buy
        cost1 = 10 * 70000
        commission1 = cost1 * 0.0003
        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1))

        # Sell
        proceeds = 10 * 72000
        commission2 = proceeds * 0.0003
        pm.execute_sell("005930", 10, 72000, datetime(2024, 1, 2))

        expected_commission = commission1 + commission2
        assert abs(pm.total_commission - expected_commission) < 0.01

    def test_trade_history_recording(self):
        """Test that all trades are recorded"""
        pm = PortfolioManager(initial_cash=10000000)

        pm.execute_buy("005930", 10, 70000, datetime(2024, 1, 1, 9, 0))
        pm.execute_buy("000660", 5, 100000, datetime(2024, 1, 1, 10, 0))
        pm.execute_sell("005930", 5, 72000, datetime(2024, 1, 2, 9, 0))

        assert len(pm.trade_history) == 3

        # Check trade details
        assert pm.trade_history[0].symbol == "005930"
        assert pm.trade_history[0].side == "buy"
        assert pm.trade_history[1].symbol == "000660"
        assert pm.trade_history[1].side == "buy"
        assert pm.trade_history[2].symbol == "005930"
        assert pm.trade_history[2].side == "sell"
        assert pm.trade_history[2].pnl is not None  # Sell has P&L
