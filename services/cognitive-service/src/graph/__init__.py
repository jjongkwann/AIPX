"""LangGraph workflow and state definitions."""

from src.graph.state import ConversationState

__all__ = ["ConversationState", "create_workflow"]


def create_workflow(*args, **kwargs):
    """Lazy import to avoid circular dependency."""
    from src.graph.workflow import create_workflow as _create_workflow
    return _create_workflow(*args, **kwargs)
