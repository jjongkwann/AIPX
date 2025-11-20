"""Main application entry point."""

import asyncio
import signal
import sys
from pathlib import Path

import structlog
import uvloop

from .config import get_config
from .consumer import StrategyConsumer
from .database import DatabasePool, ExecutionRepository
from .executor import StrategyExecutor
from .grpc_client import OMSClient
from .risk import RiskManager
from .utils.logging import setup_logging

logger = structlog.get_logger()


class StrategyWorkerApp:
    """Main application class."""

    def __init__(self) -> None:
        """Initialize application."""
        self.config = get_config()
        setup_logging(self.config.logging)

        # Core components
        self.db_pool: DatabasePool | None = None
        self.repository: ExecutionRepository | None = None
        self.oms_client: OMSClient | None = None
        self.risk_manager: RiskManager | None = None
        self.executor: StrategyExecutor | None = None
        self.consumer: StrategyConsumer | None = None

        # Shutdown flag
        self._shutdown = False

    async def startup(self) -> None:
        """Initialize all components."""
        logger.info(
            "Starting Strategy Worker",
            version="0.1.0",
            environment=self.config.service.environment,
        )

        try:
            # 1. Initialize database pool
            self.db_pool = DatabasePool(self.config.database)
            await self.db_pool.connect()

            # 2. Initialize repository
            self.repository = ExecutionRepository(self.db_pool)

            # 3. Initialize OMS gRPC client
            self.oms_client = OMSClient(self.config.oms_grpc)
            await self.oms_client.connect()

            # 4. Initialize risk manager
            self.risk_manager = RiskManager(self.config.risk, self.repository)

            # 5. Initialize strategy executor
            self.executor = StrategyExecutor(
                config=self.config,
                repository=self.repository,
                oms_client=self.oms_client,
                risk_manager=self.risk_manager,
            )

            # 6. Initialize Kafka consumer
            self.consumer = StrategyConsumer(
                config=self.config.kafka,
                message_handler=self.executor.execute_strategy,
            )
            await self.consumer.start()

            logger.info("Strategy Worker started successfully")

        except Exception as e:
            logger.error("Failed to start Strategy Worker", error=str(e))
            await self.shutdown()
            raise

    async def shutdown(self) -> None:
        """Graceful shutdown."""
        if self._shutdown:
            return

        self._shutdown = True
        logger.info("Shutting down Strategy Worker...")

        # Stop consumer first (no new messages)
        if self.consumer:
            await self.consumer.stop()

        # Stop all active executions
        if self.executor:
            for execution_id in list(self.executor._active_executions.keys()):
                try:
                    await self.executor.stop_execution(execution_id)
                except Exception as e:
                    logger.error(
                        "Error stopping execution",
                        execution_id=str(execution_id),
                        error=str(e),
                    )

        # Close connections
        if self.oms_client:
            await self.oms_client.close()

        if self.db_pool:
            await self.db_pool.close()

        logger.info("Strategy Worker shutdown complete")

    async def run(self) -> None:
        """Run the application."""
        # Setup signal handlers
        loop = asyncio.get_running_loop()

        for sig in (signal.SIGTERM, signal.SIGINT):
            loop.add_signal_handler(sig, lambda: asyncio.create_task(self.shutdown()))

        try:
            await self.startup()

            # Keep running until shutdown
            while not self._shutdown:
                await asyncio.sleep(1)

        except Exception as e:
            logger.error("Unexpected error", error=str(e))
            raise
        finally:
            await self.shutdown()


async def main() -> None:
    """Main entry point."""
    # Use uvloop for better performance
    uvloop.install()

    app = StrategyWorkerApp()

    try:
        await app.run()
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt")
    except Exception as e:
        logger.error("Fatal error", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
