"""LangGraph state definition for conversation management."""

from operator import add
from typing import Annotated, TypedDict

from langgraph.graph import MessagesState


class ConversationState(MessagesState):
    """
    LangGraph conversation state with typed annotations.

    This state manages the entire conversation flow, user profile extraction,
    and strategy generation process.
    """

    # User context
    user_id: str
    session_id: str

    # User profile slots (extracted from conversation)
    risk_tolerance: str | None = None  # "conservative", "moderate", "aggressive"
    capital: int | None = None  # Investment capital in USD
    preferred_sectors: Annotated[list[str], add] | None = None  # Technology, Healthcare, etc.
    investment_horizon: str | None = None  # "short", "medium", "long"

    # Market analysis state
    market_sentiment: str | None = None  # "bullish", "bearish", "neutral"
    recommended_sectors: Annotated[list[str], add] | None = None
    market_analysis: str | None = None  # Detailed market analysis text

    # Strategy generation state
    strategy_draft: dict | None = None  # Generated strategy configuration
    approval_status: str = "pending"  # "pending", "approved", "rejected"
    rejection_reason: str | None = None  # User's reason for rejection

    # Agent orchestration
    current_agent: str | None = None  # Name of currently active agent
    retry_count: int = 0  # Number of retry attempts
    next_agent: str | None = None  # Next agent to execute
    is_complete: bool = False  # Whether the workflow is complete

    # Tracking and debugging
    agent_history: Annotated[list[str], add] | None = None  # Sequence of agents called
    errors: Annotated[list[str], add] | None = None  # Any errors encountered


class AgentDecision(TypedDict):
    """Decision output from supervisor agent."""

    next_agent: str  # Name of next agent to call
    reason: str  # Reason for this decision
    is_complete: bool  # Whether workflow should end


class UserProfileSlots(TypedDict, total=False):
    """User profile information slots."""

    risk_tolerance: str
    capital: int
    preferred_sectors: list[str]
    investment_horizon: str


class MarketAnalysisResult(TypedDict):
    """Market analysis result from MarketAnalyst agent."""

    sentiment: str  # "bullish", "bearish", "neutral"
    recommended_sectors: list[str]
    analysis: str  # Detailed analysis text
    confidence: float  # Confidence score 0-1


class StrategyConfigDraft(TypedDict):
    """Draft strategy configuration."""

    name: str
    description: str
    asset_allocation: dict[str, float]  # {"AAPL": 0.3, "MSFT": 0.2}
    max_position_size: float  # Percentage
    max_drawdown: float  # Percentage
    stop_loss_pct: float  # Percentage
    entry_conditions: list[str]
    exit_conditions: list[str]
