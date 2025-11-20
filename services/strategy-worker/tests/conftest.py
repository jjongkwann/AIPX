"""Pytest configuration and fixtures."""

import asyncio
from decimal import Decimal
from uuid import uuid4

import pytest
import pytest_asyncio

from src.config import (
    Config,
    DatabaseConfig,
    KafkaConfig,
    OMSGrpcConfig,
    RiskConfig,
    ExecutionConfig,
    LoggingConfig,
    MonitoringConfig,
    ServiceConfig,
)


@pytest.fixture(scope="session")
def event_loop():
    """Create event loop for async tests."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
def test_config() -> Config:
    """Create test configuration."""
    return Config(
        kafka=KafkaConfig(
            bootstrap_servers="localhost:9092",
            consumer_group="test-worker",
        ),
        oms_grpc=OMSGrpcConfig(
            host="localhost",
            port=50051,
        ),
        database=DatabaseConfig(
            host="localhost",
            port=5432,
            database="aipx_test",
            user="postgres",
            password="postgres",
        ),
        risk=RiskConfig(
            max_total_exposure=100000.0,
            max_position_size=0.30,
            max_daily_loss=0.05,
        ),
        execution=ExecutionConfig(
            rebalance_interval=60,
            position_check_interval=30,
        ),
        logging=LoggingConfig(level="DEBUG"),
        monitoring=MonitoringConfig(enabled=False),
        service=ServiceConfig(environment="test"),
    )


@pytest.fixture
def sample_strategy_data() -> dict:
    """Create sample strategy data."""
    return {
        "strategy_id": str(uuid4()),
        "user_id": str(uuid4()),
        "config": {
            "name": "Test Strategy",
            "type": "MOMENTUM",
            "symbols": ["AAPL", "MSFT"],
            "allocation": {
                "AAPL": 0.50,
                "MSFT": 0.50,
            },
            "initial_capital": 10000.0,
            "risk_params": {
                "max_position_size": 0.30,
                "stop_loss_pct": 0.05,
                "max_daily_loss": 0.05,
            },
        },
    }


@pytest.fixture
def execution_id():
    """Generate test execution ID."""
    return uuid4()


@pytest.fixture
def strategy_id():
    """Generate test strategy ID."""
    return uuid4()


@pytest.fixture
def user_id():
    """Generate test user ID."""
    return uuid4()
