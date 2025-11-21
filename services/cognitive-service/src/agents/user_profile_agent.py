"""User profile extraction agent."""

import structlog
from langchain_core.messages import AIMessage, HumanMessage
from pydantic import BaseModel, Field

from src.graph.state import ConversationState
from src.services.llm_service import LLMService

logger = structlog.get_logger()


class ExtractedProfile(BaseModel):
    """Extracted user profile information with confidence scores."""

    risk_tolerance: str | None = Field(None, description="conservative, moderate, or aggressive")
    capital: int | None = Field(None, description="Investment capital in USD")
    preferred_sectors: list[str] | None = Field(None, description="Preferred investment sectors")
    investment_horizon: str | None = Field(None, description="short, medium, or long")
    confidence: float = Field(default=0.0, ge=0.0, le=1.0, description="Overall extraction confidence")
    missing_fields: list[str] = Field(default_factory=list, description="Fields that need clarification")
    follow_up_question: str | None = Field(None, description="Question to ask user for missing information")


class UserProfileAgent:
    """
    Agent for extracting user investment profile from conversation.

    Responsibilities:
    - Extract risk tolerance, capital, sectors, and investment horizon
    - Identify missing information
    - Generate follow-up questions for clarification
    - Update conversation state with extracted information
    """

    def __init__(self, llm_service: LLMService):
        """Initialize user profile agent."""
        self.llm_service = llm_service
        self.system_prompt = """You are a financial advisor assistant specializing in understanding client investment profiles.

Your task is to extract the following information from user conversations:
1. Risk Tolerance: conservative, moderate, or aggressive
2. Capital: Investment amount in USD (numeric value)
3. Preferred Sectors: e.g., Technology, Healthcare, Finance, Energy, etc.
4. Investment Horizon: short (< 1 year), medium (1-5 years), or long (> 5 years)

Guidelines:
- Extract information accurately from natural language
- If information is ambiguous, mark confidence as low
- Identify missing fields that need clarification
- Generate a natural follow-up question to gather missing information
- Be conversational and professional

Examples of user input:
- "I have $100,000 to invest in tech stocks for the long term"
  → capital: 100000, sectors: [Technology], horizon: long

- "I'm pretty risk-averse and want to invest $50k"
  → risk_tolerance: conservative, capital: 50000

- "I want aggressive growth with $200k in healthcare and tech"
  → risk_tolerance: aggressive, capital: 200000, sectors: [Healthcare, Technology]
"""

    async def process(self, state: ConversationState) -> ConversationState:
        """
        Process conversation and extract user profile.

        Args:
            state: Current conversation state

        Returns:
            Updated conversation state
        """
        try:
            # Get last few messages for context
            recent_messages = state["messages"][-5:] if len(state["messages"]) > 5 else state["messages"]

            # Build conversation context
            conversation = "\n".join(
                [f"{'User' if isinstance(m, HumanMessage) else 'Assistant'}: {m.content}" for m in recent_messages]
            )

            # Current profile state
            current_profile = {
                "risk_tolerance": state.get("risk_tolerance"),
                "capital": state.get("capital"),
                "preferred_sectors": state.get("preferred_sectors", []),
                "investment_horizon": state.get("investment_horizon"),
            }

            # Create extraction prompt
            extraction_prompt = f"""Based on the following conversation, extract user profile information.

Current Profile State:
{current_profile}

Recent Conversation:
{conversation}

Extract any new or updated information from the conversation. If a field already has a value, only update it if the user explicitly changes it.
"""

            messages = [
                self.llm_service.create_message("system", self.system_prompt),
                self.llm_service.create_message("user", extraction_prompt),
            ]

            # Use structured output
            extracted = await self.llm_service.invoke_structured(messages=messages, schema=ExtractedProfile)

            logger.info(
                "profile_extracted",
                confidence=extracted.confidence,
                missing_fields=extracted.missing_fields,
            )

            # Update state with extracted information
            updates = {}
            if extracted.risk_tolerance:
                updates["risk_tolerance"] = extracted.risk_tolerance
            if extracted.capital is not None:
                updates["capital"] = extracted.capital
            if extracted.preferred_sectors:
                updates["preferred_sectors"] = extracted.preferred_sectors
            if extracted.investment_horizon:
                updates["investment_horizon"] = extracted.investment_horizon

            # Add follow-up question if there are missing fields
            if extracted.missing_fields and extracted.follow_up_question:
                response = AIMessage(content=extracted.follow_up_question)
                updates["messages"] = [response]

            # Update agent tracking
            updates["current_agent"] = "user_profile"
            if "agent_history" not in state or state["agent_history"] is None:
                updates["agent_history"] = ["user_profile"]
            else:
                updates["agent_history"] = state["agent_history"] + ["user_profile"]

            return {**state, **updates}

        except Exception as e:
            logger.error("profile_extraction_failed", error=str(e))

            # Add error to state
            error_msg = f"Failed to extract profile: {str(e)}"
            errors = state.get("errors", []) or []
            errors.append(error_msg)

            return {
                **state,
                "errors": errors,
                "current_agent": "user_profile",
            }


async def user_profile_node(state: ConversationState, *, llm_service: LLMService) -> ConversationState:
    """
    LangGraph node function for user profile extraction.

    Args:
        state: Current conversation state
        llm_service: LLM service instance

    Returns:
        Updated conversation state
    """
    agent = UserProfileAgent(llm_service)
    return await agent.process(state)
