"""FastAPI application for Cognitive Service."""

import logging
from contextlib import asynccontextmanager

import structlog
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from src.config import get_settings
from src.graph.workflow import WorkflowManager
from src.routers import chat, strategy
from src.services import DatabaseService, KafkaService, LLMService

# Configure structured logging
structlog.configure(
    processors=[
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
    context_class=dict,
    logger_factory=structlog.PrintLoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()

# Global service instances
settings = get_settings()
llm_service: LLMService | None = None
db_service: DatabaseService | None = None
kafka_service: KafkaService | None = None
workflow_manager: WorkflowManager | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager for startup and shutdown."""
    # Startup
    logger.info("cognitive_service_starting", version="0.1.0")

    global llm_service, db_service, kafka_service, workflow_manager

    try:
        # Initialize services
        llm_service = LLMService(settings)
        await llm_service.initialize()

        db_service = DatabaseService(settings)
        await db_service.initialize()

        kafka_service = KafkaService(settings)
        await kafka_service.initialize()

        # Initialize workflow manager
        workflow_manager = WorkflowManager(llm_service)

        logger.info("all_services_initialized")

    except Exception as e:
        logger.error("service_initialization_failed", error=str(e))
        raise

    yield

    # Shutdown
    logger.info("cognitive_service_shutting_down")

    try:
        if kafka_service:
            await kafka_service.cleanup()
        if db_service:
            await db_service.cleanup()
        if llm_service:
            await llm_service.cleanup()

        logger.info("all_services_cleaned_up")

    except Exception as e:
        logger.error("cleanup_failed", error=str(e))


# Create FastAPI application
app = FastAPI(
    title="Cognitive Service",
    description="AI-powered investment strategy generation service",
    version="0.1.0",
    lifespan=lifespan,
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(chat.router, prefix="/api/v1", tags=["chat"])
app.include_router(strategy.router, prefix="/api/v1", tags=["strategy"])

# Metrics endpoint
if settings.enable_metrics:
    metrics_app = make_asgi_app()
    app.mount("/metrics", metrics_app)


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "cognitive-service",
        "version": "0.1.0",
    }


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "cognitive-service",
        "version": "0.1.0",
        "description": "AI-powered investment strategy generation",
    }


def get_llm_service() -> LLMService:
    """Dependency for LLM service."""
    if llm_service is None:
        raise RuntimeError("LLM service not initialized")
    return llm_service


def get_db_service() -> DatabaseService:
    """Dependency for database service."""
    if db_service is None:
        raise RuntimeError("Database service not initialized")
    return db_service


def get_kafka_service() -> KafkaService:
    """Dependency for Kafka service."""
    if kafka_service is None:
        raise RuntimeError("Kafka service not initialized")
    return kafka_service


def get_workflow_manager() -> WorkflowManager:
    """Dependency for workflow manager."""
    if workflow_manager is None:
        raise RuntimeError("Workflow manager not initialized")
    return workflow_manager


# Export dependencies
__all__ = [
    "app",
    "get_llm_service",
    "get_db_service",
    "get_kafka_service",
    "get_workflow_manager",
]
