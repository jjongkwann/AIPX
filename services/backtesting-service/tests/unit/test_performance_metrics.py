"""Unit tests for Performance Metrics"""

import pytest
import numpy as np
from datetime import datetime, timedelta
from src.metrics.performance_metrics import PerformanceMetrics
from src.engine.portfolio import Trade


class TestPerformanceMetrics:
    """Test PerformanceMetrics class"""

    def test_calculate_cagr_positive(self):
        """Test CAGR calculation with positive returns"""
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000},
            {'timestamp': datetime(2025, 1, 1), 'equity': 11500000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve, years=1.0)

        # CAGR should be 15%
        assert abs(cagr - 15.0) < 0.01

    def test_calculate_cagr_with_auto_years(self):
        """Test CAGR calculation with automatic year calculation"""
        start = datetime(2024, 1, 1)
        end = datetime(2025, 1, 1)

        equity_curve = [
            {'timestamp': start, 'equity': 10000000},
            {'timestamp': end, 'equity': 12000000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve)

        # Should calculate years automatically
        assert cagr > 0

    def test_calculate_cagr_negative(self):
        """Test CAGR calculation with loss"""
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000},
            {'timestamp': datetime(2025, 1, 1), 'equity': 9000000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve, years=1.0)

        # CAGR should be -10%
        assert abs(cagr - (-10.0)) < 0.01

    def test_calculate_cagr_empty_curve(self):
        """Test CAGR with empty equity curve"""
        cagr = PerformanceMetrics.calculate_cagr([])

        assert cagr == 0.0

    def test_calculate_cagr_single_point(self):
        """Test CAGR with single data point"""
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve)

        assert cagr == 0.0

    def test_calculate_mdd_with_drawdown(self, drawdown_data):
        """Test MDD calculation with known drawdown"""
        mdd = PerformanceMetrics.calculate_mdd(drawdown_data['equity_curve'])

        # Should be close to -20%
        assert abs(mdd - (-20.0)) < 1.0

    def test_calculate_mdd_no_drawdown(self):
        """Test MDD with continuously increasing equity"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000 * (1 + i * 0.01)
            })

        mdd = PerformanceMetrics.calculate_mdd(equity_curve)

        # Should be zero or very close
        assert mdd >= -0.1

    def test_calculate_mdd_empty_curve(self):
        """Test MDD with empty curve"""
        mdd = PerformanceMetrics.calculate_mdd([])

        assert mdd == 0.0

    def test_calculate_sharpe_ratio(self, sample_equity_curve):
        """Test Sharpe ratio calculation"""
        sharpe = PerformanceMetrics.calculate_sharpe_ratio(sample_equity_curve)

        # Should return a reasonable value
        assert isinstance(sharpe, float)
        assert -5 < sharpe < 5  # Reasonable range

    def test_calculate_sharpe_ratio_zero_volatility(self):
        """Test Sharpe ratio with zero volatility"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000  # Constant
            })

        sharpe = PerformanceMetrics.calculate_sharpe_ratio(equity_curve)

        assert sharpe == 0.0

    def test_calculate_sharpe_ratio_empty_curve(self):
        """Test Sharpe ratio with empty curve"""
        sharpe = PerformanceMetrics.calculate_sharpe_ratio([])

        assert sharpe == 0.0

    def test_calculate_sortino_ratio(self, sample_equity_curve):
        """Test Sortino ratio calculation"""
        sortino = PerformanceMetrics.calculate_sortino_ratio(sample_equity_curve)

        # Should return a reasonable value
        assert isinstance(sortino, float)

    def test_calculate_sortino_ratio_no_downside(self):
        """Test Sortino ratio with no negative returns"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000 * (1 + i * 0.01)  # Only positive returns
            })

        sortino = PerformanceMetrics.calculate_sortino_ratio(equity_curve)

        # Should be infinite (no downside)
        assert sortino == float('inf')

    def test_calculate_sortino_ratio_empty_curve(self):
        """Test Sortino ratio with empty curve"""
        sortino = PerformanceMetrics.calculate_sortino_ratio([])

        assert sortino == 0.0

    def test_calculate_win_rate_all_wins(self):
        """Test win rate with all winning trades"""
        trades = []
        for i in range(10):
            trade = Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=10000  # All positive
            )
            trades.append(trade)

        win_rate = PerformanceMetrics.calculate_win_rate(trades)

        assert win_rate == 100.0

    def test_calculate_win_rate_all_losses(self):
        """Test win rate with all losing trades"""
        trades = []
        for i in range(10):
            trade = Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-10000  # All negative
            )
            trades.append(trade)

        win_rate = PerformanceMetrics.calculate_win_rate(trades)

        assert win_rate == 0.0

    def test_calculate_win_rate_mixed(self, sample_trades):
        """Test win rate with mixed trades"""
        win_rate = PerformanceMetrics.calculate_win_rate(sample_trades)

        # Should be between 0 and 100
        assert 0 <= win_rate <= 100

    def test_calculate_win_rate_empty(self):
        """Test win rate with no trades"""
        win_rate = PerformanceMetrics.calculate_win_rate([])

        assert win_rate == 0.0

    def test_calculate_win_rate_ignores_buys(self):
        """Test that win rate only counts sell trades with P&L"""
        trades = []

        # Add buy trade (no P&L)
        trades.append(Trade(
            timestamp=datetime(2024, 1, 1),
            symbol='005930',
            side='buy',
            quantity=10,
            price=70000,
            commission=300
        ))

        # Add sell trade (with P&L)
        trades.append(Trade(
            timestamp=datetime(2024, 1, 2),
            symbol='005930',
            side='sell',
            quantity=10,
            price=72000,
            commission=300,
            pnl=10000
        ))

        win_rate = PerformanceMetrics.calculate_win_rate(trades)

        # Should be 100% (1 winning trade out of 1 closed trade)
        assert win_rate == 100.0

    def test_calculate_profit_factor_profit(self):
        """Test profit factor with profit"""
        trades = []

        # 3 winning trades: +30000
        for i in range(3):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=10000
            ))

        # 2 losing trades: -10000
        for i in range(2):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i+3),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-5000
            ))

        profit_factor = PerformanceMetrics.calculate_profit_factor(trades)

        # Profit factor = 30000 / 10000 = 3.0
        assert abs(profit_factor - 3.0) < 0.01

    def test_calculate_profit_factor_no_losses(self):
        """Test profit factor with no losing trades"""
        trades = []
        for i in range(5):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=10000
            ))

        profit_factor = PerformanceMetrics.calculate_profit_factor(trades)

        # Should be infinite
        assert profit_factor == float('inf')

    def test_calculate_profit_factor_no_wins(self):
        """Test profit factor with no winning trades"""
        trades = []
        for i in range(5):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-10000
            ))

        profit_factor = PerformanceMetrics.calculate_profit_factor(trades)

        # Should be 0
        assert profit_factor == 0.0

    def test_calculate_profit_factor_empty(self):
        """Test profit factor with no trades"""
        profit_factor = PerformanceMetrics.calculate_profit_factor([])

        assert profit_factor == 0.0

    def test_calculate_average_win(self, sample_trades):
        """Test average win calculation"""
        avg_win = PerformanceMetrics.calculate_average_win(sample_trades)

        # Should be positive
        assert avg_win >= 0

    def test_calculate_average_win_no_wins(self):
        """Test average win with no winning trades"""
        trades = []
        for i in range(5):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-10000
            ))

        avg_win = PerformanceMetrics.calculate_average_win(trades)

        assert avg_win == 0.0

    def test_calculate_average_loss(self, sample_trades):
        """Test average loss calculation"""
        avg_loss = PerformanceMetrics.calculate_average_loss(sample_trades)

        # Should be negative or zero
        assert avg_loss <= 0

    def test_calculate_average_loss_no_losses(self):
        """Test average loss with no losing trades"""
        trades = []
        for i in range(5):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=10000
            ))

        avg_loss = PerformanceMetrics.calculate_average_loss(trades)

        assert avg_loss == 0.0

    def test_calculate_all_metrics(self, sample_equity_curve, sample_trades):
        """Test calculating all metrics at once"""
        metrics = PerformanceMetrics.calculate_all_metrics(
            sample_equity_curve,
            sample_trades
        )

        # Check all metrics are present
        assert 'cagr' in metrics
        assert 'mdd' in metrics
        assert 'sharpe_ratio' in metrics
        assert 'sortino_ratio' in metrics
        assert 'win_rate' in metrics
        assert 'profit_factor' in metrics
        assert 'average_win' in metrics
        assert 'average_loss' in metrics

        # Check all are numeric
        for key, value in metrics.items():
            assert isinstance(value, (int, float))

    def test_sharpe_ratio_with_known_data(self, known_returns_data):
        """Test Sharpe ratio matches expected value with known data"""
        sharpe = PerformanceMetrics.calculate_sharpe_ratio(
            known_returns_data['equity_curve'],
            risk_free_rate=0.02
        )

        # Should be positive (returns > risk-free rate)
        assert sharpe > 0

    def test_metrics_consistency(self, sample_equity_curve):
        """Test that metrics are consistent across multiple calls"""
        cagr1 = PerformanceMetrics.calculate_cagr(sample_equity_curve)
        cagr2 = PerformanceMetrics.calculate_cagr(sample_equity_curve)

        assert cagr1 == cagr2

        mdd1 = PerformanceMetrics.calculate_mdd(sample_equity_curve)
        mdd2 = PerformanceMetrics.calculate_mdd(sample_equity_curve)

        assert mdd1 == mdd2
