"""Strategy architect agent for generating investment strategies."""

from typing import Any

import structlog
from langchain_core.messages import AIMessage
from pydantic import BaseModel, Field

from src.graph.state import ConversationState, StrategyConfigDraft
from src.services.llm_service import LLMService

logger = structlog.get_logger()


class GeneratedStrategy(BaseModel):
    """Generated strategy configuration."""

    name: str = Field(..., description="Strategy name")
    description: str = Field(..., description="Strategy description")
    asset_allocation: dict[str, float] = Field(..., description="Asset allocation by symbol")
    max_position_size: float = Field(..., description="Maximum position size (0-1)")
    max_drawdown: float = Field(..., description="Maximum drawdown (0-1)")
    stop_loss_pct: float = Field(..., description="Stop loss percentage (0-1)")
    take_profit_pct: float = Field(default=0.2, description="Take profit percentage")
    entry_conditions: list[str] = Field(default_factory=list, description="Entry condition rules")
    exit_conditions: list[str] = Field(default_factory=list, description="Exit condition rules")
    indicators: dict[str, Any] = Field(default_factory=dict, description="Technical indicators configuration")
    rationale: str = Field(..., description="Strategy rationale")


class StrategyArchitectAgent:
    """
    Agent for generating investment strategies.

    Responsibilities:
    - Generate strategy configuration based on user profile and market analysis
    - Define asset allocation
    - Set risk parameters (stop loss, max drawdown, position sizing)
    - Define entry/exit rules
    - Configure technical indicators
    """

    def __init__(self, llm_service: LLMService):
        """Initialize strategy architect agent."""
        self.llm_service = llm_service
        self.system_prompt = """You are an expert quantitative strategy architect specializing in systematic trading strategies.

Your task is to design executable trading strategies based on:
1. User investment profile (risk tolerance, capital, preferences, horizon)
2. Market analysis and recommendations
3. Best practices in risk management and portfolio construction

Strategy Components:
1. **Asset Allocation**: Distribute capital across recommended stocks
2. **Risk Parameters**:
   - Max position size (% of capital per stock)
   - Stop loss percentage
   - Take profit percentage
   - Maximum portfolio drawdown
3. **Entry Conditions**: When to enter positions
4. **Exit Conditions**: When to exit positions
5. **Technical Indicators**: RSI, MACD, SMA, etc.

Guidelines for Risk Levels:
**Conservative:**
- Max position size: 10-15%
- Stop loss: 3-5%
- Max drawdown: 10-15%
- Focus on value stocks and dividends

**Moderate:**
- Max position size: 20-30%
- Stop loss: 5-7%
- Max drawdown: 15-20%
- Balanced growth and value

**Aggressive:**
- Max position size: 30-40%
- Stop loss: 7-10%
- Max drawdown: 20-30%
- Growth and momentum focus

Output Requirements:
- Asset allocation must sum to 1.0
- Use 3-6 stocks from recommended sectors
- Include specific entry/exit conditions
- Configure relevant technical indicators
"""

    async def process(self, state: ConversationState) -> ConversationState:
        """
        Generate investment strategy based on profile and analysis.

        Args:
            state: Current conversation state

        Returns:
            Updated conversation state
        """
        try:
            # Get user profile
            risk_tolerance = state.get("risk_tolerance", "moderate")
            capital = state.get("capital", 100000)
            preferred_sectors = state.get("preferred_sectors", [])
            investment_horizon = state.get("investment_horizon", "medium")

            # Get market analysis
            market_sentiment = state.get("market_sentiment", "neutral")
            recommended_sectors = state.get("recommended_sectors", [])
            market_analysis = state.get("market_analysis", "")

            # Create strategy generation prompt
            strategy_prompt = f"""Design a trading strategy for the following investor:

**User Profile:**
- Risk Tolerance: {risk_tolerance}
- Capital: ${capital:,}
- Preferred Sectors: {", ".join(preferred_sectors) if preferred_sectors else "None specified"}
- Investment Horizon: {investment_horizon}

**Market Context:**
- Sentiment: {market_sentiment}
- Recommended Sectors: {", ".join(recommended_sectors)}
- Analysis: {market_analysis[:500]}

**Requirements:**
1. Create asset allocation across 3-6 stocks from recommended sectors
2. Ensure allocations sum to 1.0
3. Set appropriate risk parameters based on risk tolerance
4. Define clear entry and exit conditions
5. Configure technical indicators (RSI, MACD, SMA as appropriate)
6. Provide clear rationale for the strategy

Focus on creating a balanced, executable strategy that aligns with the user's profile.
"""

            messages = [
                self.llm_service.create_message("system", self.system_prompt),
                self.llm_service.create_message("user", strategy_prompt),
            ]

            # Generate strategy with structured output
            strategy = await self.llm_service.invoke_structured(messages=messages, schema=GeneratedStrategy)

            logger.info(
                "strategy_generated",
                name=strategy.name,
                assets=len(strategy.asset_allocation),
                risk_level=risk_tolerance,
            )

            # Create draft configuration
            strategy_draft: StrategyConfigDraft = {
                "name": strategy.name,
                "description": strategy.description,
                "asset_allocation": strategy.asset_allocation,
                "max_position_size": strategy.max_position_size,
                "max_drawdown": strategy.max_drawdown,
                "stop_loss_pct": strategy.stop_loss_pct,
                "entry_conditions": strategy.entry_conditions,
                "exit_conditions": strategy.exit_conditions,
            }

            # Create response message for user
            response_text = f"""**Strategy Generated: {strategy.name}**

**Description:**
{strategy.description}

**Asset Allocation:**
{chr(10).join(f"• {symbol}: {weight * 100:.1f}%" for symbol, weight in strategy.asset_allocation.items())}

**Risk Parameters:**
• Max Position Size: {strategy.max_position_size * 100:.1f}%
• Stop Loss: {strategy.stop_loss_pct * 100:.1f}%
• Take Profit: {strategy.take_profit_pct * 100:.1f}%
• Max Drawdown: {strategy.max_drawdown * 100:.1f}%

**Entry Conditions:**
{chr(10).join(f"• {cond}" for cond in strategy.entry_conditions)}

**Exit Conditions:**
{chr(10).join(f"• {cond}" for cond in strategy.exit_conditions)}

**Technical Indicators:**
{chr(10).join(f"• {name}: {params}" for name, params in strategy.indicators.items())}

**Rationale:**
{strategy.rationale}

---
Would you like to approve this strategy? (Yes/No)
"""

            response = AIMessage(content=response_text)

            # Update state
            updates = {
                "strategy_draft": strategy_draft,
                "messages": [response],
                "current_agent": "strategy_architect",
                "approval_status": "pending",
            }

            # Update agent history
            agent_history = state.get("agent_history", []) or []
            updates["agent_history"] = agent_history + ["strategy_architect"]

            return {**state, **updates}

        except Exception as e:
            logger.error("strategy_generation_failed", error=str(e))

            errors = state.get("errors", []) or []
            errors.append(f"Strategy generation failed: {str(e)}")

            return {
                **state,
                "errors": errors,
                "current_agent": "strategy_architect",
            }


async def strategy_architect_node(state: ConversationState, *, llm_service: LLMService) -> ConversationState:
    """
    LangGraph node function for strategy generation.

    Args:
        state: Current conversation state
        llm_service: LLM service instance

    Returns:
        Updated conversation state
    """
    agent = StrategyArchitectAgent(llm_service)
    return await agent.process(state)
