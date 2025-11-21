"""LangGraph workflow for cognitive service."""

from functools import partial
from typing import Literal

import structlog
from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, StateGraph

from src.agents import market_analyst_node, strategy_architect_node, supervisor_node, user_profile_node
from src.graph.state import ConversationState
from src.services.llm_service import LLMService

logger = structlog.get_logger()


def route_next_agent(
    state: ConversationState,
) -> Literal["user_profile", "market_analyst", "strategy_architect", "end"]:
    """
    Route to next agent based on supervisor's decision.

    Args:
        state: Current conversation state

    Returns:
        Name of next agent to call
    """
    next_agent = state.get("next_agent", "end")

    logger.debug("routing_to_agent", next_agent=next_agent)

    # Map next_agent to node names
    if next_agent == "user_profile":
        return "user_profile"
    elif next_agent == "market_analyst":
        return "market_analyst"
    elif next_agent == "strategy_architect":
        return "strategy_architect"
    else:
        return "end"


def create_workflow(llm_service: LLMService, use_memory: bool = True) -> StateGraph:
    """
    Create LangGraph workflow with agent nodes.

    Args:
        llm_service: LLM service instance
        use_memory: Whether to use memory checkpointer for persistence

    Returns:
        Compiled StateGraph workflow
    """
    # Create workflow graph
    workflow = StateGraph(ConversationState)

    # Create partial functions with llm_service bound
    user_profile_fn = partial(user_profile_node, llm_service=llm_service)
    market_analyst_fn = partial(market_analyst_node, llm_service=llm_service)
    strategy_architect_fn = partial(strategy_architect_node, llm_service=llm_service)
    supervisor_fn = partial(supervisor_node, llm_service=llm_service)

    # Add agent nodes
    workflow.add_node("user_profile", user_profile_fn)
    workflow.add_node("market_analyst", market_analyst_fn)
    workflow.add_node("strategy_architect", strategy_architect_fn)
    workflow.add_node("supervisor", supervisor_fn)

    # Set entry point
    workflow.set_entry_point("supervisor")

    # Add conditional edges from supervisor to agents
    workflow.add_conditional_edges(
        "supervisor",
        route_next_agent,
        {
            "user_profile": "user_profile",
            "market_analyst": "market_analyst",
            "strategy_architect": "strategy_architect",
            "end": END,
        },
    )

    # All agents return to supervisor for next decision
    workflow.add_edge("user_profile", "supervisor")
    workflow.add_edge("market_analyst", "supervisor")
    workflow.add_edge("strategy_architect", "supervisor")

    # Compile workflow
    if use_memory:
        memory = MemorySaver()
        compiled = workflow.compile(checkpointer=memory)
        logger.info("workflow_compiled_with_memory")
    else:
        compiled = workflow.compile()
        logger.info("workflow_compiled_without_memory")

    return compiled


class WorkflowManager:
    """Manager for executing LangGraph workflows."""

    def __init__(self, llm_service: LLMService):
        """Initialize workflow manager."""
        self.llm_service = llm_service
        self.workflow = create_workflow(llm_service, use_memory=True)

    async def execute(self, initial_state: ConversationState, config: dict | None = None) -> ConversationState:
        """
        Execute workflow from initial state.

        Args:
            initial_state: Initial conversation state
            config: Optional configuration (for thread_id, etc.)

        Returns:
            Final conversation state
        """
        try:
            logger.info(
                "workflow_execution_started",
                user_id=initial_state.get("user_id"),
                session_id=initial_state.get("session_id"),
            )

            # Execute workflow
            final_state = await self.workflow.ainvoke(initial_state, config=config)

            logger.info(
                "workflow_execution_completed",
                is_complete=final_state.get("is_complete"),
                agent_history=final_state.get("agent_history"),
            )

            return final_state

        except Exception as e:
            logger.error("workflow_execution_failed", error=str(e))
            raise

    async def stream(self, initial_state: ConversationState, config: dict | None = None):
        """
        Stream workflow execution step by step.

        Args:
            initial_state: Initial conversation state
            config: Optional configuration

        Yields:
            State updates from each step
        """
        try:
            logger.info(
                "workflow_streaming_started",
                user_id=initial_state.get("user_id"),
                session_id=initial_state.get("session_id"),
            )

            async for state in self.workflow.astream(initial_state, config=config):
                logger.debug("workflow_step", state_keys=list(state.keys()))
                yield state

        except Exception as e:
            logger.error("workflow_streaming_failed", error=str(e))
            raise

    async def get_state(self, config: dict) -> ConversationState | None:
        """
        Get current state for a thread.

        Args:
            config: Configuration with thread_id

        Returns:
            Current state or None
        """
        try:
            state = await self.workflow.aget_state(config)
            return state.values if state else None

        except Exception as e:
            logger.error("get_state_failed", error=str(e))
            return None
