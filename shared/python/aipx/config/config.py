"""
Configuration management using Pydantic Settings
"""

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings"""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Environment
    environment: str = Field(default="dev", alias="ENVIRONMENT")

    # Kafka
    kafka_brokers: str = Field(default="localhost:9092", alias="KAFKA_BROKERS")
    kafka_topic_market_data: str = Field(default="market-data-raw", alias="KAFKA_TOPIC_MARKET_DATA")
    kafka_topic_trading_signals: str = Field(
        default="trading-signals", alias="KAFKA_TOPIC_TRADING_SIGNALS"
    )
    kafka_topic_order_events: str = Field(default="order-events", alias="KAFKA_TOPIC_ORDER_EVENTS")
    kafka_topic_executions: str = Field(default="executions", alias="KAFKA_TOPIC_EXECUTIONS")
    kafka_consumer_group: str = Field(default="aipx-consumer-group", alias="KAFKA_CONSUMER_GROUP")

    # Redis
    redis_host: str = Field(default="localhost", alias="REDIS_HOST")
    redis_port: int = Field(default=6379, alias="REDIS_PORT")
    redis_password: str = Field(default="", alias="REDIS_PASSWORD")
    redis_db: int = Field(default=0, alias="REDIS_DB")
    redis_max_connections: int = Field(default=10, alias="REDIS_MAX_CONNECTIONS")

    # PostgreSQL
    postgres_host: str = Field(default="localhost", alias="POSTGRES_HOST")
    postgres_port: int = Field(default=5432, alias="POSTGRES_PORT")
    postgres_user: str = Field(default="aipx", alias="POSTGRES_USER")
    postgres_password: str = Field(default="aipx_dev_password", alias="POSTGRES_PASSWORD")
    postgres_db: str = Field(default="aipx", alias="POSTGRES_DB")
    postgres_max_connections: int = Field(default=20, alias="POSTGRES_MAX_CONNECTIONS")

    # TimescaleDB
    timescaledb_host: str = Field(default="localhost", alias="TIMESCALEDB_HOST")
    timescaledb_port: int = Field(default=5433, alias="TIMESCALEDB_PORT")
    timescaledb_user: str = Field(default="aipx", alias="TIMESCALEDB_USER")
    timescaledb_password: str = Field(default="aipx_dev_password", alias="TIMESCALEDB_PASSWORD")
    timescaledb_db: str = Field(default="aipx_timeseries", alias="TIMESCALEDB_DB")

    # AWS
    aws_region: str = Field(default="ap-northeast-2", alias="AWS_REGION")
    aws_access_key_id: str = Field(default="", alias="AWS_ACCESS_KEY_ID")
    aws_secret_access_key: str = Field(default="", alias="AWS_SECRET_ACCESS_KEY")
    s3_bucket_name: str = Field(default="aipx-data-lake-dev", alias="S3_BUCKET_NAME")
    msk_bootstrap_servers: str = Field(default="", alias="MSK_BOOTSTRAP_SERVERS")
    elasticache_endpoint: str = Field(default="", alias="ELASTICACHE_ENDPOINT")
    rds_endpoint: str = Field(default="", alias="RDS_ENDPOINT")

    # Service Ports
    data_ingestion_port: int = Field(default=8081, alias="DATA_INGESTION_PORT")
    data_ingestion_ws_port: int = Field(default=8082, alias="DATA_INGESTION_WS_PORT")
    oms_grpc_port: int = Field(default=50051, alias="OMS_GRPC_PORT")
    oms_http_port: int = Field(default=8083, alias="OMS_HTTP_PORT")
    cognitive_api_port: int = Field(default=8084, alias="COGNITIVE_API_PORT")

    # JWT
    jwt_secret: str = Field(
        default="your-super-secret-jwt-key-change-this-in-production",
        alias="JWT_SECRET",
    )
    jwt_expiration: int = Field(default=3600, alias="JWT_EXPIRATION")
    jwt_refresh_expiration: int = Field(default=604800, alias="JWT_REFRESH_EXPIRATION")

    # Trading
    max_position_size: float = Field(default=1000000.0, alias="MAX_POSITION_SIZE")
    max_daily_loss_percent: float = Field(default=5.0, alias="MAX_DAILY_LOSS_PERCENT")
    max_loss_per_trade_percent: float = Field(default=2.0, alias="MAX_LOSS_PER_TRADE_PERCENT")
    default_order_timeout: int = Field(default=30, alias="DEFAULT_ORDER_TIMEOUT")

    # Feature Flags
    enable_paper_trading: bool = Field(default=True, alias="ENABLE_PAPER_TRADING")
    enable_live_trading: bool = Field(default=False, alias="ENABLE_LIVE_TRADING")
    enable_backtesting: bool = Field(default=True, alias="ENABLE_BACKTESTING")

    # Monitoring
    enable_prometheus_metrics: bool = Field(default=True, alias="ENABLE_PROMETHEUS_METRICS")
    prometheus_port: int = Field(default=9090, alias="PROMETHEUS_PORT")
    enable_jaeger_tracing: bool = Field(default=False, alias="ENABLE_JAEGER_TRACING")
    jaeger_agent_host: str = Field(default="localhost", alias="JAEGER_AGENT_HOST")
    jaeger_agent_port: int = Field(default=6831, alias="JAEGER_AGENT_PORT")

    # Logging
    log_level: str = Field(default="INFO", alias="LOG_LEVEL")
    log_format: str = Field(default="json", alias="LOG_FORMAT")
    log_output: str = Field(default="stdout", alias="LOG_OUTPUT")

    # Development
    enable_hot_reload: bool = Field(default=True, alias="ENABLE_HOT_RELOAD")
    debug_mode: bool = Field(default=False, alias="DEBUG_MODE")
    debug_port: int = Field(default=2345, alias="DEBUG_PORT")

    @property
    def database_url(self) -> str:
        """PostgreSQL database URL"""
        return f"postgresql://{self.postgres_user}:{self.postgres_password}@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"

    @property
    def timescaledb_url(self) -> str:
        """TimescaleDB database URL"""
        return f"postgresql://{self.timescaledb_user}:{self.timescaledb_password}@{self.timescaledb_host}:{self.timescaledb_port}/{self.timescaledb_db}"

    @property
    def kafka_brokers_list(self) -> list[str]:
        """Kafka brokers as list"""
        return self.kafka_brokers.split(",")


# Global settings instance
_settings: Settings | None = None


def get_settings() -> Settings:
    """
    Get global settings instance (singleton)

    Returns:
        Settings instance
    """
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings


def reload_settings() -> Settings:
    """
    Reload settings from environment

    Returns:
        New Settings instance
    """
    global _settings
    _settings = Settings()
    return _settings
