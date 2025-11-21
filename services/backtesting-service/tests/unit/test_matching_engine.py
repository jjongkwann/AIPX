"""Unit tests for Matching Engine"""

import pytest
import numpy as np
from datetime import datetime
from src.engine.matching_engine import (
    MatchingEngine, Order, OrderType, OrderSide, Fill
)


class TestOrder:
    """Test Order class"""

    def test_order_creation(self):
        """Test creating an order"""
        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=10,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        assert order.symbol == '005930'
        assert order.side == OrderSide.BUY
        assert order.quantity == 10
        assert order.order_type == OrderType.MARKET


class TestMatchingEngine:
    """Test MatchingEngine class"""

    def test_initialization(self):
        """Test matching engine initialization"""
        engine = MatchingEngine()

        assert engine.latency_mean_ms == 50.0
        assert engine.latency_std_ms == 20.0
        assert engine.slippage_mean_pct == 0.05
        assert engine.slippage_std_pct == 0.02

    def test_custom_parameters(self):
        """Test initialization with custom parameters"""
        engine = MatchingEngine(
            latency_mean_ms=100.0,
            latency_std_ms=30.0,
            slippage_mean_pct=0.1,
            slippage_std_pct=0.03
        )

        assert engine.latency_mean_ms == 100.0
        assert engine.latency_std_ms == 30.0
        assert engine.slippage_mean_pct == 0.1
        assert engine.slippage_std_pct == 0.03

    def test_match_buy_order(self, sample_orderbook):
        """Test matching a buy order"""
        engine = MatchingEngine(random_seed=42)

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            price=71100,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        assert fill is not None
        assert fill.fill_quantity == 50
        assert fill.fill_price > 0
        assert fill.latency >= 0
        assert fill.fill_timestamp > order.timestamp

    def test_match_sell_order(self, sample_orderbook):
        """Test matching a sell order"""
        engine = MatchingEngine(random_seed=42)

        order = Order(
            symbol='005930',
            side=OrderSide.SELL,
            quantity=50,
            order_type=OrderType.MARKET,
            price=71000,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        assert fill is not None
        assert fill.fill_quantity == 50
        assert fill.fill_price > 0
        assert fill.latency >= 0

    def test_match_with_empty_orderbook(self, empty_orderbook):
        """Test matching with empty order book returns None"""
        engine = MatchingEngine()

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, empty_orderbook)

        assert fill is None

    def test_match_with_no_asks(self):
        """Test buy order with no asks returns None"""
        engine = MatchingEngine()

        orderbook = {
            'bids': [{'price': 71000, 'quantity': 100}],
            'asks': []
        }

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, orderbook)

        assert fill is None

    def test_match_with_no_bids(self):
        """Test sell order with no bids returns None"""
        engine = MatchingEngine()

        orderbook = {
            'bids': [],
            'asks': [{'price': 71100, 'quantity': 100}]
        }

        order = Order(
            symbol='005930',
            side=OrderSide.SELL,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, orderbook)

        assert fill is None

    def test_latency_simulation(self):
        """Test that latency follows normal distribution"""
        engine = MatchingEngine(
            latency_mean_ms=50.0,
            latency_std_ms=20.0,
            random_seed=42
        )

        latencies = []
        for _ in range(100):
            latency = engine._simulate_latency()
            latencies.append(latency)

        # Check mean and std are approximately correct
        mean_latency = np.mean(latencies)
        std_latency = np.std(latencies)

        assert 40 < mean_latency < 60  # Within reasonable range
        assert 10 < std_latency < 30   # Within reasonable range
        assert all(l >= 0 for l in latencies)  # All non-negative

    def test_slippage_on_buy(self, sample_orderbook):
        """Test that buy orders have positive slippage (higher price)"""
        engine = MatchingEngine(random_seed=42)

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        # Fill price should be higher than best ask due to slippage
        best_ask = min(sample_orderbook['asks'], key=lambda x: x['price'])
        assert fill.fill_price >= best_ask['price']

    def test_slippage_on_sell(self, sample_orderbook):
        """Test that sell orders have negative slippage (lower price)"""
        engine = MatchingEngine(random_seed=42)

        order = Order(
            symbol='005930',
            side=OrderSide.SELL,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        # Fill price should be lower than best bid due to slippage
        best_bid = max(sample_orderbook['bids'], key=lambda x: x['price'])
        assert fill.fill_price <= best_bid['price']

    def test_fill_price_not_negative(self, sample_orderbook):
        """Test that fill price is never negative"""
        engine = MatchingEngine(
            slippage_mean_pct=10.0,  # Large slippage
            slippage_std_pct=5.0,
            random_seed=42
        )

        order = Order(
            symbol='005930',
            side=OrderSide.SELL,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        assert fill.fill_price > 0

    def test_partial_fill_insufficient_liquidity(self):
        """Test partial fill when order quantity exceeds liquidity"""
        engine = MatchingEngine()

        orderbook = {
            'bids': [{'price': 71000, 'quantity': 30}],  # Only 30 available
            'asks': [{'price': 71100, 'quantity': 30}]
        }

        order = Order(
            symbol='005930',
            side=OrderSide.SELL,
            quantity=100,  # Want to sell 100
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, orderbook)

        assert fill is not None
        assert fill.fill_quantity == 30  # Only 30 filled

    def test_full_fill_sufficient_liquidity(self, sample_orderbook):
        """Test full fill when sufficient liquidity"""
        engine = MatchingEngine()

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,  # Want 50, orderbook has 100+200+150 = 450
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        assert fill is not None
        assert fill.fill_quantity == 50  # Full fill

    def test_large_order_fill(self, orderbook_generator):
        """Test filling a large order"""
        orderbook = orderbook_generator(
            mid_price=71000,
            spread=100,
            depth=10,
            quantity_per_level=100
        )

        engine = MatchingEngine()

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=500,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, orderbook)

        assert fill is not None
        assert fill.fill_quantity == 500

    def test_zero_liquidity(self):
        """Test with zero liquidity orderbook"""
        engine = MatchingEngine()

        orderbook = {
            'bids': [{'price': 71000, 'quantity': 0}],
            'asks': [{'price': 71100, 'quantity': 0}]
        }

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=10,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, orderbook)

        assert fill.fill_quantity == 0

    def test_slippage_calculation(self, sample_orderbook):
        """Test slippage calculation is correct"""
        engine = MatchingEngine(random_seed=42)

        order = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            price=71100,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill = engine.match_order(order, sample_orderbook)

        # Slippage should be (fill_price - reference_price) / reference_price * 100
        expected_slippage = (fill.fill_price - order.price) / order.price * 100

        assert abs(fill.slippage - expected_slippage) < 0.01

    def test_consistent_results_with_seed(self, sample_orderbook):
        """Test that same seed produces consistent results"""
        engine1 = MatchingEngine(random_seed=123)
        engine2 = MatchingEngine(random_seed=123)

        order1 = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        order2 = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill1 = engine1.match_order(order1, sample_orderbook)
        fill2 = engine2.match_order(order2, sample_orderbook)

        assert abs(fill1.fill_price - fill2.fill_price) < 0.01
        assert abs(fill1.latency - fill2.latency) < 0.01

    def test_different_seeds_produce_different_results(self, sample_orderbook):
        """Test that different seeds produce different results"""
        engine1 = MatchingEngine(random_seed=123)
        engine2 = MatchingEngine(random_seed=456)

        order1 = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        order2 = Order(
            symbol='005930',
            side=OrderSide.BUY,
            quantity=50,
            order_type=OrderType.MARKET,
            timestamp=datetime(2024, 1, 1, 9, 0)
        )

        fill1 = engine1.match_order(order1, sample_orderbook)
        fill2 = engine2.match_order(order2, sample_orderbook)

        # Should be different (with high probability)
        assert fill1.fill_price != fill2.fill_price or fill1.latency != fill2.latency
