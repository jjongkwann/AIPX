"""Risk metrics calculation"""

import numpy as np
import pandas as pd
from typing import List, Dict, Optional


class RiskMetrics:
    """Calculate risk metrics for backtesting results"""

    @staticmethod
    def calculate_var(
        equity_curve: List[Dict],
        confidence_level: float = 0.95
    ) -> float:
        """
        Calculate Value at Risk (VaR)

        Args:
            equity_curve: List of equity points with 'equity' key
            confidence_level: Confidence level (0.95 or 0.99)

        Returns:
            VaR as percentage of portfolio value
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])
        returns = np.diff(equity_values) / equity_values[:-1]

        if len(returns) == 0:
            return 0.0

        # Calculate VaR at given confidence level
        var = np.percentile(returns, (1 - confidence_level) * 100)
        return float(var * 100)  # Return as percentage

    @staticmethod
    def calculate_cvar(
        equity_curve: List[Dict],
        confidence_level: float = 0.95
    ) -> float:
        """
        Calculate Conditional Value at Risk (CVaR) / Expected Shortfall

        Args:
            equity_curve: List of equity points with 'equity' key
            confidence_level: Confidence level (0.95 or 0.99)

        Returns:
            CVaR as percentage of portfolio value
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])
        returns = np.diff(equity_values) / equity_values[:-1]

        if len(returns) == 0:
            return 0.0

        # Calculate VaR threshold
        var_threshold = np.percentile(returns, (1 - confidence_level) * 100)

        # Calculate average of returns below VaR threshold
        tail_returns = returns[returns <= var_threshold]

        if len(tail_returns) == 0:
            return 0.0

        cvar = np.mean(tail_returns)
        return float(cvar * 100)  # Return as percentage

    @staticmethod
    def calculate_beta(
        equity_curve: List[Dict],
        market_returns: Optional[np.ndarray] = None
    ) -> float:
        """
        Calculate Beta (systematic risk)

        Args:
            equity_curve: List of equity points with 'equity' key
            market_returns: Market returns array (if None, returns 1.0)

        Returns:
            Beta value
        """
        if not equity_curve or len(equity_curve) < 2:
            return 1.0

        if market_returns is None:
            return 1.0  # Default beta if no market data

        equity_values = np.array([point['equity'] for point in equity_curve])
        portfolio_returns = np.diff(equity_values) / equity_values[:-1]

        # Ensure arrays have same length
        min_len = min(len(portfolio_returns), len(market_returns))
        portfolio_returns = portfolio_returns[:min_len]
        market_returns = market_returns[:min_len]

        if len(portfolio_returns) == 0:
            return 1.0

        # Calculate beta: covariance(portfolio, market) / variance(market)
        covariance = np.cov(portfolio_returns, market_returns)[0, 1]
        market_variance = np.var(market_returns, ddof=1)

        if market_variance == 0:
            return 1.0

        beta = covariance / market_variance
        return float(beta)

    @staticmethod
    def calculate_volatility(
        equity_curve: List[Dict],
        periods_per_year: int = 252
    ) -> float:
        """
        Calculate annualized volatility (standard deviation)

        Args:
            equity_curve: List of equity points with 'equity' key
            periods_per_year: Number of periods per year (252 for daily)

        Returns:
            Annualized volatility as percentage
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        equity_values = np.array([point['equity'] for point in equity_curve])
        returns = np.diff(equity_values) / equity_values[:-1]

        if len(returns) == 0:
            return 0.0

        volatility = np.std(returns, ddof=1) * np.sqrt(periods_per_year)
        return float(volatility * 100)  # Return as percentage

    @staticmethod
    def calculate_calmar_ratio(
        equity_curve: List[Dict],
        years: Optional[float] = None
    ) -> float:
        """
        Calculate Calmar Ratio (CAGR / abs(MDD))

        Args:
            equity_curve: List of equity points with 'equity' key
            years: Optional number of years

        Returns:
            Calmar ratio
        """
        if not equity_curve or len(equity_curve) < 2:
            return 0.0

        # Import here to avoid circular dependency
        from .performance_metrics import PerformanceMetrics

        cagr = PerformanceMetrics.calculate_cagr(equity_curve, years)
        mdd = PerformanceMetrics.calculate_mdd(equity_curve)

        if mdd == 0:
            return float('inf') if cagr > 0 else 0.0

        calmar = cagr / abs(mdd)
        return float(calmar)

    @staticmethod
    def calculate_all_metrics(
        equity_curve: List[Dict],
        market_returns: Optional[np.ndarray] = None
    ) -> Dict[str, float]:
        """Calculate all risk metrics at once"""
        return {
            'var_95': RiskMetrics.calculate_var(equity_curve, 0.95),
            'var_99': RiskMetrics.calculate_var(equity_curve, 0.99),
            'cvar_95': RiskMetrics.calculate_cvar(equity_curve, 0.95),
            'cvar_99': RiskMetrics.calculate_cvar(equity_curve, 0.99),
            'beta': RiskMetrics.calculate_beta(equity_curve, market_returns),
            'volatility': RiskMetrics.calculate_volatility(equity_curve),
            'calmar_ratio': RiskMetrics.calculate_calmar_ratio(equity_curve),
        }
