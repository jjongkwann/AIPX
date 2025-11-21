"""Chat endpoints for conversational interface."""

import json
from uuid import UUID, uuid4

import structlog
from fastapi import APIRouter, Depends, HTTPException, WebSocket, WebSocketDisconnect
from langchain_core.messages import HumanMessage

from src.graph.state import ConversationState
from src.graph.workflow import WorkflowManager
from src.main import get_db_service, get_workflow_manager
from src.schemas.message import ChatRequest, ChatResponse, SessionCreate, SessionResponse
from src.services import DatabaseService

logger = structlog.get_logger()

router = APIRouter()


@router.post("/chat", response_model=ChatResponse)
async def chat(
    request: ChatRequest,
    db: DatabaseService = Depends(get_db_service),
    workflow: WorkflowManager = Depends(get_workflow_manager),
):
    """
    HTTP chat endpoint.

    Args:
        request: Chat request with message
        db: Database service
        workflow: Workflow manager

    Returns:
        Chat response with assistant message
    """
    try:
        # Get or create session
        if request.session_id:
            session = await db.get_session(request.session_id)
            if not session:
                raise HTTPException(status_code=404, detail="Session not found")
            session_id = request.session_id
            user_id = UUID(session["user_id"])
        else:
            # Create new session (using dummy user_id for now)
            session_id = uuid4()
            user_id = uuid4()
            await db.create_session(user_id, session_id)

        # Create initial state
        initial_state: ConversationState = {
            "messages": [HumanMessage(content=request.message)],
            "user_id": str(user_id),
            "session_id": str(session_id),
        }

        # Execute workflow
        config = {"configurable": {"thread_id": str(session_id)}}
        final_state = await workflow.execute(initial_state, config)

        # Get last assistant message
        assistant_messages = [msg for msg in final_state.get("messages", []) if msg.type == "ai"]
        response_message = assistant_messages[-1].content if assistant_messages else "No response"

        # Calculate profile completeness
        required_fields = ["risk_tolerance", "capital", "investment_horizon"]
        filled = sum(1 for f in required_fields if final_state.get(f))
        completeness = filled / len(required_fields)

        missing = [f for f in required_fields if not final_state.get(f)]

        # Increment message count
        await db.increment_message_count(session_id)

        return ChatResponse(
            message=response_message,
            session_id=session_id,
            agent=final_state.get("current_agent"),
            profile_completeness=completeness,
            missing_fields=missing,
            strategy_ready=bool(final_state.get("strategy_draft")),
        )

    except Exception as e:
        logger.error("chat_request_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@router.websocket("/ws/chat/{user_id}")
async def websocket_chat(
    websocket: WebSocket,
    user_id: UUID,
    db: DatabaseService = Depends(get_db_service),
    workflow: WorkflowManager = Depends(get_workflow_manager),
):
    """
    WebSocket chat endpoint for real-time conversation.

    Args:
        websocket: WebSocket connection
        user_id: User ID
        db: Database service
        workflow: Workflow manager
    """
    await websocket.accept()

    # Create new session
    session_id = uuid4()
    await db.create_session(user_id, session_id)

    logger.info("websocket_connected", user_id=str(user_id), session_id=str(session_id))

    try:
        while True:
            # Receive message
            data = await websocket.receive_text()
            message_data = json.loads(data)
            user_message = message_data.get("message", "")

            if not user_message:
                continue

            # Create state with user message
            initial_state: ConversationState = {
                "messages": [HumanMessage(content=user_message)],
                "user_id": str(user_id),
                "session_id": str(session_id),
            }

            # Stream workflow execution
            config = {"configurable": {"thread_id": str(session_id)}}

            async for state_update in workflow.stream(initial_state, config):
                # Extract messages from state update
                for node_name, node_state in state_update.items():
                    if "messages" in node_state:
                        for msg in node_state["messages"]:
                            if msg.type == "ai":
                                # Send assistant message to client
                                await websocket.send_json(
                                    {
                                        "type": "message",
                                        "agent": node_name,
                                        "content": msg.content,
                                    }
                                )

            # Send completion signal
            await websocket.send_json({"type": "complete"})

            # Increment message count
            await db.increment_message_count(session_id)

    except WebSocketDisconnect:
        logger.info("websocket_disconnected", session_id=str(session_id))
        await db.end_session(session_id)

    except Exception as e:
        logger.error("websocket_error", error=str(e), session_id=str(session_id))
        await websocket.close()


@router.post("/sessions", response_model=SessionResponse)
async def create_session(request: SessionCreate, db: DatabaseService = Depends(get_db_service)):
    """
    Create a new chat session.

    Args:
        request: Session create request
        db: Database service

    Returns:
        Session information
    """
    try:
        session_id = uuid4()
        session = await db.create_session(request.user_id, session_id)

        return SessionResponse(
            session_id=UUID(session["session_id"]),
            user_id=UUID(session["user_id"]),
            started_at=session["started_at"],
            message_count=session["message_count"],
            status="active",
        )

    except Exception as e:
        logger.error("session_creation_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/sessions/{session_id}", response_model=SessionResponse)
async def get_session(session_id: UUID, db: DatabaseService = Depends(get_db_service)):
    """
    Get session information.

    Args:
        session_id: Session ID
        db: Database service

    Returns:
        Session information
    """
    session = await db.get_session(session_id)

    if not session:
        raise HTTPException(status_code=404, detail="Session not found")

    return SessionResponse(
        session_id=UUID(session["session_id"]),
        user_id=UUID(session["user_id"]),
        started_at=session["started_at"],
        ended_at=session.get("ended_at"),
        message_count=session["message_count"],
        status="ended" if session.get("ended_at") else "active",
    )


@router.delete("/sessions/{session_id}")
async def end_session(session_id: UUID, db: DatabaseService = Depends(get_db_service)):
    """
    End a chat session.

    Args:
        session_id: Session ID
        db: Database service

    Returns:
        Success message
    """
    try:
        await db.end_session(session_id)
        return {"status": "success", "message": "Session ended"}

    except Exception as e:
        logger.error("session_end_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
