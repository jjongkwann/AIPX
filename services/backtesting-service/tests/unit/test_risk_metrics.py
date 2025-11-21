"""Unit tests for Risk Metrics"""

import pytest
import numpy as np
from datetime import datetime, timedelta
from src.metrics.risk_metrics import RiskMetrics


class TestRiskMetrics:
    """Test RiskMetrics class"""

    def test_calculate_var_95(self, sample_equity_curve):
        """Test VaR calculation at 95% confidence"""
        var = RiskMetrics.calculate_var(sample_equity_curve, 0.95)

        # VaR should be negative (loss)
        assert isinstance(var, float)
        assert var < 0  # VaR represents potential loss

    def test_calculate_var_99(self, sample_equity_curve):
        """Test VaR calculation at 99% confidence"""
        var = RiskMetrics.calculate_var(sample_equity_curve, 0.99)

        # VaR at 99% should be more extreme than 95%
        var_95 = RiskMetrics.calculate_var(sample_equity_curve, 0.95)
        assert abs(var) >= abs(var_95)

    def test_calculate_var_empty_curve(self):
        """Test VaR with empty equity curve"""
        var = RiskMetrics.calculate_var([])

        assert var == 0.0

    def test_calculate_var_single_point(self):
        """Test VaR with single data point"""
        equity_curve = [
            {'timestamp': datetime(2024, 1, 1), 'equity': 10000000}
        ]

        var = RiskMetrics.calculate_var(equity_curve)

        assert var == 0.0

    def test_calculate_cvar_95(self, sample_equity_curve):
        """Test CVaR calculation at 95% confidence"""
        cvar = RiskMetrics.calculate_cvar(sample_equity_curve, 0.95)

        # CVaR should be negative and more extreme than VaR
        var = RiskMetrics.calculate_var(sample_equity_curve, 0.95)
        assert cvar <= var  # CVaR is average of tail, should be worse

    def test_calculate_cvar_99(self, sample_equity_curve):
        """Test CVaR calculation at 99% confidence"""
        cvar = RiskMetrics.calculate_cvar(sample_equity_curve, 0.99)

        # Should be more extreme than 95% CVaR
        cvar_95 = RiskMetrics.calculate_cvar(sample_equity_curve, 0.95)
        assert abs(cvar) >= abs(cvar_95)

    def test_calculate_cvar_empty_curve(self):
        """Test CVaR with empty equity curve"""
        cvar = RiskMetrics.calculate_cvar([])

        assert cvar == 0.0

    def test_calculate_beta_default(self, sample_equity_curve):
        """Test beta with no market returns (defaults to 1.0)"""
        beta = RiskMetrics.calculate_beta(sample_equity_curve)

        assert beta == 1.0

    def test_calculate_beta_with_market(self, sample_equity_curve):
        """Test beta calculation with market returns"""
        # Generate synthetic market returns
        np.random.seed(42)
        market_returns = np.random.randn(len(sample_equity_curve) - 1) * 0.01

        beta = RiskMetrics.calculate_beta(sample_equity_curve, market_returns)

        # Beta should be a reasonable number
        assert isinstance(beta, float)
        assert -3 < beta < 3  # Reasonable range

    def test_calculate_beta_empty_curve(self):
        """Test beta with empty curve"""
        beta = RiskMetrics.calculate_beta([])

        assert beta == 1.0

    def test_calculate_volatility(self, sample_equity_curve):
        """Test volatility calculation"""
        volatility = RiskMetrics.calculate_volatility(sample_equity_curve)

        # Volatility should be positive percentage
        assert volatility > 0
        assert volatility < 200  # Reasonable upper bound

    def test_calculate_volatility_zero(self):
        """Test volatility with constant equity"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000
            })

        volatility = RiskMetrics.calculate_volatility(equity_curve)

        assert volatility == 0.0

    def test_calculate_volatility_empty_curve(self):
        """Test volatility with empty curve"""
        volatility = RiskMetrics.calculate_volatility([])

        assert volatility == 0.0

    def test_calculate_calmar_ratio(self, sample_equity_curve):
        """Test Calmar ratio calculation"""
        calmar = RiskMetrics.calculate_calmar_ratio(sample_equity_curve)

        # Calmar should be numeric
        assert isinstance(calmar, float)

    def test_calculate_calmar_ratio_no_drawdown(self):
        """Test Calmar ratio with no drawdown"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000 * (1 + i * 0.01)  # Continuous growth
            })

        calmar = RiskMetrics.calculate_calmar_ratio(equity_curve)

        # Should be infinite (no drawdown)
        assert calmar == float('inf')

    def test_calculate_calmar_ratio_negative_cagr(self):
        """Test Calmar ratio with negative returns"""
        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': 10000000 * (1 - i * 0.001)  # Declining
            })

        calmar = RiskMetrics.calculate_calmar_ratio(equity_curve)

        # Should be negative (negative CAGR / negative MDD)
        assert isinstance(calmar, float)

    def test_calculate_calmar_ratio_empty_curve(self):
        """Test Calmar ratio with empty curve"""
        calmar = RiskMetrics.calculate_calmar_ratio([])

        assert calmar == 0.0

    def test_calculate_all_metrics(self, sample_equity_curve):
        """Test calculating all risk metrics at once"""
        metrics = RiskMetrics.calculate_all_metrics(sample_equity_curve)

        # Check all metrics are present
        assert 'var_95' in metrics
        assert 'var_99' in metrics
        assert 'cvar_95' in metrics
        assert 'cvar_99' in metrics
        assert 'beta' in metrics
        assert 'volatility' in metrics
        assert 'calmar_ratio' in metrics

        # Check all are numeric
        for key, value in metrics.items():
            assert isinstance(value, (int, float))

    def test_calculate_all_metrics_with_market(self, sample_equity_curve):
        """Test all metrics with market returns"""
        np.random.seed(42)
        market_returns = np.random.randn(len(sample_equity_curve) - 1) * 0.01

        metrics = RiskMetrics.calculate_all_metrics(
            sample_equity_curve,
            market_returns
        )

        # Beta should be calculated with market data
        assert metrics['beta'] != 1.0

    def test_var_cvar_relationship(self, sample_equity_curve):
        """Test that CVaR is more extreme than VaR"""
        var_95 = RiskMetrics.calculate_var(sample_equity_curve, 0.95)
        cvar_95 = RiskMetrics.calculate_cvar(sample_equity_curve, 0.95)

        # CVaR should be <= VaR (more negative)
        assert cvar_95 <= var_95

    def test_volatility_scaling(self, sample_equity_curve):
        """Test volatility scales correctly with period"""
        # Daily volatility
        daily_vol = RiskMetrics.calculate_volatility(
            sample_equity_curve,
            periods_per_year=252
        )

        # Weekly volatility (52 weeks)
        weekly_vol = RiskMetrics.calculate_volatility(
            sample_equity_curve,
            periods_per_year=52
        )

        # Should be different due to scaling
        assert daily_vol != weekly_vol

    def test_beta_with_perfect_correlation(self):
        """Test beta with perfectly correlated returns"""
        np.random.seed(42)

        # Generate equity curve
        returns = np.random.randn(100) * 0.01
        equity_values = 10000000 * np.exp(np.cumsum(returns))

        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity_values[i]
            })

        # Use same returns as market
        market_returns = returns[1:]  # Skip first to match diff

        beta = RiskMetrics.calculate_beta(equity_curve, market_returns)

        # Beta should be close to 1.0 (perfect correlation)
        assert abs(beta - 1.0) < 0.3

    def test_beta_with_inverse_correlation(self):
        """Test beta with inversely correlated returns"""
        np.random.seed(42)

        # Generate equity curve with inverse market returns
        market_returns = np.random.randn(99) * 0.01
        portfolio_returns = -market_returns  # Inverse

        equity_values = 10000000 * np.exp(np.cumsum(portfolio_returns))

        equity_curve = []
        for i in range(100):
            equity_curve.append({
                'timestamp': datetime(2024, 1, 1) + timedelta(days=i),
                'equity': equity_values[i]
            })

        beta = RiskMetrics.calculate_beta(equity_curve, market_returns)

        # Beta should be negative (inverse correlation)
        assert beta < 0

    def test_volatility_with_known_data(self, known_returns_data):
        """Test volatility matches expected value with known data"""
        volatility = RiskMetrics.calculate_volatility(
            known_returns_data['equity_curve'],
            periods_per_year=252
        )

        expected_vol = known_returns_data['expected_annual_volatility'] * 100

        # Should be close (within reasonable tolerance due to sampling)
        assert abs(volatility - expected_vol) < 5.0

    def test_metrics_consistency(self, sample_equity_curve):
        """Test that metrics are consistent across multiple calls"""
        var1 = RiskMetrics.calculate_var(sample_equity_curve, 0.95)
        var2 = RiskMetrics.calculate_var(sample_equity_curve, 0.95)

        assert var1 == var2

        volatility1 = RiskMetrics.calculate_volatility(sample_equity_curve)
        volatility2 = RiskMetrics.calculate_volatility(sample_equity_curve)

        assert volatility1 == volatility2

    def test_var_cvar_different_confidence_levels(self, sample_equity_curve):
        """Test VaR and CVaR at different confidence levels"""
        var_90 = RiskMetrics.calculate_var(sample_equity_curve, 0.90)
        var_95 = RiskMetrics.calculate_var(sample_equity_curve, 0.95)
        var_99 = RiskMetrics.calculate_var(sample_equity_curve, 0.99)

        # Higher confidence should give more extreme values
        assert abs(var_90) <= abs(var_95) <= abs(var_99)

        cvar_90 = RiskMetrics.calculate_cvar(sample_equity_curve, 0.90)
        cvar_95 = RiskMetrics.calculate_cvar(sample_equity_curve, 0.95)
        cvar_99 = RiskMetrics.calculate_cvar(sample_equity_curve, 0.99)

        assert abs(cvar_90) <= abs(cvar_95) <= abs(cvar_99)
