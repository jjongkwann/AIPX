"""Pytest configuration and fixtures"""

import pytest
import numpy as np
import pandas as pd
from datetime import datetime, timedelta


@pytest.fixture
def sample_tick_data():
    """Generate synthetic tick data for testing"""
    dates = pd.date_range('2024-01-01', '2024-12-31', freq='D')
    np.random.seed(42)

    # Generate price series with trend and noise
    # Starting at 100,000 KRW
    base_price = 100000
    returns = np.random.randn(len(dates)) * 0.02  # 2% daily volatility
    price_multipliers = np.exp(np.cumsum(returns))
    prices = base_price * price_multipliers

    return pd.DataFrame({
        'timestamp': dates,
        'symbol': '005930',  # Samsung Electronics
        'open': prices * 0.99,
        'high': prices * 1.02,
        'low': prices * 0.98,
        'close': prices,
        'volume': np.random.randint(1000000, 10000000, len(dates))
    })


@pytest.fixture
def sample_orderbook():
    """Generate sample order book"""
    return {
        'bids': [
            {'price': 71000, 'quantity': 100},
            {'price': 70900, 'quantity': 200},
            {'price': 70800, 'quantity': 150},
        ],
        'asks': [
            {'price': 71100, 'quantity': 100},
            {'price': 71200, 'quantity': 200},
            {'price': 71300, 'quantity': 150},
        ]
    }


@pytest.fixture
def empty_orderbook():
    """Generate empty order book"""
    return {
        'bids': [],
        'asks': []
    }


@pytest.fixture
def sample_equity_curve():
    """Generate sample equity curve for testing metrics"""
    np.random.seed(42)
    dates = pd.date_range('2024-01-01', '2024-12-31', freq='D')
    initial_equity = 10000000  # 10M KRW

    # Generate equity curve with some growth and volatility
    returns = np.random.randn(len(dates)) * 0.015 + 0.0003  # Slight positive drift
    equity_multipliers = np.exp(np.cumsum(returns))
    equity_values = initial_equity * equity_multipliers

    equity_curve = []
    for i, date in enumerate(dates):
        equity_curve.append({
            'timestamp': date,
            'equity': equity_values[i],
            'cash': equity_values[i] * 0.3,  # 30% cash
            'position_value': equity_values[i] * 0.7  # 70% in positions
        })

    return equity_curve


@pytest.fixture
def sample_trades():
    """Generate sample trades for testing"""
    from src.engine.portfolio import Trade

    trades = []
    base_time = datetime(2024, 1, 1, 9, 0)

    # Generate 20 trades with varied P&L
    np.random.seed(42)
    for i in range(10):
        # Buy trade
        trades.append(Trade(
            timestamp=base_time + timedelta(days=i*2),
            symbol='005930',
            side='buy',
            quantity=10,
            price=100000 + i * 1000,
            commission=300
        ))

        # Sell trade with P&L
        pnl = np.random.randn() * 100000  # Random P&L
        trades.append(Trade(
            timestamp=base_time + timedelta(days=i*2+1),
            symbol='005930',
            side='sell',
            quantity=10,
            price=100000 + i * 1000 + pnl / 10,
            commission=300,
            pnl=pnl
        ))

    return trades


@pytest.fixture
def known_returns_data():
    """Generate data with known statistical properties for validation"""
    np.random.seed(123)

    # Generate 1 year of daily data
    dates = pd.date_range('2024-01-01', '2024-12-31', freq='D')
    initial_equity = 10000000

    # Known parameters
    annual_return = 0.15  # 15% annual return
    annual_volatility = 0.20  # 20% annual volatility
    daily_return = annual_return / 252
    daily_volatility = annual_volatility / np.sqrt(252)

    # Generate returns
    returns = np.random.normal(daily_return, daily_volatility, len(dates))
    equity_multipliers = np.exp(np.cumsum(returns))
    equity_values = initial_equity * equity_multipliers

    equity_curve = []
    for i, date in enumerate(dates):
        equity_curve.append({
            'timestamp': date,
            'equity': equity_values[i],
            'cash': equity_values[i] * 0.2,
            'position_value': equity_values[i] * 0.8
        })

    return {
        'equity_curve': equity_curve,
        'expected_annual_return': annual_return,
        'expected_annual_volatility': annual_volatility
    }


@pytest.fixture
def drawdown_data():
    """Generate data with known drawdown for testing"""
    dates = pd.date_range('2024-01-01', '2024-12-31', freq='D')
    initial_equity = 10000000

    # Create a known drawdown pattern
    equity_values = []
    for i in range(len(dates)):
        if i < 100:
            # Growth phase
            equity = initial_equity * (1 + i * 0.001)
        elif i < 150:
            # Drawdown phase: -20% from peak
            peak = initial_equity * (1 + 99 * 0.001)
            equity = peak * (1 - (i - 100) * 0.004)
        else:
            # Recovery phase
            bottom = initial_equity * (1 + 99 * 0.001) * 0.8
            equity = bottom * (1 + (i - 150) * 0.002)

        equity_values.append(equity)

    equity_curve = []
    for i, date in enumerate(dates):
        equity_curve.append({
            'timestamp': date,
            'equity': equity_values[i],
            'cash': equity_values[i] * 0.3,
            'position_value': equity_values[i] * 0.7
        })

    return {
        'equity_curve': equity_curve,
        'expected_mdd': -20.0  # -20% maximum drawdown
    }


@pytest.fixture
def market_data_generator():
    """Factory fixture to generate market data with specific parameters"""
    def _generate(
        start_date='2024-01-01',
        end_date='2024-12-31',
        symbol='005930',
        initial_price=100000,
        volatility=0.02,
        trend=0.0003,
        seed=42
    ):
        np.random.seed(seed)
        dates = pd.date_range(start_date, end_date, freq='D')

        returns = np.random.randn(len(dates)) * volatility + trend
        price_multipliers = np.exp(np.cumsum(returns))
        prices = initial_price * price_multipliers

        return pd.DataFrame({
            'timestamp': dates,
            'symbol': symbol,
            'open': prices * 0.99,
            'high': prices * 1.02,
            'low': prices * 0.98,
            'close': prices,
            'volume': np.random.randint(1000000, 10000000, len(dates))
        })

    return _generate


@pytest.fixture
def orderbook_generator():
    """Factory fixture to generate order books with specific parameters"""
    def _generate(
        mid_price=71000,
        spread=100,
        depth=5,
        quantity_per_level=100
    ):
        bids = []
        asks = []

        for i in range(depth):
            bids.append({
                'price': mid_price - spread/2 - i * spread,
                'quantity': quantity_per_level * (i + 1)
            })
            asks.append({
                'price': mid_price + spread/2 + i * spread,
                'quantity': quantity_per_level * (i + 1)
            })

        return {'bids': bids, 'asks': asks}

    return _generate
