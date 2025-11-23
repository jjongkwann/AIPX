"""End-to-end backtesting integration tests"""

import pandas as pd
import pytest
from src.engine.event_loop import Event, EventLoop, EventType
from src.engine.matching_engine import MatchingEngine, Order, OrderSide, OrderType
from src.engine.portfolio import PortfolioManager
from src.metrics.performance_metrics import PerformanceMetrics
from src.metrics.risk_metrics import RiskMetrics


class SimpleMomentumStrategy:
    """
    Simple momentum strategy for testing
    Buy when price increases 2% over 20-period lookback
    Sell when price drops 2% from entry
    """

    def __init__(self, lookback=20, buy_threshold=0.02, sell_threshold=-0.02):
        self.lookback = lookback
        self.buy_threshold = buy_threshold
        self.sell_threshold = sell_threshold
        self.price_history = []
        self.entry_price = None
        self.position_open = False

    def on_market_data(self, event, portfolio, matching_engine, event_loop):
        """Process market data and generate signals"""
        data = event.data
        current_price = data["close"]
        symbol = data["symbol"]

        self.price_history.append(current_price)

        # Need enough history
        if len(self.price_history) < self.lookback + 1:
            return

        # Calculate momentum
        old_price = self.price_history[-self.lookback]
        momentum = (current_price - old_price) / old_price

        # Generate orderbook for matching
        orderbook = self._generate_orderbook(current_price)

        # Trading logic
        if not self.position_open and momentum > self.buy_threshold:
            # Buy signal
            order = Order(
                symbol=symbol,
                side=OrderSide.BUY,
                quantity=10,
                order_type=OrderType.MARKET,
                price=current_price,
                timestamp=event.timestamp,
            )

            fill = matching_engine.match_order(order, orderbook)

            if fill and portfolio.execute_buy(
                symbol=symbol, quantity=fill.fill_quantity, price=fill.fill_price, timestamp=fill.fill_timestamp
            ):
                self.entry_price = fill.fill_price
                self.position_open = True

        elif self.position_open and self.entry_price is not None:
            # Check exit condition
            pnl_pct = (current_price - self.entry_price) / self.entry_price

            if pnl_pct <= self.sell_threshold:
                # Sell signal
                position = portfolio.get_position(symbol)

                if position:
                    order = Order(
                        symbol=symbol,
                        side=OrderSide.SELL,
                        quantity=position.quantity,
                        order_type=OrderType.MARKET,
                        price=current_price,
                        timestamp=event.timestamp,
                    )

                    fill = matching_engine.match_order(order, orderbook)

                    if fill:
                        portfolio.execute_sell(
                            symbol=symbol,
                            quantity=fill.fill_quantity,
                            price=fill.fill_price,
                            timestamp=fill.fill_timestamp,
                        )
                        self.position_open = False
                        self.entry_price = None

        # Update equity curve
        portfolio.update_equity_curve(event.timestamp, {symbol: current_price})

    def _generate_orderbook(self, mid_price):
        """Generate synthetic orderbook"""
        spread = mid_price * 0.001  # 0.1% spread

        return {
            "bids": [
                {"price": mid_price - spread, "quantity": 100},
                {"price": mid_price - spread * 2, "quantity": 200},
            ],
            "asks": [
                {"price": mid_price + spread, "quantity": 100},
                {"price": mid_price + spread * 2, "quantity": 200},
            ],
        }


@pytest.mark.integration
class TestBacktestEndToEnd:
    """End-to-end backtest integration tests"""

    def test_complete_backtest_workflow(self, market_data_generator):
        """Test complete backtest with momentum strategy"""
        # Generate 1 year of daily data
        market_data = market_data_generator(
            start_date="2024-01-01",
            end_date="2024-12-31",
            symbol="005930",
            initial_price=70000,
            volatility=0.02,
            trend=0.0003,
            seed=42,
        )

        # Initialize components
        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)
        strategy = SimpleMomentumStrategy(lookback=20)

        # Register handler
        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        # Load market data as events
        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        # Run backtest
        processed = event_loop.process_events()

        # Verify events were processed
        assert processed == len(market_data)

        # Verify portfolio has reasonable state
        final_equity = portfolio.get_equity({"005930": market_data.iloc[-1]["close"]})
        assert final_equity > 0

        # Verify trades were executed
        assert len(portfolio.trade_history) > 0

        # Verify equity curve was recorded
        assert len(portfolio.equity_curve) > 0

        # Calculate metrics
        metrics = PerformanceMetrics.calculate_all_metrics(portfolio.equity_curve, portfolio.trade_history)

        # Verify all metrics are present
        assert "cagr" in metrics
        assert "mdd" in metrics
        assert "sharpe_ratio" in metrics
        assert "win_rate" in metrics

        # Metrics should be reasonable
        assert -100 < metrics["cagr"] < 200  # CAGR between -100% and 200%
        assert -100 <= metrics["mdd"] <= 0  # MDD should be negative or zero
        assert 0 <= metrics["win_rate"] <= 100  # Win rate 0-100%

    def test_backtest_with_multiple_symbols(self, market_data_generator):
        """Test backtest with multiple symbols"""
        # Generate data for 2 symbols
        data1 = market_data_generator(symbol="005930", initial_price=70000, seed=42)
        data2 = market_data_generator(symbol="000660", initial_price=100000, seed=43)

        combined_data = pd.concat([data1, data2]).sort_values("timestamp")

        # Initialize components
        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=20000000)

        # Simple strategy that trades both symbols
        strategies = {"005930": SimpleMomentumStrategy(lookback=20), "000660": SimpleMomentumStrategy(lookback=20)}

        def market_handler(event):
            symbol = event.data["symbol"]
            if symbol in strategies:
                strategies[symbol].on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        # Load events
        for _, row in combined_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        # Run backtest
        processed = event_loop.process_events()

        assert processed == len(combined_data)

        # Verify trades in both symbols
        symbols_traded = set(trade.symbol for trade in portfolio.trade_history)
        assert len(symbols_traded) >= 1  # At least one symbol traded

    def test_backtest_final_portfolio_value_reasonable(self, market_data_generator):
        """Test that final portfolio value is reasonable"""
        market_data = market_data_generator(
            start_date="2024-01-01", end_date="2024-12-31", initial_price=70000, volatility=0.015, trend=0.0002, seed=42
        )

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        initial_cash = 10000000
        portfolio = PortfolioManager(initial_cash=initial_cash)
        strategy = SimpleMomentumStrategy()

        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        event_loop.process_events()

        final_equity = portfolio.get_equity({"005930": market_data.iloc[-1]["close"]})

        # Final equity should be within reasonable range of initial
        assert final_equity > initial_cash * 0.5  # Not lost more than 50%
        assert final_equity < initial_cash * 3.0  # Not gained more than 200%

    def test_backtest_trade_history_complete(self, market_data_generator):
        """Test that trade history is complete and consistent"""
        market_data = market_data_generator(seed=42)

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)
        strategy = SimpleMomentumStrategy()

        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        event_loop.process_events()

        # Check trade history
        trades = portfolio.trade_history

        if len(trades) > 0:
            # Verify timestamps are in order
            timestamps = [t.timestamp for t in trades]
            assert timestamps == sorted(timestamps)

            # Verify buy/sell alternation (mostly)
            # Count buys and sells
            buys = sum(1 for t in trades if t.side == "buy")
            sells = sum(1 for t in trades if t.side == "sell")

            # Sells should not exceed buys
            assert sells <= buys

    def test_backtest_equity_curve_monotonic_timestamps(self, market_data_generator):
        """Test that equity curve has monotonic timestamps"""
        market_data = market_data_generator(seed=42)

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)
        strategy = SimpleMomentumStrategy()

        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        event_loop.process_events()

        # Verify equity curve timestamps are in order
        if len(portfolio.equity_curve) > 1:
            timestamps = [point["timestamp"] for point in portfolio.equity_curve]
            assert timestamps == sorted(timestamps)

    def test_backtest_metrics_validation(self, market_data_generator):
        """Test that calculated metrics are valid"""
        market_data = market_data_generator(seed=42)

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)
        strategy = SimpleMomentumStrategy()

        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        event_loop.process_events()

        # Calculate all metrics
        perf_metrics = PerformanceMetrics.calculate_all_metrics(portfolio.equity_curve, portfolio.trade_history)

        risk_metrics = RiskMetrics.calculate_all_metrics(portfolio.equity_curve)

        # Verify performance metrics ranges
        assert isinstance(perf_metrics["cagr"], float)
        assert isinstance(perf_metrics["mdd"], float)
        assert perf_metrics["mdd"] <= 0  # MDD should be negative or zero
        assert 0 <= perf_metrics["win_rate"] <= 100
        assert perf_metrics["profit_factor"] >= 0

        # Verify risk metrics ranges
        assert risk_metrics["volatility"] >= 0
        assert risk_metrics["var_95"] <= 0  # VaR should be negative
        assert risk_metrics["var_99"] <= risk_metrics["var_95"]  # 99% more extreme

    @pytest.mark.slow
    def test_backtest_performance_1year_daily(self, market_data_generator):
        """Test backtest performance with 1 year of daily data"""
        import time

        market_data = market_data_generator(start_date="2024-01-01", end_date="2024-12-31", seed=42)

        event_loop = EventLoop()
        matching_engine = MatchingEngine(random_seed=42)
        portfolio = PortfolioManager(initial_cash=10000000)
        strategy = SimpleMomentumStrategy()

        def market_handler(event):
            strategy.on_market_data(event, portfolio, matching_engine, event_loop)

        event_loop.register_handler(EventType.MARKET_DATA, market_handler)

        for _, row in market_data.iterrows():
            event = Event(
                timestamp=row["timestamp"],
                event_type=EventType.MARKET_DATA,
                data={
                    "symbol": row["symbol"],
                    "open": row["open"],
                    "high": row["high"],
                    "low": row["low"],
                    "close": row["close"],
                    "volume": row["volume"],
                },
            )
            event_loop.add_event(event)

        # Measure execution time
        start_time = time.time()
        event_loop.process_events()
        end_time = time.time()

        execution_time = end_time - start_time

        # Should complete in less than 5 seconds
        assert execution_time < 5.0

        print(f"Backtest execution time: {execution_time:.2f}s")
