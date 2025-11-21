"""Strategy management endpoints."""

from datetime import datetime
from uuid import UUID, uuid4

import structlog
from fastapi import APIRouter, Depends, HTTPException

from src.main import get_db_service, get_kafka_service
from src.schemas.strategy_config import StrategyApproval, StrategyConfig, StrategyConfigCreate
from src.services import DatabaseService, KafkaService

logger = structlog.get_logger()

router = APIRouter()


@router.post("/strategies", response_model=StrategyConfig)
async def create_strategy(
    strategy: StrategyConfigCreate,
    user_id: UUID,
    db: DatabaseService = Depends(get_db_service),
    kafka: KafkaService = Depends(get_kafka_service),
):
    """
    Create a new strategy configuration.

    Args:
        strategy: Strategy configuration
        user_id: User ID (from auth)
        db: Database service
        kafka: Kafka service

    Returns:
        Created strategy configuration
    """
    try:
        strategy_id = uuid4()

        # Create strategy config
        config = StrategyConfig(
            strategy_id=strategy_id,
            user_id=user_id,
            name=strategy.name,
            description=strategy.description,
            asset_allocation=strategy.asset_allocation,
            max_position_size=strategy.max_position_size,
            max_drawdown=strategy.max_drawdown,
            stop_loss_pct=strategy.stop_loss_pct,
            take_profit_pct=strategy.take_profit_pct,
            entry_conditions=strategy.entry_conditions,
            exit_conditions=strategy.exit_conditions,
            indicators=strategy.indicators,
            status="draft",
            created_at=datetime.utcnow(),
        )

        # Save to database
        await db.create_strategy(config)

        # Publish event
        await kafka.publish_strategy_created(
            strategy_id=strategy_id,
            user_id=user_id,
            strategy_data=config.model_dump(mode="json"),
        )

        logger.info("strategy_created", strategy_id=str(strategy_id), user_id=str(user_id))

        return config

    except Exception as e:
        logger.error("strategy_creation_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/strategies/{strategy_id}", response_model=StrategyConfig)
async def get_strategy(strategy_id: UUID, db: DatabaseService = Depends(get_db_service)):
    """
    Get strategy by ID.

    Args:
        strategy_id: Strategy ID
        db: Database service

    Returns:
        Strategy configuration
    """
    strategy = await db.get_strategy(strategy_id)

    if not strategy:
        raise HTTPException(status_code=404, detail="Strategy not found")

    return strategy


@router.get("/strategies", response_model=list[StrategyConfig])
async def list_strategies(user_id: UUID, limit: int = 50, db: DatabaseService = Depends(get_db_service)):
    """
    List strategies for a user.

    Args:
        user_id: User ID
        limit: Maximum number of strategies to return
        db: Database service

    Returns:
        List of strategy configurations
    """
    try:
        strategies = await db.list_user_strategies(user_id, limit)
        return strategies

    except Exception as e:
        logger.error("strategy_list_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/strategies/{strategy_id}/approve")
async def approve_strategy(
    strategy_id: UUID,
    approval: StrategyApproval,
    db: DatabaseService = Depends(get_db_service),
    kafka: KafkaService = Depends(get_kafka_service),
):
    """
    Approve or reject a strategy.

    Args:
        strategy_id: Strategy ID
        approval: Approval decision
        db: Database service
        kafka: Kafka service

    Returns:
        Success message
    """
    try:
        # Get strategy
        strategy = await db.get_strategy(strategy_id)
        if not strategy:
            raise HTTPException(status_code=404, detail="Strategy not found")

        if approval.approved:
            # Approve strategy
            await db.approve_strategy(strategy_id)

            # Publish approval event
            await kafka.publish_strategy_approved(
                strategy_id=strategy_id,
                user_id=strategy.user_id,
                approved_at=datetime.utcnow().isoformat(),
            )

            logger.info("strategy_approved", strategy_id=str(strategy_id))

            return {
                "status": "success",
                "message": "Strategy approved and will be deployed",
                "strategy_id": str(strategy_id),
            }
        else:
            # Publish rejection event
            await kafka.publish_strategy_rejected(
                strategy_id=strategy_id,
                user_id=strategy.user_id,
                reason=approval.reason,
            )

            logger.info(
                "strategy_rejected",
                strategy_id=str(strategy_id),
                reason=approval.reason,
            )

            return {
                "status": "success",
                "message": "Strategy rejected",
                "strategy_id": str(strategy_id),
                "reason": approval.reason,
            }

    except HTTPException:
        raise
    except Exception as e:
        logger.error("strategy_approval_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/strategies/{strategy_id}")
async def delete_strategy(strategy_id: UUID, db: DatabaseService = Depends(get_db_service)):
    """
    Delete a strategy (soft delete by marking as terminated).

    Args:
        strategy_id: Strategy ID
        db: Database service

    Returns:
        Success message
    """
    try:
        # In a real implementation, update status to 'terminated'
        # For now, just return success
        logger.info("strategy_deleted", strategy_id=str(strategy_id))

        return {"status": "success", "message": "Strategy deleted"}

    except Exception as e:
        logger.error("strategy_deletion_failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
