"""Performance benchmarks for metrics calculation"""

import pytest
import time
import numpy as np
from datetime import datetime, timedelta

from src.metrics.performance_metrics import PerformanceMetrics
from src.metrics.risk_metrics import RiskMetrics
from src.engine.portfolio import Trade


@pytest.mark.performance
class TestMetricsPerformance:
    """Performance benchmarks for metrics calculation"""

    def generate_large_equity_curve(self, num_points=10000):
        """Generate large equity curve for benchmarking"""
        np.random.seed(42)
        dates = [datetime(2024, 1, 1) + timedelta(hours=i) for i in range(num_points)]
        initial_equity = 10000000

        returns = np.random.randn(num_points) * 0.01
        equity_values = initial_equity * np.exp(np.cumsum(returns))

        equity_curve = []
        for i, date in enumerate(dates):
            equity_curve.append({
                'timestamp': date,
                'equity': equity_values[i],
                'cash': equity_values[i] * 0.3,
                'position_value': equity_values[i] * 0.7
            })

        return equity_curve

    def generate_large_trade_history(self, num_trades=1000):
        """Generate large trade history for benchmarking"""
        np.random.seed(42)
        trades = []

        for i in range(num_trades):
            pnl = np.random.randn() * 50000 if i % 2 == 1 else None

            trade = Trade(
                timestamp=datetime(2024, 1, 1) + timedelta(hours=i),
                symbol='005930',
                side='buy' if i % 2 == 0 else 'sell',
                quantity=10,
                price=70000 + np.random.randn() * 1000,
                commission=300,
                pnl=pnl
            )
            trades.append(trade)

        return trades

    def test_cagr_performance_10k_points(self):
        """Benchmark CAGR calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):  # Run 100 times
            cagr = PerformanceMetrics.calculate_cagr(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000  # ms per call

        print(f"\nCAGR Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"CAGR took {execution_time:.2f}ms, target is < 100ms"

    def test_mdd_performance_10k_points(self):
        """Benchmark MDD calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            mdd = PerformanceMetrics.calculate_mdd(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nMDD Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"MDD took {execution_time:.2f}ms, target is < 100ms"

    def test_sharpe_ratio_performance_10k_points(self):
        """Benchmark Sharpe ratio calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            sharpe = PerformanceMetrics.calculate_sharpe_ratio(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nSharpe Ratio Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"Sharpe took {execution_time:.2f}ms, target is < 100ms"

    def test_sortino_ratio_performance_10k_points(self):
        """Benchmark Sortino ratio calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            sortino = PerformanceMetrics.calculate_sortino_ratio(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nSortino Ratio Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"Sortino took {execution_time:.2f}ms, target is < 100ms"

    def test_win_rate_performance_1k_trades(self):
        """Benchmark win rate calculation with 1,000 trades"""
        trades = self.generate_large_trade_history(1000)

        # Benchmark
        start_time = time.time()
        for _ in range(1000):
            win_rate = PerformanceMetrics.calculate_win_rate(trades)
        end_time = time.time()

        execution_time = (end_time - start_time) / 1000 * 1000

        print(f"\nWin Rate Performance (1k trades):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 50ms
        assert execution_time < 50, f"Win rate took {execution_time:.2f}ms, target is < 50ms"

    def test_profit_factor_performance_1k_trades(self):
        """Benchmark profit factor calculation with 1,000 trades"""
        trades = self.generate_large_trade_history(1000)

        # Benchmark
        start_time = time.time()
        for _ in range(1000):
            pf = PerformanceMetrics.calculate_profit_factor(trades)
        end_time = time.time()

        execution_time = (end_time - start_time) / 1000 * 1000

        print(f"\nProfit Factor Performance (1k trades):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 50ms
        assert execution_time < 50, f"Profit factor took {execution_time:.2f}ms, target is < 50ms"

    def test_var_performance_10k_points(self):
        """Benchmark VaR calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            var = RiskMetrics.calculate_var(equity_curve, 0.95)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nVaR Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"VaR took {execution_time:.2f}ms, target is < 100ms"

    def test_cvar_performance_10k_points(self):
        """Benchmark CVaR calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            cvar = RiskMetrics.calculate_cvar(equity_curve, 0.95)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nCVaR Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"CVaR took {execution_time:.2f}ms, target is < 100ms"

    def test_volatility_performance_10k_points(self):
        """Benchmark volatility calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            vol = RiskMetrics.calculate_volatility(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nVolatility Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"Volatility took {execution_time:.2f}ms, target is < 100ms"

    def test_beta_performance_10k_points(self):
        """Benchmark Beta calculation with 10,000 data points"""
        equity_curve = self.generate_large_equity_curve(10000)
        market_returns = np.random.randn(9999) * 0.01

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            beta = RiskMetrics.calculate_beta(equity_curve, market_returns)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nBeta Performance (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 100ms
        assert execution_time < 100, f"Beta took {execution_time:.2f}ms, target is < 100ms"

    def test_all_performance_metrics_10k_points(self):
        """Benchmark calculating all performance metrics at once"""
        equity_curve = self.generate_large_equity_curve(10000)
        trades = self.generate_large_trade_history(1000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            metrics = PerformanceMetrics.calculate_all_metrics(equity_curve, trades)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nAll Performance Metrics (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 500ms for all metrics
        assert execution_time < 500, f"All metrics took {execution_time:.2f}ms, target is < 500ms"

    def test_all_risk_metrics_10k_points(self):
        """Benchmark calculating all risk metrics at once"""
        equity_curve = self.generate_large_equity_curve(10000)

        # Benchmark
        start_time = time.time()
        for _ in range(100):
            metrics = RiskMetrics.calculate_all_metrics(equity_curve)
        end_time = time.time()

        execution_time = (end_time - start_time) / 100 * 1000

        print(f"\nAll Risk Metrics (10k points):")
        print(f"  Time per calculation: {execution_time:.2f}ms")

        # Target: < 500ms for all metrics
        assert execution_time < 500, f"All risk metrics took {execution_time:.2f}ms, target is < 500ms"

    @pytest.mark.slow
    def test_metrics_scaling_performance(self):
        """Test how metrics performance scales with data size"""
        sizes = [100, 1000, 5000, 10000]
        results = []

        for size in sizes:
            equity_curve = self.generate_large_equity_curve(size)

            start_time = time.time()
            for _ in range(10):
                PerformanceMetrics.calculate_cagr(equity_curve)
                PerformanceMetrics.calculate_mdd(equity_curve)
                PerformanceMetrics.calculate_sharpe_ratio(equity_curve)
            end_time = time.time()

            avg_time = (end_time - start_time) / 10 * 1000
            results.append((size, avg_time))

        print(f"\nMetrics Scaling Performance:")
        for size, time_ms in results:
            print(f"  {size:5d} points: {time_ms:6.2f}ms")

        # Check that scaling is roughly linear (not exponential)
        # Time for 10k should be < 20x time for 1k
        time_1k = results[1][1]
        time_10k = results[3][1]

        assert time_10k < time_1k * 20, "Metrics scaling is not efficient"
