"""Main strategy execution logic."""

import asyncio
from decimal import Decimal
from typing import Any
from uuid import UUID, uuid4

import structlog

from ..config import Config
from ..database import ExecutionRepository
from ..grpc_client import OMSClient, OrderRequest
from ..monitor import PositionMonitor
from ..risk import RiskManager

logger = structlog.get_logger()


class StrategyExecutor:
    """Executes trading strategies."""

    def __init__(
        self,
        config: Config,
        repository: ExecutionRepository,
        oms_client: OMSClient,
        risk_manager: RiskManager,
    ) -> None:
        """Initialize strategy executor.

        Args:
            config: Application configuration
            repository: Execution repository
            oms_client: OMS gRPC client
            risk_manager: Risk manager
        """
        self.config = config
        self.repository = repository
        self.oms_client = oms_client
        self.risk_manager = risk_manager
        self._active_executions: dict[UUID, PositionMonitor] = {}
        self._execution_tasks: dict[UUID, asyncio.Task] = {}

    async def execute_strategy(self, strategy_data: dict[str, Any]) -> None:
        """Execute an approved strategy.

        Args:
            strategy_data: Strategy data from Kafka message
        """
        strategy_id = UUID(strategy_data["strategy_id"])
        user_id = UUID(strategy_data["user_id"])
        config = strategy_data["config"]

        execution_id = uuid4()

        logger.info(
            "Starting strategy execution",
            execution_id=str(execution_id),
            strategy_id=str(strategy_id),
            user_id=str(user_id),
        )

        try:
            # 1. Create execution record
            await self.repository.create_execution(
                execution_id=execution_id,
                strategy_id=strategy_id,
                user_id=user_id,
                config=config,
            )

            # 2. Initialize position monitor
            position_monitor = PositionMonitor(
                execution_id=execution_id,
                repository=self.repository,
                check_interval=self.config.execution.position_check_interval,
            )
            await position_monitor.start()
            self._active_executions[execution_id] = position_monitor

            # 3. Parse strategy configuration
            symbols = config.get("symbols", [])
            allocation = config.get("allocation", {})
            initial_capital = Decimal(str(config.get("initial_capital", 100000)))

            # 4. Execute initial orders
            await self._execute_initial_orders(
                execution_id=execution_id,
                strategy_id=strategy_id,
                symbols=symbols,
                allocation=allocation,
                initial_capital=initial_capital,
                position_monitor=position_monitor,
            )

            # 5. Start rebalancing task
            task = asyncio.create_task(
                self._rebalance_loop(
                    execution_id=execution_id,
                    config=config,
                    position_monitor=position_monitor,
                )
            )
            self._execution_tasks[execution_id] = task

            logger.info(
                "Strategy execution started successfully",
                execution_id=str(execution_id),
            )

        except Exception as e:
            logger.error(
                "Failed to execute strategy",
                execution_id=str(execution_id),
                error=str(e),
            )
            # Update status to error
            await self.repository.update_execution_status(
                execution_id=execution_id,
                status="error",
            )
            raise

    async def stop_execution(self, execution_id: UUID) -> None:
        """Stop a strategy execution.

        Args:
            execution_id: Execution ID to stop
        """
        logger.info("Stopping strategy execution", execution_id=str(execution_id))

        # Cancel rebalancing task
        if execution_id in self._execution_tasks:
            task = self._execution_tasks[execution_id]
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass
            del self._execution_tasks[execution_id]

        # Stop position monitor
        if execution_id in self._active_executions:
            monitor = self._active_executions[execution_id]
            await monitor.stop()
            del self._active_executions[execution_id]

        # Close all positions
        await self._close_all_positions(execution_id)

        # Update status
        from datetime import datetime

        await self.repository.update_execution_status(
            execution_id=execution_id,
            status="stopped",
            stopped_at=datetime.now(),
        )

        logger.info("Strategy execution stopped", execution_id=str(execution_id))

    async def _execute_initial_orders(
        self,
        execution_id: UUID,
        strategy_id: UUID,
        symbols: list[str],
        allocation: dict[str, float],
        initial_capital: Decimal,
        position_monitor: PositionMonitor,
    ) -> None:
        """Execute initial orders based on allocation.

        Args:
            execution_id: Execution ID
            strategy_id: Strategy ID
            symbols: Trading symbols
            allocation: Symbol allocation percentages
            initial_capital: Initial capital
            position_monitor: Position monitor
        """
        logger.info(
            "Executing initial orders",
            execution_id=str(execution_id),
            symbols=symbols,
            allocation=allocation,
        )

        for symbol in symbols:
            alloc_pct = Decimal(str(allocation.get(symbol, 0)))
            if alloc_pct <= 0:
                continue

            # Calculate order size
            order_value = initial_capital * alloc_pct

            # Get current price (placeholder - would come from market data feed)
            current_price = Decimal("100.00")  # TODO: Get from market data

            quantity = int(order_value / current_price)
            if quantity <= 0:
                continue

            # Create order
            order_id = uuid4()
            await self.repository.create_order(
                order_id=order_id,
                execution_id=execution_id,
                symbol=symbol,
                side="BUY",
                order_type="LIMIT",
                quantity=quantity,
                price=current_price,
            )

            # Risk check
            portfolio_value = await position_monitor.get_portfolio_value()
            if portfolio_value == 0:
                portfolio_value = initial_capital

            risk_result = await self.risk_manager.check_order(
                execution_id=execution_id,
                symbol=symbol,
                side="BUY",
                quantity=quantity,
                price=current_price,
                current_positions={},
                portfolio_value=portfolio_value,
            )

            if not risk_result.passed:
                logger.warning(
                    "Order failed risk check",
                    order_id=str(order_id),
                    violations=[v.violation_type for v in risk_result.violations],
                )
                await self.repository.update_order_status(
                    order_id=order_id,
                    status="REJECTED",
                    error_message="Risk check failed",
                )
                continue

            # Submit order to OMS
            try:
                order_request = OrderRequest(
                    order_id=order_id,
                    symbol=symbol,
                    side="BUY",
                    order_type="LIMIT",
                    quantity=quantity,
                    price=float(current_price),
                    strategy_id=strategy_id,
                )

                # Note: This will work after proto generation
                # response = await self.oms_client.submit_order(order_request)

                # Placeholder response
                logger.info(
                    "Order submitted (placeholder)",
                    order_id=str(order_id),
                    symbol=symbol,
                )

                # Update order status
                await self.repository.update_order_status(
                    order_id=order_id,
                    status="ACCEPTED",
                    oms_order_id="OMS_" + str(order_id)[:8],
                )

                # Add to position monitor
                await position_monitor.add_position(
                    symbol=symbol,
                    quantity=quantity,
                    entry_price=current_price,
                    side="LONG",
                )

            except Exception as e:
                logger.error(
                    "Failed to submit order",
                    order_id=str(order_id),
                    error=str(e),
                )
                await self.repository.update_order_status(
                    order_id=order_id,
                    status="REJECTED",
                    error_message=str(e),
                )

    async def _rebalance_loop(
        self,
        execution_id: UUID,
        config: dict[str, Any],
        position_monitor: PositionMonitor,
    ) -> None:
        """Periodic rebalancing loop.

        Args:
            execution_id: Execution ID
            config: Strategy configuration
            position_monitor: Position monitor
        """
        rebalance_interval = self.config.execution.rebalance_interval

        logger.info(
            "Starting rebalance loop",
            execution_id=str(execution_id),
            interval=rebalance_interval,
        )

        try:
            while True:
                await asyncio.sleep(rebalance_interval)

                try:
                    await self._rebalance_positions(
                        execution_id=execution_id,
                        config=config,
                        position_monitor=position_monitor,
                    )
                except Exception as e:
                    logger.error(
                        "Error during rebalancing",
                        execution_id=str(execution_id),
                        error=str(e),
                    )

        except asyncio.CancelledError:
            logger.info("Rebalance loop cancelled", execution_id=str(execution_id))

    async def _rebalance_positions(
        self,
        execution_id: UUID,
        config: dict[str, Any],
        position_monitor: PositionMonitor,
    ) -> None:
        """Rebalance positions based on strategy.

        Args:
            execution_id: Execution ID
            config: Strategy configuration
            position_monitor: Position monitor
        """
        logger.info("Rebalancing positions", execution_id=str(execution_id))

        # Get current positions
        current_positions = position_monitor.positions
        target_allocation = config.get("allocation", {})

        # Calculate portfolio value
        portfolio_value = await position_monitor.get_portfolio_value()

        if portfolio_value == 0:
            logger.warning("Portfolio value is zero, skipping rebalance")
            return

        # Check each symbol
        for symbol, target_pct in target_allocation.items():
            target_value = portfolio_value * Decimal(str(target_pct))

            if symbol in current_positions:
                current_value = current_positions[symbol].market_value
                diff_value = target_value - current_value
                diff_pct = abs(diff_value / portfolio_value)

                # Only rebalance if difference is significant (>5%)
                if diff_pct > Decimal("0.05"):
                    logger.info(
                        "Rebalancing needed",
                        symbol=symbol,
                        current_pct=float(current_value / portfolio_value),
                        target_pct=target_pct,
                    )
                    # TODO: Submit rebalancing orders
            else:
                # Symbol not in portfolio but in target allocation
                if Decimal(str(target_pct)) > 0:
                    logger.info("New symbol needed in portfolio", symbol=symbol)
                    # TODO: Submit buy orders

    async def _close_all_positions(self, execution_id: UUID) -> None:
        """Close all positions for an execution.

        Args:
            execution_id: Execution ID
        """
        if execution_id not in self._active_executions:
            return

        position_monitor = self._active_executions[execution_id]

        logger.info(
            "Closing all positions",
            execution_id=str(execution_id),
            positions=len(position_monitor.positions),
        )

        # Close each position
        for symbol, position in list(position_monitor.positions.items()):
            # Get current price (placeholder)
            current_price = position.current_price

            # Submit sell order
            order_id = uuid4()
            await self.repository.create_order(
                order_id=order_id,
                execution_id=execution_id,
                symbol=symbol,
                side="SELL",
                order_type="MARKET",
                quantity=position.quantity,
            )

            # TODO: Submit to OMS
            logger.info("Position close order submitted", symbol=symbol, order_id=str(order_id))

            # Close position in monitor
            await position_monitor.close_position(symbol, current_price)
