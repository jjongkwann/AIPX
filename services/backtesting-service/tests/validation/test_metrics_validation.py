"""Validation tests comparing metrics against reference implementations"""

import pytest
import numpy as np
import pandas as pd
from datetime import datetime, timedelta

from src.metrics.performance_metrics import PerformanceMetrics
from src.metrics.risk_metrics import RiskMetrics


@pytest.mark.validation
class TestMetricsValidation:
    """Validate metrics against known values and reference implementations"""

    def test_cagr_validation_known_values(self):
        """Validate CAGR against manually calculated known values"""
        # Simple case: 10M -> 12M in 1 year = 20% CAGR
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000},
            {'timestamp': datetime(2025, 1, 1), 'equity': 12000000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve, years=1.0)

        # Expected: ((12M / 10M) ^ (1/1) - 1) * 100 = 20%
        expected = 20.0
        assert abs(cagr - expected) < 0.01

    def test_cagr_validation_multi_year(self):
        """Validate CAGR over multiple years"""
        # 10M -> 14.641M in 2 years = ~20% CAGR (1.2^2 = 1.44)
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000},
            {'timestamp': datetime(2026, 1, 1), 'equity': 14400000}
        ]

        cagr = PerformanceMetrics.calculate_cagr(equity_curve, years=2.0)

        # Expected: ((14.4M / 10M) ^ (1/2) - 1) * 100 = 20%
        expected = 20.0
        assert abs(cagr - expected) < 0.1

    def test_mdd_validation_known_drawdown(self):
        """Validate MDD with known drawdown scenario"""
        # Create equity curve with known 20% drawdown
        equity_values = [
            10000000,  # Start
            11000000,  # +10%
            12000000,  # +20% (peak)
            11400000,  # -5%
            10800000,  # -10%
            9600000,   # -20% from peak (trough)
            10200000,  # Recovery
            11000000,
        ]

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        mdd = PerformanceMetrics.calculate_mdd(equity_curve)

        # Expected: -20%
        expected = -20.0
        assert abs(mdd - expected) < 0.1

    def test_sharpe_ratio_validation_manual_calculation(self):
        """Validate Sharpe ratio against manual calculation"""
        # Create equity curve with known returns
        np.random.seed(42)
        returns = np.array([0.01, 0.02, -0.01, 0.015, 0.005, -0.005, 0.01, 0.02])

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        sharpe = PerformanceMetrics.calculate_sharpe_ratio(
            equity_curve,
            risk_free_rate=0.0,  # No risk-free rate for simplicity
            periods_per_year=252
        )

        # Manual calculation
        mean_return = np.mean(returns)
        std_return = np.std(returns, ddof=1)
        expected_sharpe = (mean_return * 252) / (std_return * np.sqrt(252))

        assert abs(sharpe - expected_sharpe) < 0.01

    def test_sortino_ratio_validation(self):
        """Validate Sortino ratio calculation"""
        # Create equity curve with known asymmetric returns
        returns = np.array([0.02, 0.03, 0.01, -0.02, 0.02, -0.01, 0.025, 0.015])

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        sortino = PerformanceMetrics.calculate_sortino_ratio(
            equity_curve,
            risk_free_rate=0.0,
            periods_per_year=252
        )

        # Manual calculation
        mean_return = np.mean(returns)
        negative_returns = returns[returns < 0]
        downside_std = np.std(negative_returns, ddof=1)
        expected_sortino = (mean_return * 252) / (downside_std * np.sqrt(252))

        assert abs(sortino - expected_sortino) < 0.01

    def test_win_rate_validation(self):
        """Validate win rate with known trades"""
        from src.engine.portfolio import Trade

        # 7 winning, 3 losing = 70% win rate
        trades = []

        # 7 wins
        for i in range(7):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i*2+1),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=10000  # Positive
            ))

        # 3 losses
        for i in range(3):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i*2+15),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-5000  # Negative
            ))

        win_rate = PerformanceMetrics.calculate_win_rate(trades)

        # Expected: 70%
        expected = 70.0
        assert abs(win_rate - expected) < 0.01

    def test_profit_factor_validation(self):
        """Validate profit factor with known trades"""
        from src.engine.portfolio import Trade

        trades = []

        # Total profit: 100,000
        for i in range(5):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=20000
            ))

        # Total loss: 40,000
        for i in range(4):
            trades.append(Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(days=i+5),
                symbol='005930',
                side='sell',
                quantity=10,
                price=71000,
                commission=300,
                pnl=-10000
            ))

        profit_factor = PerformanceMetrics.calculate_profit_factor(trades)

        # Expected: 100,000 / 40,000 = 2.5
        expected = 2.5
        assert abs(profit_factor - expected) < 0.01

    def test_var_validation_normal_distribution(self):
        """Validate VaR with normally distributed returns"""
        np.random.seed(42)

        # Generate returns from normal distribution
        returns = np.random.normal(0.001, 0.02, 1000)  # Mean 0.1%, std 2%

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(hours=i),
                'equity': equity
            })

        var_95 = RiskMetrics.calculate_var(equity_curve, 0.95)

        # For normal distribution, 5th percentile is approximately -1.645 * std
        expected_var = (np.percentile(returns, 5)) * 100  # As percentage

        assert abs(var_95 - expected_var) < 0.5

    def test_volatility_validation_known_std(self):
        """Validate volatility calculation with known standard deviation"""
        np.random.seed(42)

        # Generate returns with known std = 2% daily
        returns = np.random.normal(0, 0.02, 252)  # 1 year daily

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        volatility = RiskMetrics.calculate_volatility(equity_curve, periods_per_year=252)

        # Expected: ~2% * sqrt(252) = ~31.7% annualized
        actual_returns = np.diff([p['equity'] for p in equity_curve]) / \
                        np.array([p['equity'] for p in equity_curve[:-1]])
        expected_vol = np.std(actual_returns, ddof=1) * np.sqrt(252) * 100

        assert abs(volatility - expected_vol) < 0.01

    def test_calmar_ratio_validation(self):
        """Validate Calmar ratio calculation"""
        # Create scenario with known CAGR and MDD
        # CAGR = 20%, MDD = -10% => Calmar = 2.0

        equity_values = [
            10000000,  # Start
            11000000,
            10000000,  # -9.09% drawdown from 11M (approx -10%)
            11000000,
            12000000,  # End (20% gain overall in 1 year)
        ]

        # Spread over 1 year
        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i*73),
                'equity': equity
            })

        calmar = RiskMetrics.calculate_calmar_ratio(equity_curve)

        # Expected: CAGR / abs(MDD) ≈ 20 / 9.09 ≈ 2.2
        assert 1.8 < calmar < 2.5  # Reasonable range

    def test_beta_validation_perfect_correlation(self):
        """Validate Beta with perfect market correlation"""
        np.random.seed(42)

        # Generate portfolio returns identical to market
        market_returns = np.random.randn(252) * 0.01

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(market_returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        beta = RiskMetrics.calculate_beta(equity_curve, market_returns[1:])

        # Expected: Beta ≈ 1.0 (perfect correlation)
        assert abs(beta - 1.0) < 0.2

    def test_cvar_validation_tail_risk(self):
        """Validate CVaR captures tail risk correctly"""
        np.random.seed(42)

        # Generate returns with fat tails
        returns = np.random.standard_t(df=3, size=1000) * 0.01

        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(hours=i),
                'equity': equity
            })

        var_95 = RiskMetrics.calculate_var(equity_curve, 0.95)
        cvar_95 = RiskMetrics.calculate_cvar(equity_curve, 0.95)

        # CVaR should be more negative than VaR (captures tail)
        assert cvar_95 < var_95

        # Manual CVaR calculation
        threshold = np.percentile(returns, 5)
        expected_cvar = np.mean(returns[returns <= threshold]) * 100

        assert abs(cvar_95 - expected_cvar) < 0.5

    def test_metrics_consistency_across_implementations(self):
        """Test that metrics are consistent across different calculation methods"""
        np.random.seed(42)

        # Generate consistent data
        returns = np.random.randn(365) * 0.015 + 0.0005
        initial_equity = 10000000
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity
            })

        # Calculate metrics
        cagr = PerformanceMetrics.calculate_cagr(equity_curve)
        sharpe = PerformanceMetrics.calculate_sharpe_ratio(equity_curve)
        volatility = RiskMetrics.calculate_volatility(equity_curve)

        # Verify relationships
        # Sharpe ≈ (CAGR - risk_free_rate) / Volatility
        risk_free_rate = 2.0  # 2%
        expected_sharpe_approx = (cagr - risk_free_rate) / volatility

        # Should be in same ballpark (allowing for calculation differences)
        assert abs(sharpe - expected_sharpe_approx) < 1.0

    @pytest.mark.slow
    def test_metrics_stability_with_resampling(self):
        """Test that metrics are stable with different data sampling"""
        np.random.seed(42)

        # Generate base returns
        returns = np.random.randn(1000) * 0.015

        initial_equity = 10000000

        # Calculate metrics with full dataset
        equity_values = initial_equity * np.exp(np.cumsum(returns))
        equity_curve = []
        for i, equity in enumerate(equity_values):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(hours=i),
                'equity': equity
            })

        sharpe_full = PerformanceMetrics.calculate_sharpe_ratio(equity_curve)
        vol_full = RiskMetrics.calculate_volatility(equity_curve)

        # Calculate with bootstrap samples
        sharpes = []
        vols = []

        for seed in range(10):
            np.random.seed(seed)
            sample_idx = np.random.choice(len(returns), size=500, replace=True)
            sample_returns = returns[sample_idx]

            equity_values = initial_equity * np.exp(np.cumsum(sample_returns))
            equity_curve = []
            for i, equity in enumerate(equity_values):
                equity_curve.append({
                    'timestamp': datetime(2024, 1, 1) + timedelta(hours=i),
                    'equity': equity
                })

            sharpes.append(PerformanceMetrics.calculate_sharpe_ratio(equity_curve))
            vols.append(RiskMetrics.calculate_volatility(equity_curve))

        # Metrics should be reasonably stable (within 50% range)
        assert abs(np.mean(sharpes) - sharpe_full) < abs(sharpe_full) * 0.5
        assert abs(np.mean(vols) - vol_full) < vol_full * 0.3
