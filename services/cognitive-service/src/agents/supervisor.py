"""Supervisor agent for orchestrating the multi-agent workflow."""

import structlog
from pydantic import BaseModel, Field

from src.graph.state import ConversationState
from src.services.llm_service import LLMService

logger = structlog.get_logger()


class SupervisorDecision(BaseModel):
    """Supervisor's decision on next action."""

    next_agent: str = Field(
        ...,
        description="Next agent to call: user_profile, market_analyst, strategy_architect, or end",
    )
    reason: str = Field(..., description="Reasoning for this decision")
    is_complete: bool = Field(default=False, description="Whether the workflow should end")


class SupervisorAgent:
    """
    Supervisor agent for orchestrating the workflow.

    Responsibilities:
    - Decide which agent to call next based on current state
    - Check profile completeness
    - Determine when to request market analysis
    - Trigger strategy generation when ready
    - Handle approval/rejection flow
    - Manage retry logic
    """

    def __init__(self, llm_service: LLMService):
        """Initialize supervisor agent."""
        self.llm_service = llm_service
        self.system_prompt = """You are a workflow supervisor managing an investment strategy generation system.

Your job is to decide which agent should act next based on the current state.

Available Agents:
1. **user_profile**: Extracts user investment profile (risk tolerance, capital, sectors, horizon)
2. **market_analyst**: Analyzes market conditions and recommends sectors
3. **strategy_architect**: Generates complete trading strategy
4. **end**: Complete the workflow

Decision Logic:
1. If user profile is incomplete → call "user_profile"
2. If profile complete but no market analysis → call "market_analyst"
3. If profile and market analysis complete but no strategy → call "strategy_architect"
4. If strategy generated and approved → call "end"
5. If strategy rejected → call "strategy_architect" (regenerate)
6. If max retries exceeded → call "end"

Profile Completeness Requirements:
- Must have: risk_tolerance, capital, investment_horizon
- Optional but recommended: preferred_sectors

Always provide clear reasoning for your decision.
"""

    def check_profile_complete(self, state: ConversationState) -> tuple[bool, list[str]]:
        """
        Check if user profile is complete.

        Args:
            state: Current conversation state

        Returns:
            Tuple of (is_complete, missing_fields)
        """
        required_fields = ["risk_tolerance", "capital", "investment_horizon"]
        missing = []

        for field in required_fields:
            if not state.get(field):
                missing.append(field)

        return len(missing) == 0, missing

    async def process(self, state: ConversationState) -> ConversationState:
        """
        Decide next agent to call.

        Args:
            state: Current conversation state

        Returns:
            Updated conversation state with next_agent set
        """
        try:
            # Check current state
            profile_complete, missing_fields = self.check_profile_complete(state)
            has_market_analysis = bool(state.get("market_sentiment"))
            has_strategy = bool(state.get("strategy_draft"))
            approval_status = state.get("approval_status", "pending")
            retry_count = state.get("retry_count", 0)
            max_retries = 3

            # Build state summary for LLM
            state_summary = f"""Current Workflow State:

**User Profile:**
- Complete: {profile_complete}
- Missing Fields: {missing_fields if missing_fields else "None"}
- Risk Tolerance: {state.get("risk_tolerance", "Not set")}
- Capital: ${state.get("capital", "Not set")}
- Investment Horizon: {state.get("investment_horizon", "Not set")}
- Preferred Sectors: {state.get("preferred_sectors", "Not set")}

**Market Analysis:**
- Completed: {has_market_analysis}
- Sentiment: {state.get("market_sentiment", "Not analyzed")}

**Strategy:**
- Generated: {has_strategy}
- Approval Status: {approval_status}

**Workflow:**
- Retry Count: {retry_count}/{max_retries}
- Agent History: {state.get("agent_history", [])}
- Errors: {state.get("errors", [])}

Based on this state, what should happen next?
"""

            # Special cases (rule-based decisions for efficiency)
            if retry_count >= max_retries:
                logger.warning("max_retries_exceeded", retry_count=retry_count)
                return {
                    **state,
                    "next_agent": "end",
                    "is_complete": True,
                    "current_agent": "supervisor",
                }

            if has_strategy and approval_status == "approved":
                logger.info("strategy_approved_completing_workflow")
                return {
                    **state,
                    "next_agent": "end",
                    "is_complete": True,
                    "current_agent": "supervisor",
                }

            if not profile_complete:
                logger.info("profile_incomplete_calling_user_profile", missing=missing_fields)
                return {
                    **state,
                    "next_agent": "user_profile",
                    "is_complete": False,
                    "current_agent": "supervisor",
                }

            # For more complex decisions, use LLM
            messages = [
                self.llm_service.create_message("system", self.system_prompt),
                self.llm_service.create_message("user", state_summary),
            ]

            decision = await self.llm_service.invoke_structured(messages=messages, schema=SupervisorDecision)

            logger.info(
                "supervisor_decision",
                next_agent=decision.next_agent,
                reason=decision.reason,
                is_complete=decision.is_complete,
            )

            # Update state
            updates = {
                "next_agent": decision.next_agent,
                "is_complete": decision.is_complete,
                "current_agent": "supervisor",
            }

            # Update agent history
            agent_history = state.get("agent_history", []) or []
            updates["agent_history"] = agent_history + ["supervisor"]

            return {**state, **updates}

        except Exception as e:
            logger.error("supervisor_decision_failed", error=str(e))

            # Default to ending on supervisor error
            errors = state.get("errors", []) or []
            errors.append(f"Supervisor failed: {str(e)}")

            return {
                **state,
                "next_agent": "end",
                "is_complete": True,
                "errors": errors,
                "current_agent": "supervisor",
            }


async def supervisor_node(state: ConversationState, *, llm_service: LLMService) -> ConversationState:
    """
    LangGraph node function for supervisor.

    Args:
        state: Current conversation state
        llm_service: LLM service instance

    Returns:
        Updated conversation state with routing decision
    """
    agent = SupervisorAgent(llm_service)
    return await agent.process(state)
