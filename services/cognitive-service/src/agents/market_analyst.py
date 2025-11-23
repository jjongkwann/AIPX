"""Market analysis agent."""

import structlog
from langchain_core.messages import AIMessage
from pydantic import BaseModel, Field

from src.graph.state import ConversationState
from src.services.llm_service import LLMService

logger = structlog.get_logger()


class MarketAnalysis(BaseModel):
    """Market analysis output."""

    sentiment: str = Field(..., description="bullish, bearish, or neutral")
    recommended_sectors: list[str] = Field(
        default_factory=list, description="Recommended sectors based on market conditions"
    )
    analysis: str = Field(..., description="Detailed market analysis text")
    confidence: float = Field(default=0.0, ge=0.0, le=1.0, description="Analysis confidence score")
    key_points: list[str] = Field(default_factory=list, description="Key analysis points")


class MarketAnalystAgent:
    """
    Agent for analyzing current market conditions.

    Responsibilities:
    - Analyze current market sentiment (bullish/bearish/neutral)
    - Recommend suitable sectors based on market conditions
    - Provide market analysis and trends
    - Consider user's risk tolerance in recommendations
    """

    def __init__(self, llm_service: LLMService):
        """Initialize market analyst agent."""
        self.llm_service = llm_service
        self.system_prompt = """You are an experienced market analyst specializing in equity markets and sector analysis.

Your task is to analyze current market conditions and provide recommendations.

Your analysis should include:
1. Overall market sentiment (bullish, bearish, or neutral)
2. Recommended sectors based on:
   - Current market trends
   - Economic indicators
   - User's risk tolerance
3. Key market drivers and risks
4. Sector-specific opportunities

Guidelines:
- Provide balanced, data-driven analysis
- Consider both opportunities and risks
- Tailor recommendations to user's risk profile
- Be clear and concise
- Cite general market trends (you can use general knowledge, no real-time data for now)

Risk Tolerance Mapping:
- Conservative: Focus on defensive sectors (Utilities, Consumer Staples, Healthcare)
- Moderate: Balanced mix of growth and value (Finance, Technology, Healthcare)
- Aggressive: Growth sectors (Technology, Biotech, Clean Energy)
"""

    async def process(self, state: ConversationState) -> ConversationState:
        """
        Analyze market conditions and update state.

        Args:
            state: Current conversation state

        Returns:
            Updated conversation state
        """
        try:
            # Get user profile for context
            risk_tolerance = state.get("risk_tolerance", "moderate")
            preferred_sectors = state.get("preferred_sectors", [])
            investment_horizon = state.get("investment_horizon", "medium")

            # Create analysis prompt
            analysis_prompt = f"""Analyze current market conditions for an investor with the following profile:

Risk Tolerance: {risk_tolerance}
Preferred Sectors: {preferred_sectors if preferred_sectors else "None specified"}
Investment Horizon: {investment_horizon}

Provide:
1. Overall market sentiment
2. Recommended sectors that align with this profile
3. Brief market analysis (2-3 paragraphs)
4. Key points to consider

Note: Use general market knowledge. This is for strategy generation purposes.
"""

            messages = [
                self.llm_service.create_message("system", self.system_prompt),
                self.llm_service.create_message("user", analysis_prompt),
            ]

            # Get structured analysis
            analysis = await self.llm_service.invoke_structured(messages=messages, schema=MarketAnalysis)

            logger.info(
                "market_analysis_complete",
                sentiment=analysis.sentiment,
                sectors=analysis.recommended_sectors,
                confidence=analysis.confidence,
            )

            # Create response message
            response_text = f"""**Market Analysis**

**Sentiment:** {analysis.sentiment.upper()}

**Recommended Sectors:**
{chr(10).join(f"• {sector}" for sector in analysis.recommended_sectors)}

**Analysis:**
{analysis.analysis}

**Key Points:**
{chr(10).join(f"• {point}" for point in analysis.key_points)}
"""

            response = AIMessage(content=response_text)

            # Update state
            updates = {
                "market_sentiment": analysis.sentiment,
                "recommended_sectors": analysis.recommended_sectors,
                "market_analysis": analysis.analysis,
                "messages": [response],
                "current_agent": "market_analyst",
            }

            # Update agent history
            agent_history = state.get("agent_history", []) or []
            updates["agent_history"] = agent_history + ["market_analyst"]

            return {**state, **updates}

        except Exception as e:
            logger.error("market_analysis_failed", error=str(e))

            errors = state.get("errors", []) or []
            errors.append(f"Market analysis failed: {str(e)}")

            return {
                **state,
                "errors": errors,
                "current_agent": "market_analyst",
            }


async def market_analyst_node(state: ConversationState, *, llm_service: LLMService) -> ConversationState:
    """
    LangGraph node function for market analysis.

    Args:
        state: Current conversation state
        llm_service: LLM service instance

    Returns:
        Updated conversation state
    """
    agent = MarketAnalystAgent(llm_service)
    return await agent.process(state)
