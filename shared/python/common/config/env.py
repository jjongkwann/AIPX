"""Environment configuration for AIPX."""

import os
from typing import Optional, Any
from pathlib import Path


class EnvConfig:
    """
    Environment-based configuration for AIPX.

    Provides type-safe environment variable access with defaults and validation.

    Example:
        >>> env = EnvConfig()
        >>> port = env.get_int("PORT", default=8000)
        >>> debug = env.get_bool("DEBUG", default=False)
        >>> api_key = env.get_str("API_KEY", required=True)
    """

    def __init__(self, env_file: Optional[str] = None) -> None:
        """
        Initialize EnvConfig.

        Args:
            env_file: Path to .env file (optional)
        """
        if env_file and Path(env_file).exists():
            self._load_env_file(env_file)

    def _load_env_file(self, env_file: str) -> None:
        """
        Load environment variables from file.

        Args:
            env_file: Path to .env file
        """
        with open(env_file, "r") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#"):
                    key, _, value = line.partition("=")
                    key = key.strip()
                    value = value.strip().strip('"').strip("'")
                    os.environ[key] = value

    def get_str(
        self,
        key: str,
        default: Optional[str] = None,
        required: bool = False,
    ) -> str:
        """
        Get string environment variable.

        Args:
            key: Environment variable name
            default: Default value if not found
            required: Whether variable is required

        Returns:
            String value

        Raises:
            ValueError: If required and not found
        """
        value = os.getenv(key, default)

        if required and value is None:
            raise ValueError(f"Required environment variable '{key}' not found")

        return value

    def get_int(
        self,
        key: str,
        default: Optional[int] = None,
        required: bool = False,
    ) -> int:
        """
        Get integer environment variable.

        Args:
            key: Environment variable name
            default: Default value if not found
            required: Whether variable is required

        Returns:
            Integer value

        Raises:
            ValueError: If required and not found, or invalid integer
        """
        value = os.getenv(key)

        if value is None:
            if required:
                raise ValueError(f"Required environment variable '{key}' not found")
            return default

        try:
            return int(value)
        except ValueError as e:
            raise ValueError(
                f"Environment variable '{key}' has invalid integer value: {value}"
            ) from e

    def get_float(
        self,
        key: str,
        default: Optional[float] = None,
        required: bool = False,
    ) -> float:
        """
        Get float environment variable.

        Args:
            key: Environment variable name
            default: Default value if not found
            required: Whether variable is required

        Returns:
            Float value

        Raises:
            ValueError: If required and not found, or invalid float
        """
        value = os.getenv(key)

        if value is None:
            if required:
                raise ValueError(f"Required environment variable '{key}' not found")
            return default

        try:
            return float(value)
        except ValueError as e:
            raise ValueError(
                f"Environment variable '{key}' has invalid float value: {value}"
            ) from e

    def get_bool(
        self,
        key: str,
        default: Optional[bool] = None,
        required: bool = False,
    ) -> bool:
        """
        Get boolean environment variable.

        Args:
            key: Environment variable name
            default: Default value if not found
            required: Whether variable is required

        Returns:
            Boolean value

        Raises:
            ValueError: If required and not found
        """
        value = os.getenv(key)

        if value is None:
            if required:
                raise ValueError(f"Required environment variable '{key}' not found")
            return default

        return value.lower() in ("true", "1", "yes", "on")

    def get_list(
        self,
        key: str,
        separator: str = ",",
        default: Optional[list] = None,
        required: bool = False,
    ) -> list[str]:
        """
        Get list environment variable.

        Args:
            key: Environment variable name
            separator: List item separator
            default: Default value if not found
            required: Whether variable is required

        Returns:
            List of strings

        Raises:
            ValueError: If required and not found
        """
        value = os.getenv(key)

        if value is None:
            if required:
                raise ValueError(f"Required environment variable '{key}' not found")
            return default or []

        return [item.strip() for item in value.split(separator) if item.strip()]

    def get_dict(
        self,
        prefix: str,
        strip_prefix: bool = True,
    ) -> dict[str, str]:
        """
        Get all environment variables with a prefix.

        Args:
            prefix: Environment variable prefix
            strip_prefix: Whether to strip prefix from keys

        Returns:
            Dictionary of matching environment variables

        Example:
            >>> # With DB_HOST=localhost, DB_PORT=5432
            >>> env.get_dict("DB_")
            {'HOST': 'localhost', 'PORT': '5432'}
        """
        result = {}
        prefix_len = len(prefix)

        for key, value in os.environ.items():
            if key.startswith(prefix):
                result_key = key[prefix_len:] if strip_prefix else key
                result[result_key] = value

        return result


def get_env_config(env_file: Optional[str] = None) -> EnvConfig:
    """
    Get EnvConfig instance.

    Args:
        env_file: Path to .env file (optional)

    Returns:
        EnvConfig instance

    Example:
        >>> env = get_env_config(".env")
        >>> port = env.get_int("PORT", default=8000)
    """
    return EnvConfig(env_file=env_file)
