"""Chat message schema definitions."""

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, Field


class ChatMessage(BaseModel):
    """Single chat message."""

    role: Literal["user", "assistant", "system"]
    content: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)


class ChatRequest(BaseModel):
    """Chat request from user."""

    message: str = Field(..., min_length=1, max_length=5000)
    session_id: UUID | None = None


class ChatResponse(BaseModel):
    """Chat response to user."""

    message: str
    session_id: UUID
    agent: str | None = Field(None, description="Name of agent that generated response")
    profile_completeness: float = Field(default=0.0, ge=0.0, le=1.0, description="User profile completeness score")
    missing_fields: list[str] = Field(default_factory=list, description="Missing profile fields")
    strategy_ready: bool = Field(default=False, description="Whether strategy is ready for generation")


class SessionCreate(BaseModel):
    """Create new chat session."""

    user_id: UUID


class SessionResponse(BaseModel):
    """Chat session information."""

    session_id: UUID
    user_id: UUID
    started_at: datetime
    ended_at: datetime | None = None
    message_count: int = 0
    status: Literal["active", "ended"] = "active"


class ChatHistory(BaseModel):
    """Chat history for a session."""

    session_id: UUID
    messages: list[ChatMessage]
    total_messages: int
