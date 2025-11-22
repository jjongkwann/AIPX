"""Configuration management for ML Inference Service."""
from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    """Service settings."""

    # Service Configuration
    service_name: str = "ml-inference-service"
    service_port: int = 8005
    env: str = "development"

    # Triton Server
    triton_url: str = "triton:8001"
    triton_http_url: str = "triton:8000"
    triton_metrics_url: str = "triton:8002"

    # Database
    db_host: str = "timescaledb"
    db_port: int = 5432
    db_name: str = "aipx_trading"
    db_user: str = "aipx_user"
    db_password: str

    # Redis
    redis_host: str = "redis"
    redis_port: int = 6379
    redis_db: int = 0

    # Model Configuration
    model_repository: str = "/models"
    model_refresh_interval: int = 300

    # Feature Engineering
    feature_window_size: int = 60
    feature_normalization: str = "standard"

    # Inference Configuration
    max_batch_size: int = 32
    batch_timeout_ms: int = 100
    gpu_memory_fraction: float = 0.8

    # Model Versions
    lstm_model_version: int = 1
    transformer_model_version: int = 1
    ensemble_model_version: int = 1

    # Monitoring
    metrics_port: int = 9090
    log_level: str = "INFO"

    # Training Configuration
    training_data_path: str = "/data/training"
    model_checkpoint_path: str = "/models/checkpoints"
    training_epochs: int = 100
    learning_rate: float = 0.001
    batch_size: int = 64

    # S3 Configuration
    aws_region: str = "ap-northeast-2"
    s3_bucket: str = "aipx-ml-models"
    s3_prefix: str = "models/"

    # Performance
    worker_processes: int = 4
    worker_timeout: int = 300
    keep_alive: int = 5

    @property
    def database_url(self) -> str:
        """Get async database URL."""
        return f"postgresql+asyncpg://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    @property
    def sync_database_url(self) -> str:
        """Get sync database URL."""
        return f"postgresql://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    @property
    def redis_url(self) -> str:
        """Get Redis URL."""
        return f"redis://{self.redis_host}:{self.redis_port}/{self.redis_db}"

    class Config:
        env_file = ".env"
        case_sensitive = False


# Lazy loading for settings to support testing
_settings: Optional[Settings] = None


def get_settings() -> Settings:
    """Get settings instance (lazy loading)."""
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings


# For backwards compatibility - but prefer get_settings() for testability
class _SettingsProxy:
    """Proxy class for lazy settings access."""

    def __getattr__(self, name):
        return getattr(get_settings(), name)


settings = _SettingsProxy()
