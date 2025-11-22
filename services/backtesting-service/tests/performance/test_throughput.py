"""Throughput benchmark tests"""

import pytest
import time
import numpy as np
from datetime import datetime, timedelta

from src.engine.event_loop import EventLoop, Event, EventType
from src.engine.matching_engine import MatchingEngine
from src.engine.portfolio import PortfolioManager


@pytest.mark.performance
class TestThroughputBenchmarks:
    """Performance benchmarks for backtesting throughput"""

    @pytest.mark.slow
    def test_backtest_1year_daily_data(self, market_data_generator):
        """Benchmark: 1 year of daily data should complete < 5 seconds"""
        market_data = market_data_generator(
            start_date='2024-01-01',
            end_date='2024-12-31',
            seed=42
        )

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)

        def simple_handler(event):
            """Simple handler that just processes data"""
            data = event.data
            current_price = data['close']
            symbol = data['symbol']

            orderbook = {
                'bids': [{'price': current_price * 0.999, 'quantity': 100}],
                'asks': [{'price': current_price * 1.001, 'quantity': 100}]
            }

            portfolio.update_equity_curve(
                event.timestamp,
                {symbol: current_price}
            )

        event_loop.register_handler(EventType.MARKET_DATA, simple_handler)

        # Load events
        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row['timestamp'],
                event_type=EventType.MARKET_DATA,
                data={
                    'symbol': row['symbol'],
                    'open': row['open'],
                    'high': row['high'],
                    'low': row['low'],
                    'close': row['close'],
                    'volume': row['volume']
                }
            )
            event_loop.add_event(event)

        # Benchmark
        start_time = time.time()
        processed = event_loop.process_events()
        end_time = time.time()

        execution_time = end_time - start_time

        print(f"\n1 Year Daily Data Benchmark:")
        print(f"  Events processed: {processed}")
        print(f"  Execution time: {execution_time:.3f}s")
        print(f"  Throughput: {processed/execution_time:.0f} events/sec")

        # Target: < 5 seconds
        assert execution_time < 5.0, f"Execution took {execution_time:.2f}s, target is < 5.0s"

    @pytest.mark.slow
    def test_backtest_1month_tick_data(self):
        """Benchmark: 1 month of simulated tick data should complete < 30 seconds"""
        # Generate high-frequency tick data (every minute for 1 month)
        start_date = datetime(2024, 1, 1)
        end_date = datetime(2024, 2, 1)

        # ~43,200 minutes in 30 days
        timestamps = []
        current = start_date
        while current < end_date:
            timestamps.append(current)
            current += timedelta(minutes=1)

        # Generate price series
        np.random.seed(42)
        initial_price = 70000
        returns = np.random.randn(len(timestamps)) * 0.0005  # Much smaller moves
        prices = initial_price * np.exp(np.cumsum(returns))

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)

        def simple_handler(event):
            data = event.data
            current_price = data['price']
            portfolio.update_equity_curve(event.timestamp, {'005930': current_price})

        event_loop.register_handler(EventType.MARKET_DATA, simple_handler)

        # Load events
        for i, ts in enumerate(timestamps):
            event = Event(
                timestamp=ts,
                event_type=EventType.MARKET_DATA,
                data={'symbol': '005930', 'price': prices[i]}
            )
            event_loop.add_event(event)

        # Benchmark
        start_time = time.time()
        processed = event_loop.process_events()
        end_time = time.time()

        execution_time = end_time - start_time

        print(f"\n1 Month Tick Data Benchmark:")
        print(f"  Events processed: {processed}")
        print(f"  Execution time: {execution_time:.3f}s")
        print(f"  Throughput: {processed/execution_time:.0f} events/sec")

        # Target: < 30 seconds
        assert execution_time < 30.0, f"Execution took {execution_time:.2f}s, target is < 30.0s"

    def test_order_matching_throughput(self, sample_orderbook):
        """Benchmark order matching throughput"""
        engine = MatchingEngine(random_seed=42)

        from src.engine.matching_engine import Order, OrderSide, OrderType

        orders = []
        base_time = datetime(2024, 1, 1, 9, 0, 0)
        for i in range(1000):
            order = Order(
                symbol='005930',
                side=OrderSide.BUY if i % 2 == 0 else OrderSide.SELL,
                quantity=10,
                order_type=OrderType.MARKET,
                price=71000,
                timestamp=base_time + timedelta(seconds=i)
            )
            orders.append(order)

        # Benchmark
        start_time = time.time()
        fills = []
        for order in orders:
            fill = engine.match_order(order, sample_orderbook)
            fills.append(fill)
        end_time = time.time()

        execution_time = end_time - start_time

        print(f"\nOrder Matching Benchmark:")
        print(f"  Orders matched: {len(orders)}")
        print(f"  Execution time: {execution_time:.3f}s")
        print(f"  Throughput: {len(orders)/execution_time:.0f} orders/sec")

        # Should handle > 1000 orders/sec
        assert len(orders)/execution_time > 1000

    def test_portfolio_operations_throughput(self):
        """Benchmark portfolio operations throughput"""
        portfolio = PortfolioManager(initial_cash=100000000)

        # Execute many trades
        num_trades = 1000
        base_time = datetime(2024, 1, 1, 9, 0, 0)
        start_time = time.time()

        for i in range(num_trades):
            if i % 2 == 0:
                # Buy
                portfolio.execute_buy(
                    symbol='005930',
                    quantity=10,
                    price=70000 + (i % 100),
                    timestamp=base_time + timedelta(seconds=i)
                )
            else:
                # Sell (if we have position)
                if portfolio.get_position('005930'):
                    portfolio.execute_sell(
                        symbol='005930',
                        quantity=10,
                        price=71000 + (i % 100),
                        timestamp=base_time + timedelta(seconds=i)
                    )

        end_time = time.time()
        execution_time = end_time - start_time

        print(f"\nPortfolio Operations Benchmark:")
        print(f"  Operations: {num_trades}")
        print(f"  Execution time: {execution_time:.3f}s")
        print(f"  Throughput: {num_trades/execution_time:.0f} ops/sec")

        # Should handle > 5000 ops/sec
        assert num_trades/execution_time > 5000

    @pytest.mark.slow
    def test_memory_usage_1year_backtest(self, market_data_generator):
        """Benchmark memory usage during 1 year backtest"""
        import psutil
        import os

        process = psutil.Process(os.getpid())
        initial_memory = process.memory_info().rss / 1024 / 1024  # MB

        market_data = market_data_generator(
            start_date='2024-01-01',
            end_date='2024-12-31',
            seed=42
        )

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)

        def simple_handler(event):
            data = event.data
            current_price = data['close']
            symbol = data['symbol']
            portfolio.update_equity_curve(event.timestamp, {symbol: current_price})

        event_loop.register_handler(EventType.MARKET_DATA, simple_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row['timestamp'],
                event_type=EventType.MARKET_DATA,
                data={
                    'symbol': row['symbol'],
                    'open': row['open'],
                    'high': row['high'],
                    'low': row['low'],
                    'close': row['close'],
                    'volume': row['volume']
                }
            )
            event_loop.add_event(event)

        event_loop.process_events()

        final_memory = process.memory_info().rss / 1024 / 1024  # MB
        memory_used = final_memory - initial_memory

        print(f"\nMemory Usage Benchmark:")
        print(f"  Initial memory: {initial_memory:.1f} MB")
        print(f"  Final memory: {final_memory:.1f} MB")
        print(f"  Memory used: {memory_used:.1f} MB")

        # Should use < 500 MB for 1 year daily data
        assert memory_used < 500, f"Memory usage {memory_used:.1f}MB exceeds 500MB limit"

    def test_concurrent_backtests_throughput(self, market_data_generator):
        """Benchmark running multiple backtests concurrently"""
        import concurrent.futures

        def run_single_backtest(seed):
            """Run a single backtest"""
            market_data = market_data_generator(seed=seed)

            event_loop = EventLoop()
            matching_engine = MatchingEngine(random_seed=seed)
            portfolio = PortfolioManager(initial_cash=10000000)

            def handler(event):
                data = event.data
                portfolio.update_equity_curve(
                    event.timestamp,
                    {data['symbol']: data['close']}
                )

            event_loop.register_handler(EventType.MARKET_DATA, handler)

            for _, row in market_data.iterrows():
                event = Event(
                    timestamp=row['timestamp'],
                    event_type=EventType.MARKET_DATA,
                    data={
                        'symbol': row['symbol'],
                        'open': row['open'],
                        'high': row['high'],
                        'low': row['low'],
                        'close': row['close'],
                        'volume': row['volume']
                    }
                )
                event_loop.add_event(event)

            event_loop.process_events()
            return portfolio.equity_curve[-1]['equity'] if portfolio.equity_curve else 0

        # Run 4 backtests concurrently
        num_backtests = 4
        start_time = time.time()

        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
            futures = [executor.submit(run_single_backtest, i) for i in range(num_backtests)]
            results = [f.result() for f in concurrent.futures.as_completed(futures)]

        end_time = time.time()
        execution_time = end_time - start_time

        print(f"\nConcurrent Backtests Benchmark:")
        print(f"  Number of backtests: {num_backtests}")
        print(f"  Execution time: {execution_time:.3f}s")
        print(f"  Time per backtest: {execution_time/num_backtests:.3f}s")

        # All backtests should complete
        assert len(results) == num_backtests
        assert all(r > 0 for r in results)
