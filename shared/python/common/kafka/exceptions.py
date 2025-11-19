"""Kafka-related exceptions for AIPX."""


class KafkaError(Exception):
    """Base exception for all Kafka-related errors."""

    def __init__(self, message: str, original_error: Exception | None = None) -> None:
        """
        Initialize KafkaError.

        Args:
            message: Error message
            original_error: Original exception that caused this error
        """
        super().__init__(message)
        self.original_error = original_error


class KafkaProducerError(KafkaError):
    """Exception raised when producer operations fail."""

    pass


class KafkaConsumerError(KafkaError):
    """Exception raised when consumer operations fail."""

    pass


class KafkaConnectionError(KafkaError):
    """Exception raised when Kafka connection fails."""

    pass


class KafkaSerializationError(KafkaError):
    """Exception raised when message serialization/deserialization fails."""

    pass
