"""Matching engine with realistic latency and slippage"""

import numpy as np
from dataclasses import dataclass
from datetime import datetime, timedelta
from enum import Enum
from typing import Dict, List, Optional


class OrderType(Enum):
    """Order types"""
    MARKET = "market"
    LIMIT = "limit"


class OrderSide(Enum):
    """Order sides"""
    BUY = "buy"
    SELL = "sell"


@dataclass
class Order:
    """Order representation"""
    symbol: str
    side: OrderSide
    quantity: int
    order_type: OrderType
    price: Optional[float] = None
    timestamp: Optional[datetime] = None


@dataclass
class Fill:
    """Fill (execution) representation"""
    order: Order
    fill_price: float
    fill_quantity: int
    fill_timestamp: datetime
    slippage: float
    latency: float  # in milliseconds


class MatchingEngine:
    """
    Matching engine with realistic latency and slippage simulation

    Latency model: Normal distribution N(50ms, 20ms)
    Slippage model: Normal distribution N(0.05%, 0.02%) of price
    """

    def __init__(
        self,
        latency_mean_ms: float = 50.0,
        latency_std_ms: float = 20.0,
        slippage_mean_pct: float = 0.05,
        slippage_std_pct: float = 0.02,
        random_seed: Optional[int] = None
    ):
        self.latency_mean_ms = latency_mean_ms
        self.latency_std_ms = latency_std_ms
        self.slippage_mean_pct = slippage_mean_pct
        self.slippage_std_pct = slippage_std_pct
        self._rng = np.random.RandomState(random_seed)

    def match_order(
        self,
        order: Order,
        order_book: Dict[str, List[Dict[str, float]]]
    ) -> Optional[Fill]:
        """
        Match an order against the order book

        Args:
            order: Order to match
            order_book: Dict with 'bids' and 'asks' lists
                       Each item: {'price': float, 'quantity': int}

        Returns:
            Fill object if successful, None otherwise
        """
        if not order_book:
            return None

        # Simulate latency
        latency = self._simulate_latency()

        # Get execution price with slippage
        fill_price = self._calculate_fill_price(order, order_book)

        if fill_price is None or fill_price <= 0:
            return None

        # Calculate slippage
        reference_price = order.price if order.price else fill_price
        slippage = (fill_price - reference_price) / reference_price * 100

        # Determine fill quantity (handle partial fills)
        fill_quantity = self._calculate_fill_quantity(order, order_book)

        # Create fill
        fill_timestamp = order.timestamp + timedelta(milliseconds=latency)

        return Fill(
            order=order,
            fill_price=fill_price,
            fill_quantity=fill_quantity,
            fill_timestamp=fill_timestamp,
            slippage=slippage,
            latency=latency
        )

    def _simulate_latency(self) -> float:
        """Simulate network latency using normal distribution"""
        latency = self._rng.normal(self.latency_mean_ms, self.latency_std_ms)
        return max(0.0, latency)  # Ensure non-negative

    def _calculate_fill_price(
        self,
        order: Order,
        order_book: Dict[str, List[Dict[str, float]]]
    ) -> Optional[float]:
        """Calculate fill price with slippage"""
        if order.side == OrderSide.BUY:
            # Buy from asks (take liquidity from sellers)
            asks = order_book.get('asks', [])
            if not asks:
                return None

            # Get best ask (lowest price)
            best_ask = min(asks, key=lambda x: x['price'])
            base_price = best_ask['price']

            # Apply slippage (buy at higher price)
            slippage_pct = self._rng.normal(
                self.slippage_mean_pct,
                self.slippage_std_pct
            )
            fill_price = base_price * (1 + slippage_pct / 100)

        else:  # SELL
            # Sell to bids (take liquidity from buyers)
            bids = order_book.get('bids', [])
            if not bids:
                return None

            # Get best bid (highest price)
            best_bid = max(bids, key=lambda x: x['price'])
            base_price = best_bid['price']

            # Apply slippage (sell at lower price)
            slippage_pct = self._rng.normal(
                self.slippage_mean_pct,
                self.slippage_std_pct
            )
            fill_price = base_price * (1 - slippage_pct / 100)

        return max(0.0, fill_price)  # Ensure non-negative

    def _calculate_fill_quantity(
        self,
        order: Order,
        order_book: Dict[str, List[Dict[str, float]]]
    ) -> int:
        """Calculate fill quantity (handle partial fills)"""
        if order.side == OrderSide.BUY:
            available_liquidity = sum(
                ask['quantity'] for ask in order_book.get('asks', [])
            )
        else:
            available_liquidity = sum(
                bid['quantity'] for bid in order_book.get('bids', [])
            )

        # Return minimum of order quantity and available liquidity
        return min(order.quantity, available_liquidity) if available_liquidity > 0 else 0
