"""Tests for individual agents."""

from unittest.mock import AsyncMock, MagicMock

import pytest
from langchain_core.messages import HumanMessage

from src.agents.supervisor import SupervisorAgent
from src.agents.user_profile_agent import UserProfileAgent
from src.graph.state import ConversationState
from src.services.llm_service import LLMService


@pytest.fixture
def mock_llm_service():
    """Create mock LLM service."""
    service = MagicMock(spec=LLMService)
    service.invoke_structured = AsyncMock()
    service.create_message = MagicMock()
    return service


class TestSupervisorAgent:
    """Test Supervisor agent."""

    @pytest.mark.asyncio
    async def test_profile_incomplete_routes_to_user_profile(self, mock_llm_service):
        """Test supervisor routes to user_profile when profile incomplete."""
        agent = SupervisorAgent(mock_llm_service)

        state: ConversationState = {
            "messages": [HumanMessage(content="Test")],
            "user_id": "test",
            "session_id": "test",
        }

        result = await agent.process(state)

        assert result.get("next_agent") == "user_profile"
        assert result.get("is_complete") is False

    @pytest.mark.asyncio
    async def test_approved_strategy_completes_workflow(self, mock_llm_service):
        """Test supervisor completes workflow when strategy approved."""
        agent = SupervisorAgent(mock_llm_service)

        state: ConversationState = {
            "messages": [HumanMessage(content="Test")],
            "user_id": "test",
            "session_id": "test",
            "risk_tolerance": "moderate",
            "capital": 100000,
            "investment_horizon": "long",
            "strategy_draft": {"name": "Test Strategy"},
            "approval_status": "approved",
        }

        result = await agent.process(state)

        assert result.get("next_agent") == "end"
        assert result.get("is_complete") is True

    @pytest.mark.asyncio
    async def test_max_retries_ends_workflow(self, mock_llm_service):
        """Test supervisor ends workflow when max retries exceeded."""
        agent = SupervisorAgent(mock_llm_service)

        state: ConversationState = {
            "messages": [HumanMessage(content="Test")],
            "user_id": "test",
            "session_id": "test",
            "retry_count": 5,
        }

        result = await agent.process(state)

        assert result.get("next_agent") == "end"
        assert result.get("is_complete") is True


class TestUserProfileAgent:
    """Test UserProfile agent."""

    @pytest.mark.asyncio
    async def test_extracts_profile_information(self, mock_llm_service):
        """Test user profile extraction."""
        # Mock LLM response
        mock_llm_service.invoke_structured.return_value = MagicMock(
            risk_tolerance="moderate",
            capital=100000,
            preferred_sectors=["Technology"],
            investment_horizon="long",
            confidence=0.9,
            missing_fields=[],
            follow_up_question=None,
        )

        agent = UserProfileAgent(mock_llm_service)

        state: ConversationState = {
            "messages": [HumanMessage(content="I want to invest $100k in tech stocks for long term")],
            "user_id": "test",
            "session_id": "test",
        }

        result = await agent.process(state)

        assert result.get("risk_tolerance") == "moderate"
        assert result.get("capital") == 100000
        assert "Technology" in result.get("preferred_sectors", [])
        assert result.get("investment_horizon") == "long"

    @pytest.mark.asyncio
    async def test_generates_follow_up_question(self, mock_llm_service):
        """Test follow-up question generation for missing fields."""
        # Mock LLM response with missing fields
        mock_llm_service.invoke_structured.return_value = MagicMock(
            risk_tolerance=None,
            capital=100000,
            preferred_sectors=[],
            investment_horizon=None,
            confidence=0.5,
            missing_fields=["risk_tolerance", "investment_horizon"],
            follow_up_question="What is your risk tolerance and investment timeline?",
        )

        agent = UserProfileAgent(mock_llm_service)

        state: ConversationState = {
            "messages": [HumanMessage(content="I want to invest $100k")],
            "user_id": "test",
            "session_id": "test",
        }

        result = await agent.process(state)

        assert result.get("capital") == 100000
        # Should have follow-up message
        messages = result.get("messages", [])
        assert len(messages) > 0
