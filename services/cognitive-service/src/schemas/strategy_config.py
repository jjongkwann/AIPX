"""Strategy configuration schema definitions."""

from datetime import datetime
from typing import Any, Literal
from uuid import UUID

from pydantic import BaseModel, Field, field_validator


class StrategyConfig(BaseModel):
    """Complete strategy configuration for execution."""

    strategy_id: UUID
    user_id: UUID
    name: str = Field(..., min_length=1, max_length=255)
    description: str = Field(..., min_length=1, max_length=1000)

    # Asset allocation
    asset_allocation: dict[str, float] = Field(..., description="Asset allocation map: symbol -> weight (0-1)")

    # Risk parameters
    max_position_size: float = Field(..., gt=0, le=1.0, description="Maximum position size as percentage of capital")
    max_drawdown: float = Field(..., gt=0, le=1.0, description="Maximum allowed drawdown percentage")
    stop_loss_pct: float = Field(..., gt=0, le=1.0, description="Stop loss percentage per position")
    take_profit_pct: float = Field(default=0.2, gt=0, description="Take profit percentage per position")

    # Entry/exit rules
    entry_conditions: list[str] = Field(default_factory=list, description="List of entry condition rules")
    exit_conditions: list[str] = Field(default_factory=list, description="List of exit condition rules")

    # Technical indicators configuration
    indicators: dict[str, Any] = Field(default_factory=dict, description="Technical indicator parameters")

    # Metadata
    status: Literal["draft", "approved", "active", "paused", "terminated"] = Field(default="draft")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    approved_at: datetime | None = None
    activated_at: datetime | None = None

    @field_validator("asset_allocation")
    @classmethod
    def validate_allocation(cls, v: dict[str, float]) -> dict[str, float]:
        """Validate asset allocation weights sum to 1.0."""
        if not v:
            raise ValueError("Asset allocation cannot be empty")

        total = sum(v.values())
        if not (0.99 <= total <= 1.01):  # Allow small floating point errors
            raise ValueError(f"Asset allocation must sum to 1.0, got {total}")

        for symbol, weight in v.items():
            if not (0 < weight <= 1.0):
                raise ValueError(f"Weight for {symbol} must be between 0 and 1, got {weight}")

        return v

    @field_validator("indicators")
    @classmethod
    def validate_indicators(cls, v: dict[str, Any]) -> dict[str, Any]:
        """Validate indicator configuration."""
        valid_indicators = {"RSI", "MACD", "EMA", "SMA", "BB", "ATR"}
        invalid = set(v.keys()) - valid_indicators
        if invalid:
            raise ValueError(f"Invalid indicators: {invalid}. Valid: {valid_indicators}")
        return v

    class Config:
        json_schema_extra = {
            "example": {
                "strategy_id": "550e8400-e29b-41d4-a716-446655440000",
                "user_id": "650e8400-e29b-41d4-a716-446655440000",
                "name": "Tech Growth Strategy",
                "description": "Moderate risk strategy focusing on tech sector growth stocks",
                "asset_allocation": {
                    "AAPL": 0.3,
                    "MSFT": 0.25,
                    "GOOGL": 0.25,
                    "NVDA": 0.2,
                },
                "max_position_size": 0.3,
                "max_drawdown": 0.15,
                "stop_loss_pct": 0.05,
                "take_profit_pct": 0.2,
                "entry_conditions": [
                    "RSI < 30",
                    "MACD bullish crossover",
                    "Price above 50-day SMA",
                ],
                "exit_conditions": [
                    "RSI > 70",
                    "MACD bearish crossover",
                    "Stop loss hit",
                ],
                "indicators": {
                    "RSI": {"period": 14},
                    "MACD": {"fast": 12, "slow": 26, "signal": 9},
                    "SMA": {"period": 50},
                },
                "status": "draft",
            }
        }


class StrategyConfigCreate(BaseModel):
    """Schema for creating a new strategy configuration."""

    name: str = Field(..., min_length=1, max_length=255)
    description: str = Field(..., min_length=1, max_length=1000)
    asset_allocation: dict[str, float]
    max_position_size: float = Field(default=0.25, gt=0, le=1.0)
    max_drawdown: float = Field(default=0.2, gt=0, le=1.0)
    stop_loss_pct: float = Field(default=0.05, gt=0, le=1.0)
    take_profit_pct: float = Field(default=0.15, gt=0)
    entry_conditions: list[str] = Field(default_factory=list)
    exit_conditions: list[str] = Field(default_factory=list)
    indicators: dict[str, Any] = Field(default_factory=dict)


class StrategyConfigUpdate(BaseModel):
    """Schema for updating strategy configuration."""

    name: str | None = Field(None, min_length=1, max_length=255)
    description: str | None = Field(None, min_length=1, max_length=1000)
    asset_allocation: dict[str, float] | None = None
    max_position_size: float | None = Field(None, gt=0, le=1.0)
    max_drawdown: float | None = Field(None, gt=0, le=1.0)
    stop_loss_pct: float | None = Field(None, gt=0, le=1.0)
    take_profit_pct: float | None = Field(None, gt=0)
    entry_conditions: list[str] | None = None
    exit_conditions: list[str] | None = None
    indicators: dict[str, Any] | None = None
    status: Literal["draft", "approved", "active", "paused", "terminated"] | None = None


class StrategyApproval(BaseModel):
    """Schema for strategy approval/rejection."""

    approved: bool
    reason: str | None = Field(None, max_length=500)
