"""Risk management for strategy execution."""

from dataclasses import dataclass
from decimal import Decimal
from typing import Any
from uuid import UUID, uuid4

import structlog

from ..config import RiskConfig
from ..database import ExecutionRepository

logger = structlog.get_logger()


@dataclass
class RiskViolation:
    """Risk violation details."""

    violation_type: str
    severity: str
    description: str
    limit: float | Decimal
    current: float | Decimal
    action_required: str


@dataclass
class RiskCheckResult:
    """Result of risk check."""

    passed: bool
    violations: list[RiskViolation]
    warnings: list[RiskViolation]

    @property
    def has_critical_violations(self) -> bool:
        """Check if there are critical violations."""
        return any(v.severity == "CRITICAL" for v in self.violations)


class RiskManager:
    """Manages risk checks and limits for strategy execution."""

    def __init__(self, config: RiskConfig, repository: ExecutionRepository) -> None:
        """Initialize risk manager.

        Args:
            config: Risk configuration
            repository: Execution repository
        """
        self.config = config
        self.repository = repository
        logger.info(
            "Risk manager initialized",
            max_total_exposure=self.config.max_total_exposure,
            max_position_size=self.config.max_position_size,
            max_daily_loss=self.config.max_daily_loss,
        )

    async def check_order(
        self,
        execution_id: UUID,
        symbol: str,
        side: str,
        quantity: int,
        price: Decimal,
        current_positions: dict[str, Any],
        portfolio_value: Decimal,
    ) -> RiskCheckResult:
        """Perform pre-execution risk checks for an order.

        Args:
            execution_id: Execution ID
            symbol: Trading symbol
            side: Order side (BUY/SELL)
            quantity: Order quantity
            price: Order price
            current_positions: Current position holdings
            portfolio_value: Current portfolio value

        Returns:
            Risk check result
        """
        violations: list[RiskViolation] = []
        warnings: list[RiskViolation] = []

        # Calculate order value
        order_value = Decimal(quantity) * price

        # 1. Check minimum/maximum order size
        if order_value < Decimal(self.config.min_order_size):
            violations.append(
                RiskViolation(
                    violation_type="MIN_ORDER_SIZE",
                    severity="CRITICAL",
                    description=f"Order value ${order_value} below minimum ${self.config.min_order_size}",
                    limit=self.config.min_order_size,
                    current=float(order_value),
                    action_required="REJECT_ORDER",
                )
            )

        if order_value > Decimal(self.config.max_order_size):
            violations.append(
                RiskViolation(
                    violation_type="MAX_ORDER_SIZE",
                    severity="CRITICAL",
                    description=f"Order value ${order_value} exceeds maximum ${self.config.max_order_size}",
                    limit=self.config.max_order_size,
                    current=float(order_value),
                    action_required="REJECT_ORDER",
                )
            )

        # 2. Check position size limit
        current_symbol_position = Decimal(
            current_positions.get(symbol, {}).get("value", 0)
        )

        # Calculate new position after order execution
        if side == "BUY":
            new_position_value = current_symbol_position + order_value
        else:  # SELL
            new_position_value = current_symbol_position - order_value

        position_pct = (new_position_value / portfolio_value) if portfolio_value > 0 else Decimal(0)

        if position_pct > Decimal(self.config.max_position_size):
            violations.append(
                RiskViolation(
                    violation_type="MAX_POSITION_SIZE",
                    severity="CRITICAL",
                    description=f"Position in {symbol} would be {position_pct:.2%} of portfolio, exceeding {self.config.max_position_size:.2%}",
                    limit=float(self.config.max_position_size),
                    current=float(position_pct),
                    action_required="REDUCE_ORDER",
                )
            )
        elif position_pct > Decimal(self.config.max_position_size * 0.8):
            warnings.append(
                RiskViolation(
                    violation_type="POSITION_SIZE_WARNING",
                    severity="WARNING",
                    description=f"Position in {symbol} approaching limit at {position_pct:.2%}",
                    limit=float(self.config.max_position_size),
                    current=float(position_pct),
                    action_required="MONITOR",
                )
            )

        # 3. Check total exposure limit
        total_exposure = sum(
            Decimal(pos.get("value", 0)) for pos in current_positions.values()
        )

        if side == "BUY":
            new_total_exposure = total_exposure + order_value
        else:
            new_total_exposure = total_exposure

        if new_total_exposure > Decimal(self.config.max_total_exposure):
            violations.append(
                RiskViolation(
                    violation_type="MAX_TOTAL_EXPOSURE",
                    severity="CRITICAL",
                    description=f"Total exposure would be ${new_total_exposure}, exceeding ${self.config.max_total_exposure}",
                    limit=float(self.config.max_total_exposure),
                    current=float(new_total_exposure),
                    action_required="REJECT_ORDER",
                )
            )

        # 4. Check daily loss limit
        daily_pnl = await self.repository.get_daily_pnl(execution_id)
        daily_loss_pct = abs(daily_pnl / portfolio_value) if portfolio_value > 0 else Decimal(0)

        if daily_pnl < 0:
            if abs(daily_pnl) > Decimal(self.config.max_daily_loss_amount):
                violations.append(
                    RiskViolation(
                        violation_type="MAX_DAILY_LOSS_AMOUNT",
                        severity="CRITICAL",
                        description=f"Daily loss ${abs(daily_pnl)} exceeds limit ${self.config.max_daily_loss_amount}",
                        limit=float(self.config.max_daily_loss_amount),
                        current=float(abs(daily_pnl)),
                        action_required="HALT_TRADING",
                    )
                )

            if daily_loss_pct > Decimal(self.config.max_daily_loss):
                violations.append(
                    RiskViolation(
                        violation_type="MAX_DAILY_LOSS_PCT",
                        severity="CRITICAL",
                        description=f"Daily loss {daily_loss_pct:.2%} exceeds limit {self.config.max_daily_loss:.2%}",
                        limit=float(self.config.max_daily_loss),
                        current=float(daily_loss_pct),
                        action_required="HALT_TRADING",
                    )
                )
            elif daily_loss_pct > Decimal(self.config.max_daily_loss * 0.7):
                warnings.append(
                    RiskViolation(
                        violation_type="DAILY_LOSS_WARNING",
                        severity="WARNING",
                        description=f"Daily loss {daily_loss_pct:.2%} approaching limit",
                        limit=float(self.config.max_daily_loss),
                        current=float(daily_loss_pct),
                        action_required="MONITOR",
                    )
                )

        # Log risk check results
        if violations:
            logger.warning(
                "Risk violations detected",
                execution_id=str(execution_id),
                symbol=symbol,
                violations=[v.violation_type for v in violations],
            )
            # Log to database
            for violation in violations:
                await self._log_risk_event(execution_id, violation)

        if warnings:
            logger.info(
                "Risk warnings detected",
                execution_id=str(execution_id),
                symbol=symbol,
                warnings=[w.violation_type for w in warnings],
            )

        passed = len(violations) == 0
        return RiskCheckResult(passed=passed, violations=violations, warnings=warnings)

    async def check_stop_loss(
        self,
        execution_id: UUID,
        symbol: str,
        entry_price: Decimal,
        current_price: Decimal,
        stop_loss_pct: Decimal,
    ) -> bool:
        """Check if stop loss has been triggered.

        Args:
            execution_id: Execution ID
            symbol: Trading symbol
            entry_price: Entry price
            current_price: Current price
            stop_loss_pct: Stop loss percentage (e.g., 0.05 for 5%)

        Returns:
            True if stop loss triggered
        """
        price_change_pct = (current_price - entry_price) / entry_price
        stop_loss_with_buffer = stop_loss_pct + Decimal(self.config.stop_loss_buffer)

        if abs(price_change_pct) >= stop_loss_with_buffer:
            logger.warning(
                "Stop loss triggered",
                execution_id=str(execution_id),
                symbol=symbol,
                entry_price=float(entry_price),
                current_price=float(current_price),
                change_pct=f"{price_change_pct:.2%}",
                stop_loss_pct=f"{stop_loss_pct:.2%}",
            )

            await self._log_risk_event(
                execution_id,
                RiskViolation(
                    violation_type="STOP_LOSS_TRIGGERED",
                    severity="WARNING",
                    description=f"Stop loss triggered for {symbol}",
                    limit=float(stop_loss_pct),
                    current=float(abs(price_change_pct)),
                    action_required="CLOSE_POSITION",
                ),
            )
            return True

        return False

    async def _log_risk_event(self, execution_id: UUID, violation: RiskViolation) -> None:
        """Log risk event to database.

        Args:
            execution_id: Execution ID
            violation: Risk violation
        """
        await self.repository.create_risk_event(
            event_id=uuid4(),
            execution_id=execution_id,
            event_type=violation.violation_type,
            severity=violation.severity,
            description=violation.description,
            data={
                "limit": float(violation.limit),
                "current": float(violation.current),
            },
            action_taken=violation.action_required,
        )
