"""Performance metrics calculation"""

import numpy as np
import pandas as pd
from typing import List, Dict, Optional


class PerformanceMetrics:
    """Calculate performance metrics for backtesting results"""

    @staticmethod
    def calculate_cagr(
        equity_curve: List[Dict],
        years: Optional[float] = None
    ) -> float:
        """
        Calculate Compound Annual Growth Rate

        Args:
            equity_curve: List of equity points with 'equity' key
            years: Optional number of years (calculated from data if not provided)

        Returns:
            CAGR as percentage
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        initial_equity = equity_curve[0]['equity']
        final_equity = equity_curve[-1]['equity']

        if initial_equity <= 0:
            return 0.0

        if years is None:
            # Calculate years from timestamps
            start_date = pd.to_datetime(equity_curve[0]['timestamp'])
            end_date = pd.to_datetime(equity_curve[-1]['timestamp'])
            years = (end_date - start_date).days / 365.25

        if years <= 0:
            return 0.0

        cagr = ((final_equity / initial_equity) ** (1 / years) - 1) * 100
        return cagr

    @staticmethod
    def calculate_mdd(equity_curve: List[Dict]) -> float:
        """
        Calculate Maximum Drawdown

        Args:
            equity_curve: List of equity points with 'equity' key

        Returns:
            MDD as percentage (negative value)
        """
        if not equity_curve:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])

        # Calculate running maximum
        running_max = np.maximum.accumulate(equity_values)

        # Calculate drawdown at each point
        drawdowns = (equity_values - running_max) / running_max * 100

        # Return maximum drawdown (most negative)
        return float(np.min(drawdowns))

    @staticmethod
    def calculate_sharpe_ratio(
        equity_curve: List[Dict],
        risk_free_rate: float = 0.02,
        periods_per_year: int = 252
    ) -> float:
        """
        Calculate Sharpe Ratio

        Args:
            equity_curve: List of equity points with 'equity' key
            risk_free_rate: Annual risk-free rate (default 2%)
            periods_per_year: Number of periods per year (252 for daily)

        Returns:
            Sharpe ratio
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])
        returns = np.diff(equity_values) / equity_values[:-1]

        if len(returns) == 0:
            return 0.0

        mean_return = np.mean(returns) * periods_per_year
        std_return = np.std(returns, ddof=1) * np.sqrt(periods_per_year)

        if std_return == 0:
            return 0.0

        sharpe = (mean_return - risk_free_rate) / std_return
        return float(sharpe)

    @staticmethod
    def calculate_sortino_ratio(
        equity_curve: List[Dict],
        risk_free_rate: float = 0.02,
        periods_per_year: int = 252
    ) -> float:
        """
        Calculate Sortino Ratio (uses downside deviation)

        Args:
            equity_curve: List of equity points with 'equity' key
            risk_free_rate: Annual risk-free rate (default 2%)
            periods_per_year: Number of periods per year (252 for daily)

        Returns:
            Sortino ratio
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])
        returns = np.diff(equity_values) / equity_values[:-1]

        if len(returns) == 0:
            return 0.0

        mean_return = np.mean(returns) * periods_per_year

        # Calculate downside deviation (only negative returns)
        negative_returns = returns[returns < 0]
        if len(negative_returns) == 0:
            return float('inf')  # Perfect - no downside

        downside_std = np.std(negative_returns, ddof=1) * np.sqrt(periods_per_year)

        if downside_std == 0:
            return float('inf')

        sortino = (mean_return - risk_free_rate) / downside_std
        return float(sortino)

    @staticmethod
    def calculate_win_rate(trades: List) -> float:
        """
        Calculate win rate

        Args:
            trades: List of Trade objects with 'pnl' attribute

        Returns:
            Win rate as percentage
        """
        if not trades:
            return 0.0

        # Filter sell trades (which have pnl)
        closed_trades = [t for t in trades if hasattr(t, 'pnl') and t.pnl is not None]

        if not closed_trades:
            return 0.0

        winning_trades = sum(1 for t in closed_trades if t.pnl > 0)
        win_rate = (winning_trades / len(closed_trades)) * 100
        return win_rate

    @staticmethod
    def calculate_profit_factor(trades: List) -> float:
        """
        Calculate profit factor (gross profit / gross loss)

        Args:
            trades: List of Trade objects with 'pnl' attribute

        Returns:
            Profit factor
        """
        if not trades:
            return 0.0

        closed_trades = [t for t in trades if hasattr(t, 'pnl') and t.pnl is not None]

        if not closed_trades:
            return 0.0

        gross_profit = sum(t.pnl for t in closed_trades if t.pnl > 0)
        gross_loss = abs(sum(t.pnl for t in closed_trades if t.pnl < 0))

        if gross_loss == 0:
            return float('inf') if gross_profit > 0 else 0.0

        return gross_profit / gross_loss

    @staticmethod
    def calculate_average_win(trades: List) -> float:
        """Calculate average winning trade P&L"""
        if not trades:
            return 0.0

        closed_trades = [t for t in trades if hasattr(t, 'pnl') and t.pnl is not None]
        winning_trades = [t.pnl for t in closed_trades if t.pnl > 0]

        if not winning_trades:
            return 0.0

        return float(np.mean(winning_trades))

    @staticmethod
    def calculate_average_loss(trades: List) -> float:
        """Calculate average losing trade P&L"""
        if not trades:
            return 0.0

        closed_trades = [t for t in trades if hasattr(t, 'pnl') and t.pnl is not None]
        losing_trades = [t.pnl for t in closed_trades if t.pnl < 0]

        if not losing_trades:
            return 0.0

        return float(np.mean(losing_trades))

    @staticmethod
    def calculate_all_metrics(
        equity_curve: List[Dict],
        trades: List,
        risk_free_rate: float = 0.02
    ) -> Dict[str, float]:
        """Calculate all performance metrics at once"""
        return {
            'cagr': PerformanceMetrics.calculate_cagr(equity_curve),
            'mdd': PerformanceMetrics.calculate_mdd(equity_curve),
            'sharpe_ratio': PerformanceMetrics.calculate_sharpe_ratio(
                equity_curve, risk_free_rate
            ),
            'sortino_ratio': PerformanceMetrics.calculate_sortino_ratio(
                equity_curve, risk_free_rate
            ),
            'win_rate': PerformanceMetrics.calculate_win_rate(trades),
            'profit_factor': PerformanceMetrics.calculate_profit_factor(trades),
            'average_win': PerformanceMetrics.calculate_average_win(trades),
            'average_loss': PerformanceMetrics.calculate_average_loss(trades),
        }
