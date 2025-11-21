"""Portfolio manager for backtesting"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Dict, List, Optional


@dataclass
class Position:
    """Position in a symbol"""
    symbol: str
    quantity: int
    avg_price: float


@dataclass
class Trade:
    """Trade record"""
    timestamp: datetime
    symbol: str
    side: str  # 'buy' or 'sell'
    quantity: int
    price: float
    commission: float
    pnl: Optional[float] = None  # Realized P&L for sells


class PortfolioManager:
    """
    Manages portfolio state during backtesting

    Tracks:
    - Cash balance
    - Positions
    - Trade history
    - Equity curve
    - P&L
    """

    def __init__(
        self,
        initial_cash: float,
        commission_rate: float = 0.0003,  # 0.03%
        tax_rate: float = 0.0023  # 0.23% (Korean stock tax)
    ):
        if initial_cash <= 0:
            raise ValueError("initial_cash must be positive")

        self.initial_cash = initial_cash
        self.cash = initial_cash
        self.commission_rate = commission_rate
        self.tax_rate = tax_rate

        self.positions: Dict[str, Position] = {}
        self.trade_history: List[Trade] = []
        self.equity_curve: List[Dict[str, float]] = []

        # P&L tracking
        self.realized_pnl = 0.0
        self.total_commission = 0.0

    def execute_buy(
        self,
        symbol: str,
        quantity: int,
        price: float,
        timestamp: datetime
    ) -> bool:
        """
        Execute a buy order

        Returns:
            True if successful, False if insufficient funds
        """
        if quantity <= 0 or price <= 0:
            raise ValueError("quantity and price must be positive")

        # Calculate total cost
        cost = quantity * price
        commission = cost * self.commission_rate
        total_cost = cost + commission

        # Check if sufficient funds
        if total_cost > self.cash:
            return False

        # Update cash
        self.cash -= total_cost
        self.total_commission += commission

        # Update position
        if symbol in self.positions:
            pos = self.positions[symbol]
            total_quantity = pos.quantity + quantity
            total_cost_basis = (pos.avg_price * pos.quantity) + (price * quantity)
            pos.avg_price = total_cost_basis / total_quantity
            pos.quantity = total_quantity
        else:
            self.positions[symbol] = Position(
                symbol=symbol,
                quantity=quantity,
                avg_price=price
            )

        # Record trade
        self.trade_history.append(Trade(
            timestamp=timestamp,
            symbol=symbol,
            side='buy',
            quantity=quantity,
            price=price,
            commission=commission
        ))

        return True

    def execute_sell(
        self,
        symbol: str,
        quantity: int,
        price: float,
        timestamp: datetime
    ) -> bool:
        """
        Execute a sell order

        Returns:
            True if successful, False if insufficient position
        """
        if quantity <= 0 or price <= 0:
            raise ValueError("quantity and price must be positive")

        # Check if position exists and has sufficient quantity
        if symbol not in self.positions:
            return False

        pos = self.positions[symbol]
        if pos.quantity < quantity:
            return False

        # Calculate proceeds
        proceeds = quantity * price
        commission = proceeds * self.commission_rate
        tax = proceeds * self.tax_rate
        net_proceeds = proceeds - commission - tax

        # Update cash
        self.cash += net_proceeds
        self.total_commission += commission

        # Calculate realized P&L
        cost_basis = pos.avg_price * quantity
        pnl = net_proceeds - cost_basis
        self.realized_pnl += pnl

        # Update position
        pos.quantity -= quantity
        if pos.quantity == 0:
            del self.positions[symbol]

        # Record trade
        self.trade_history.append(Trade(
            timestamp=timestamp,
            symbol=symbol,
            side='sell',
            quantity=quantity,
            price=price,
            commission=commission,
            pnl=pnl
        ))

        return True

    def get_equity(self, current_prices: Dict[str, float]) -> float:
        """
        Calculate total equity (cash + position values)

        Args:
            current_prices: Dict of symbol -> current price

        Returns:
            Total equity value
        """
        position_value = sum(
            pos.quantity * current_prices.get(pos.symbol, pos.avg_price)
            for pos in self.positions.values()
        )
        return self.cash + position_value

    def get_unrealized_pnl(self, current_prices: Dict[str, float]) -> float:
        """Calculate unrealized P&L on open positions"""
        unrealized = 0.0
        for pos in self.positions.values():
            current_price = current_prices.get(pos.symbol, pos.avg_price)
            current_value = pos.quantity * current_price
            cost_basis = pos.quantity * pos.avg_price
            unrealized += (current_value - cost_basis)
        return unrealized

    def update_equity_curve(
        self,
        timestamp: datetime,
        current_prices: Dict[str, float]
    ):
        """Record current equity for equity curve"""
        equity = self.get_equity(current_prices)
        self.equity_curve.append({
            'timestamp': timestamp,
            'equity': equity,
            'cash': self.cash,
            'position_value': equity - self.cash
        })

    def get_total_pnl(self, current_prices: Dict[str, float]) -> float:
        """Get total P&L (realized + unrealized)"""
        return self.realized_pnl + self.get_unrealized_pnl(current_prices)

    def get_position(self, symbol: str) -> Optional[Position]:
        """Get position for a symbol"""
        return self.positions.get(symbol)

    def get_all_positions(self) -> List[Position]:
        """Get all positions"""
        return list(self.positions.values())
