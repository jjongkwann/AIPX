"""Settings management for AIPX."""

import os
from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class DatabaseSettings:
    """Database configuration settings."""

    host: str = "localhost"
    port: int = 5432
    database: str = "aipx"
    username: str = "postgres"
    password: str = ""
    pool_size: int = 10
    max_overflow: int = 20
    echo: bool = False

    @classmethod
    def from_env(cls, prefix: str = "DB_") -> "DatabaseSettings":
        """
        Create DatabaseSettings from environment variables.

        Args:
            prefix: Environment variable prefix

        Returns:
            DatabaseSettings instance
        """
        return cls(
            host=os.getenv(f"{prefix}HOST", "localhost"),
            port=int(os.getenv(f"{prefix}PORT", "5432")),
            database=os.getenv(f"{prefix}DATABASE", "aipx"),
            username=os.getenv(f"{prefix}USERNAME", "postgres"),
            password=os.getenv(f"{prefix}PASSWORD", ""),
            pool_size=int(os.getenv(f"{prefix}POOL_SIZE", "10")),
            max_overflow=int(os.getenv(f"{prefix}MAX_OVERFLOW", "20")),
            echo=os.getenv(f"{prefix}ECHO", "false").lower() == "true",
        )

    def get_connection_string(self, async_driver: bool = True) -> str:
        """
        Get database connection string.

        Args:
            async_driver: Whether to use async driver

        Returns:
            PostgreSQL connection string
        """
        driver = "postgresql+asyncpg" if async_driver else "postgresql"
        return f"{driver}://{self.username}:{self.password}@{self.host}:{self.port}/{self.database}"


@dataclass
class RedisSettings:
    """Redis configuration settings."""

    host: str = "localhost"
    port: int = 6379
    db: int = 0
    password: Optional[str] = None
    ssl: bool = False

    @classmethod
    def from_env(cls, prefix: str = "REDIS_") -> "RedisSettings":
        """
        Create RedisSettings from environment variables.

        Args:
            prefix: Environment variable prefix

        Returns:
            RedisSettings instance
        """
        return cls(
            host=os.getenv(f"{prefix}HOST", "localhost"),
            port=int(os.getenv(f"{prefix}PORT", "6379")),
            db=int(os.getenv(f"{prefix}DB", "0")),
            password=os.getenv(f"{prefix}PASSWORD"),
            ssl=os.getenv(f"{prefix}SSL", "false").lower() == "true",
        )


@dataclass
class KafkaSettings:
    """Kafka configuration settings."""

    bootstrap_servers: str = "localhost:9092"
    security_protocol: str = "PLAINTEXT"
    sasl_mechanism: Optional[str] = None
    sasl_username: Optional[str] = None
    sasl_password: Optional[str] = None

    @classmethod
    def from_env(cls, prefix: str = "KAFKA_") -> "KafkaSettings":
        """
        Create KafkaSettings from environment variables.

        Args:
            prefix: Environment variable prefix

        Returns:
            KafkaSettings instance
        """
        return cls(
            bootstrap_servers=os.getenv(f"{prefix}BOOTSTRAP_SERVERS", "localhost:9092"),
            security_protocol=os.getenv(f"{prefix}SECURITY_PROTOCOL", "PLAINTEXT"),
            sasl_mechanism=os.getenv(f"{prefix}SASL_MECHANISM"),
            sasl_username=os.getenv(f"{prefix}SASL_USERNAME"),
            sasl_password=os.getenv(f"{prefix}SASL_PASSWORD"),
        )


@dataclass
class AppSettings:
    """Application configuration settings."""

    name: str = "AIPX"
    version: str = "0.1.0"
    environment: str = "development"
    debug: bool = False
    host: str = "0.0.0.0"
    port: int = 8000
    log_level: str = "INFO"
    cors_origins: list[str] = field(default_factory=list)

    @classmethod
    def from_env(cls, prefix: str = "APP_") -> "AppSettings":
        """
        Create AppSettings from environment variables.

        Args:
            prefix: Environment variable prefix

        Returns:
            AppSettings instance
        """
        cors_origins_str = os.getenv(f"{prefix}CORS_ORIGINS", "")
        cors_origins = [o.strip() for o in cors_origins_str.split(",") if o.strip()]

        return cls(
            name=os.getenv(f"{prefix}NAME", "AIPX"),
            version=os.getenv(f"{prefix}VERSION", "0.1.0"),
            environment=os.getenv(f"{prefix}ENVIRONMENT", "development"),
            debug=os.getenv(f"{prefix}DEBUG", "false").lower() == "true",
            host=os.getenv(f"{prefix}HOST", "0.0.0.0"),
            port=int(os.getenv(f"{prefix}PORT", "8000")),
            log_level=os.getenv(f"{prefix}LOG_LEVEL", "INFO"),
            cors_origins=cors_origins,
        )


@dataclass
class Settings:
    """
    Main settings class for AIPX applications.

    Aggregates all configuration settings from environment variables.

    Example:
        >>> settings = Settings.from_env()
        >>> print(settings.app.name)
        >>> print(settings.database.get_connection_string())
        >>> print(settings.redis.host)
    """

    app: AppSettings = field(default_factory=AppSettings)
    database: DatabaseSettings = field(default_factory=DatabaseSettings)
    redis: RedisSettings = field(default_factory=RedisSettings)
    kafka: KafkaSettings = field(default_factory=KafkaSettings)

    @classmethod
    def from_env(cls, env_file: Optional[str] = None) -> "Settings":
        """
        Create Settings from environment variables.

        Args:
            env_file: Optional path to .env file

        Returns:
            Settings instance

        Example:
            >>> settings = Settings.from_env(".env")
        """
        # Load .env file if provided
        if env_file:
            from .env import EnvConfig

            EnvConfig(env_file=env_file)

        return cls(
            app=AppSettings.from_env(),
            database=DatabaseSettings.from_env(),
            redis=RedisSettings.from_env(),
            kafka=KafkaSettings.from_env(),
        )

    def to_dict(self) -> Dict[str, Any]:
        """
        Convert settings to dictionary.

        Returns:
            Dictionary representation of settings
        """
        from dataclasses import asdict

        return asdict(self)
