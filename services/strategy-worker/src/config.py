"""Configuration management for Strategy Worker."""

import os
from pathlib import Path
from typing import Any

import yaml
from pydantic import Field
from pydantic_settings import BaseSettings


class KafkaConfig(BaseSettings):
    """Kafka configuration."""

    bootstrap_servers: str = "localhost:9092"
    consumer_group: str = "strategy-worker"
    auto_offset_reset: str = "earliest"
    enable_auto_commit: bool = False
    max_poll_records: int = 10
    session_timeout_ms: int = 30000
    topics: dict[str, str] = Field(
        default_factory=lambda: {
            "approved": "strategy.approved",
            "execution": "strategy.execution",
            "positions": "strategy.positions",
            "errors": "strategy.errors",
        }
    )


class RetryConfig(BaseSettings):
    """Retry configuration."""

    max_attempts: int = 3
    backoff_ms: int = 1000
    max_backoff_ms: int = 10000
    exponential_base: int = 2


class OMSGrpcConfig(BaseSettings):
    """OMS gRPC configuration."""

    host: str = "localhost"
    port: int = 50051
    timeout: int = 5
    max_message_size: int = 4194304  # 4MB
    retry: RetryConfig = Field(default_factory=RetryConfig)


class DatabaseConfig(BaseSettings):
    """Database configuration."""

    host: str = "localhost"
    port: int = 5432
    database: str = "aipx"
    user: str = "postgres"
    password: str = Field(default="postgres")
    min_pool_size: int = 5
    max_pool_size: int = 20
    command_timeout: int = 60
    ssl_mode: str = "prefer"

    @property
    def dsn(self) -> str:
        """Build PostgreSQL DSN."""
        return (
            f"postgresql://{self.user}:{self.password}@"
            f"{self.host}:{self.port}/{self.database}"
        )


class RedisConfig(BaseSettings):
    """Redis configuration."""

    host: str = "localhost"
    port: int = 6379
    db: int = 0
    password: str | None = None
    max_connections: int = 50
    socket_timeout: int = 5
    socket_connect_timeout: int = 5
    decode_responses: bool = False
    key_prefix: str = "strategy:worker:"


class RiskConfig(BaseSettings):
    """Risk management configuration."""

    max_total_exposure: float = 1000000.00
    max_position_size: float = 0.30
    max_daily_loss: float = 0.05
    max_daily_loss_amount: float = 50000.00
    stop_loss_buffer: float = 0.02
    min_order_size: float = 100.00
    max_order_size: float = 100000.00


class ExecutionConfig(BaseSettings):
    """Execution configuration."""

    rebalance_interval: int = 300
    position_check_interval: int = 60
    order_timeout: int = 30
    max_retries: int = 3
    batch_size: int = 10


class LoggingConfig(BaseSettings):
    """Logging configuration."""

    level: str = "INFO"
    format: str = "json"
    output: str = "stdout"


class MonitoringConfig(BaseSettings):
    """Monitoring configuration."""

    enabled: bool = True
    prometheus_port: int = 9091
    health_check_port: int = 8080


class ServiceConfig(BaseSettings):
    """Service configuration."""

    name: str = "strategy-worker"
    environment: str = Field(default="development")
    shutdown_timeout: int = 30
    graceful_shutdown: bool = True


class Config(BaseSettings):
    """Main configuration."""

    kafka: KafkaConfig = Field(default_factory=KafkaConfig)
    oms_grpc: OMSGrpcConfig = Field(default_factory=OMSGrpcConfig)
    database: DatabaseConfig = Field(default_factory=DatabaseConfig)
    redis: RedisConfig = Field(default_factory=RedisConfig)
    risk: RiskConfig = Field(default_factory=RiskConfig)
    execution: ExecutionConfig = Field(default_factory=ExecutionConfig)
    logging: LoggingConfig = Field(default_factory=LoggingConfig)
    monitoring: MonitoringConfig = Field(default_factory=MonitoringConfig)
    service: ServiceConfig = Field(default_factory=ServiceConfig)

    @classmethod
    def from_yaml(cls, config_path: str | Path) -> "Config":
        """Load configuration from YAML file with environment variable substitution."""
        config_path = Path(config_path)
        if not config_path.exists():
            raise FileNotFoundError(f"Config file not found: {config_path}")

        with open(config_path) as f:
            config_dict = yaml.safe_load(f)

        # Recursively substitute environment variables
        config_dict = cls._substitute_env_vars(config_dict)

        return cls(**config_dict)

    @classmethod
    def _substitute_env_vars(cls, config: Any) -> Any:
        """Recursively substitute environment variables in config."""
        if isinstance(config, dict):
            return {key: cls._substitute_env_vars(value) for key, value in config.items()}
        elif isinstance(config, list):
            return [cls._substitute_env_vars(item) for item in config]
        elif isinstance(config, str):
            # Handle ${VAR} and ${VAR:default} patterns
            if config.startswith("${") and config.endswith("}"):
                var_expr = config[2:-1]
                if ":" in var_expr:
                    var_name, default = var_expr.split(":", 1)
                    return os.getenv(var_name, default)
                else:
                    return os.getenv(var_expr, config)
            return config
        else:
            return config


# Global config instance
config: Config | None = None


def get_config() -> Config:
    """Get global config instance."""
    global config
    if config is None:
        config_path = os.getenv(
            "CONFIG_PATH", "/Users/jk/workspace/AIPX/services/strategy-worker/config/config.yaml"
        )
        config = Config.from_yaml(config_path)
    return config


def set_config(new_config: Config) -> None:
    """Set global config instance (mainly for testing)."""
    global config
    config = new_config
