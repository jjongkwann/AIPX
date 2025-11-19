"""Redis configuration for AIPX."""

from dataclasses import dataclass, field
from typing import Optional


@dataclass
class RedisConfig:
    """
    Configuration for Redis client.

    Attributes:
        host: Redis host
        port: Redis port
        db: Database number
        password: Optional password
        socket_timeout: Socket timeout in seconds
        socket_connect_timeout: Socket connection timeout in seconds
        max_connections: Maximum number of connections in the pool
        decode_responses: Whether to decode responses to strings
        ssl: Whether to use SSL
        ssl_cert_reqs: SSL certificate requirements
    """

    host: str = "localhost"
    port: int = 6379
    db: int = 0
    password: Optional[str] = None
    socket_timeout: float = 5.0
    socket_connect_timeout: float = 5.0
    max_connections: int = 50
    decode_responses: bool = True
    ssl: bool = False
    ssl_cert_reqs: Optional[str] = None

    @classmethod
    def from_env(cls) -> "RedisConfig":
        """
        Create RedisConfig from environment variables.

        Returns:
            RedisConfig instance

        Environment variables:
            REDIS_HOST: Redis host
            REDIS_PORT: Redis port
            REDIS_DB: Database number
            REDIS_PASSWORD: Password
            REDIS_SSL: Whether to use SSL
        """
        import os

        return cls(
            host=os.getenv("REDIS_HOST", "localhost"),
            port=int(os.getenv("REDIS_PORT", "6379")),
            db=int(os.getenv("REDIS_DB", "0")),
            password=os.getenv("REDIS_PASSWORD"),
            ssl=os.getenv("REDIS_SSL", "false").lower() == "true",
        )

    def to_dict(self) -> dict:
        """
        Convert configuration to dictionary.

        Returns:
            Dictionary of configuration parameters
        """
        config = {
            "host": self.host,
            "port": self.port,
            "db": self.db,
            "socket_timeout": self.socket_timeout,
            "socket_connect_timeout": self.socket_connect_timeout,
            "max_connections": self.max_connections,
            "decode_responses": self.decode_responses,
        }

        if self.password:
            config["password"] = self.password

        if self.ssl:
            config["ssl"] = True
            if self.ssl_cert_reqs:
                config["ssl_cert_reqs"] = self.ssl_cert_reqs

        return config
