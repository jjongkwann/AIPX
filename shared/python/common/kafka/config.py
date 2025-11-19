"""Kafka configuration for AIPX."""

from dataclasses import dataclass, field
from typing import Dict, Any


@dataclass
class KafkaConfig:
    """
    Configuration for Kafka producer and consumer.

    Attributes:
        bootstrap_servers: Comma-separated list of Kafka brokers
        security_protocol: Security protocol (PLAINTEXT, SSL, SASL_PLAINTEXT, SASL_SSL)
        sasl_mechanism: SASL mechanism if using SASL
        sasl_username: SASL username
        sasl_password: SASL password
        producer_config: Additional producer configuration
        consumer_config: Additional consumer configuration
    """

    bootstrap_servers: str
    security_protocol: str = "PLAINTEXT"
    sasl_mechanism: str | None = None
    sasl_username: str | None = None
    sasl_password: str | None = None
    producer_config: Dict[str, Any] = field(default_factory=dict)
    consumer_config: Dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_env(cls) -> "KafkaConfig":
        """
        Create KafkaConfig from environment variables.

        Returns:
            KafkaConfig instance

        Environment variables:
            KAFKA_BOOTSTRAP_SERVERS: Kafka brokers
            KAFKA_SECURITY_PROTOCOL: Security protocol
            KAFKA_SASL_MECHANISM: SASL mechanism
            KAFKA_SASL_USERNAME: SASL username
            KAFKA_SASL_PASSWORD: SASL password
        """
        import os

        return cls(
            bootstrap_servers=os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
            security_protocol=os.getenv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
            sasl_mechanism=os.getenv("KAFKA_SASL_MECHANISM"),
            sasl_username=os.getenv("KAFKA_SASL_USERNAME"),
            sasl_password=os.getenv("KAFKA_SASL_PASSWORD"),
        )

    def get_producer_config(self) -> Dict[str, Any]:
        """
        Get producer configuration dictionary.

        Returns:
            Configuration dictionary for KafkaProducer
        """
        config = {
            "bootstrap_servers": self.bootstrap_servers,
            "security_protocol": self.security_protocol,
            "value_serializer": lambda v: v.encode("utf-8") if isinstance(v, str) else v,
            "key_serializer": lambda k: k.encode("utf-8") if isinstance(k, str) else k,
        }

        if self.sasl_mechanism:
            config["sasl_mechanism"] = self.sasl_mechanism
            config["sasl_plain_username"] = self.sasl_username
            config["sasl_plain_password"] = self.sasl_password

        config.update(self.producer_config)
        return config

    def get_consumer_config(self, group_id: str) -> Dict[str, Any]:
        """
        Get consumer configuration dictionary.

        Args:
            group_id: Consumer group ID

        Returns:
            Configuration dictionary for KafkaConsumer
        """
        config = {
            "bootstrap_servers": self.bootstrap_servers,
            "security_protocol": self.security_protocol,
            "group_id": group_id,
            "value_deserializer": lambda v: v.decode("utf-8") if isinstance(v, bytes) else v,
            "key_deserializer": lambda k: k.decode("utf-8") if isinstance(k, bytes) else k,
            "auto_offset_reset": "earliest",
            "enable_auto_commit": False,
        }

        if self.sasl_mechanism:
            config["sasl_mechanism"] = self.sasl_mechanism
            config["sasl_plain_username"] = self.sasl_username
            config["sasl_plain_password"] = self.sasl_password

        config.update(self.consumer_config)
        return config
