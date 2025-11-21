"""Configuration management for Cognitive Service."""

from functools import lru_cache
from typing import Literal

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # Service configuration
    service_name: str = "cognitive-service"
    host: str = "0.0.0.0"
    port: int = 8001
    environment: Literal["development", "staging", "production"] = "development"
    debug: bool = False

    # LLM Configuration
    anthropic_api_key: str = Field(..., description="Anthropic API key for Claude")
    openai_api_key: str | None = Field(None, description="OpenAI API key (optional)")
    default_llm: Literal["claude", "openai"] = "claude"
    claude_model: str = "claude-3-5-sonnet-20241022"
    openai_model: str = "gpt-4-turbo-preview"

    # LLM Parameters
    llm_temperature: float = Field(default=0.7, ge=0.0, le=2.0)
    llm_max_tokens: int = Field(default=4096, gt=0)
    llm_timeout: int = Field(default=60, gt=0, description="LLM request timeout in seconds")

    # Database Configuration
    postgres_host: str = Field(default="localhost")
    postgres_port: int = Field(default=5432, ge=1, le=65535)
    postgres_db: str = Field(default="aipx_cognitive")
    postgres_user: str = Field(default="postgres")
    postgres_password: str = Field(..., description="PostgreSQL password")
    postgres_pool_size: int = Field(default=10, ge=1)
    postgres_max_overflow: int = Field(default=20, ge=0)

    # Kafka Configuration
    kafka_brokers: str = Field(default="localhost:9092", description="Comma-separated Kafka broker addresses")
    kafka_topic_strategy_created: str = "strategy.created"
    kafka_topic_strategy_approved: str = "strategy.approved"
    kafka_topic_strategy_rejected: str = "strategy.rejected"

    # Redis Configuration
    redis_host: str = Field(default="localhost")
    redis_port: int = Field(default=6379, ge=1, le=65535)
    redis_db: int = Field(default=0, ge=0)
    redis_password: str | None = None
    redis_key_prefix: str = "cognitive:"

    # Session Configuration
    session_ttl: int = Field(default=3600, gt=0, description="Session TTL in seconds (1 hour)")
    max_retry_count: int = Field(default=3, ge=1, le=10)

    # CORS Configuration
    cors_origins: list[str] = Field(default=["http://localhost:3000", "http://localhost:8080"])

    # Logging
    log_level: Literal["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"] = "INFO"
    log_format: Literal["json", "text"] = "json"

    # Monitoring
    enable_metrics: bool = True
    metrics_port: int = Field(default=9090, ge=1, le=65535)

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    @field_validator("kafka_brokers")
    @classmethod
    def validate_kafka_brokers(cls, v: str) -> str:
        """Validate Kafka brokers format."""
        brokers = [b.strip() for b in v.split(",")]
        for broker in brokers:
            if ":" not in broker:
                raise ValueError(f"Invalid broker format: {broker}. Expected host:port")
        return v

    @property
    def database_url(self) -> str:
        """Get PostgreSQL connection URL."""
        return (
            f"postgresql://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
        )

    @property
    def async_database_url(self) -> str:
        """Get async PostgreSQL connection URL."""
        return (
            f"postgresql+asyncpg://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
        )

    @property
    def redis_url(self) -> str:
        """Get Redis connection URL."""
        auth = f":{self.redis_password}@" if self.redis_password else ""
        return f"redis://{auth}{self.redis_host}:{self.redis_port}/{self.redis_db}"

    @property
    def kafka_broker_list(self) -> list[str]:
        """Get Kafka brokers as a list."""
        return [b.strip() for b in self.kafka_brokers.split(",")]


@lru_cache()
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
