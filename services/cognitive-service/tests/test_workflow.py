"""Tests for LangGraph workflow."""

from unittest.mock import AsyncMock, MagicMock

import pytest
from langchain_core.messages import HumanMessage

from src.graph.state import ConversationState
from src.graph.workflow import WorkflowManager, create_workflow
from src.services.llm_service import LLMService


@pytest.fixture
def mock_llm_service():
    """Create mock LLM service."""
    service = MagicMock(spec=LLMService)
    service.invoke = AsyncMock()
    service.invoke_structured = AsyncMock()
    return service


@pytest.fixture
def workflow_manager(mock_llm_service):
    """Create workflow manager with mock LLM service."""
    return WorkflowManager(mock_llm_service)


class TestWorkflow:
    """Test LangGraph workflow."""

    @pytest.mark.asyncio
    async def test_create_workflow(self, mock_llm_service):
        """Test workflow creation."""
        workflow = create_workflow(mock_llm_service, use_memory=False)
        assert workflow is not None

    @pytest.mark.asyncio
    async def test_workflow_execution_with_complete_profile(self, workflow_manager, mock_llm_service):
        """Test workflow with complete user profile."""
        # Mock supervisor decision to end
        mock_llm_service.invoke_structured.return_value = MagicMock(
            next_agent="end", reason="Profile complete", is_complete=True
        )

        initial_state: ConversationState = {
            "messages": [HumanMessage(content="I want to invest")],
            "user_id": "test-user",
            "session_id": "test-session",
            "risk_tolerance": "moderate",
            "capital": 100000,
            "investment_horizon": "long",
        }

        config = {"configurable": {"thread_id": "test-thread"}}

        final_state = await workflow_manager.execute(initial_state, config)

        assert final_state is not None
        assert final_state.get("user_id") == "test-user"

    @pytest.mark.asyncio
    async def test_workflow_state_transitions(self, mock_llm_service):
        """Test workflow state transitions."""
        workflow = create_workflow(mock_llm_service, use_memory=False)

        initial_state: ConversationState = {
            "messages": [HumanMessage(content="Test message")],
            "user_id": "test-user",
            "session_id": "test-session",
        }

        # Mock responses
        mock_llm_service.invoke_structured.side_effect = [
            MagicMock(
                next_agent="user_profile",
                reason="Need profile",
                is_complete=False,
            ),
            MagicMock(
                next_agent="end",
                reason="Complete",
                is_complete=True,
            ),
        ]

        config = {"configurable": {"thread_id": "test-thread"}}

        # Execute workflow
        final_state = await workflow.ainvoke(initial_state, config)

        assert final_state is not None


class TestWorkflowManager:
    """Test WorkflowManager class."""

    @pytest.mark.asyncio
    async def test_workflow_manager_initialization(self, mock_llm_service):
        """Test workflow manager initialization."""
        manager = WorkflowManager(mock_llm_service)
        assert manager.llm_service == mock_llm_service
        assert manager.workflow is not None

    @pytest.mark.asyncio
    async def test_workflow_manager_execute(self, workflow_manager, mock_llm_service):
        """Test workflow manager execute method."""
        # Use side_effect to return proper structured objects for each call
        # First call: user_profile agent extracts profile
        # Second call: supervisor decides to end
        from src.agents.supervisor import SupervisorDecision
        from src.agents.user_profile_agent import ExtractedProfile

        profile_response = ExtractedProfile(
            risk_tolerance="moderate",
            capital=100000,
            investment_horizon="long",
            confidence=0.9,
            missing_fields=[],
        )
        supervisor_response = SupervisorDecision(
            next_agent="end",
            reason="Profile complete",
            is_complete=True,
        )

        mock_llm_service.invoke_structured.side_effect = [profile_response, supervisor_response]

        initial_state: ConversationState = {
            "messages": [HumanMessage(content="Test")],
            "user_id": "test",
            "session_id": "test",
        }

        config = {"configurable": {"thread_id": "test"}}

        result = await workflow_manager.execute(initial_state, config)
        assert result is not None
