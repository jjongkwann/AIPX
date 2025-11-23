"""Configuration validators for AIPX."""

from typing import Any, Dict, List


class ConfigValidationError(Exception):
    """Exception raised when configuration validation fails."""

    def __init__(self, errors: List[str]) -> None:
        """
        Initialize ConfigValidationError.

        Args:
            errors: List of validation error messages
        """
        self.errors = errors
        super().__init__(f"Configuration validation failed: {', '.join(errors)}")


def validate_config(config: Dict[str, Any], schema: Dict[str, Any]) -> None:
    """
    Validate configuration against schema.

    Args:
        config: Configuration dictionary to validate
        schema: Schema dictionary with validation rules

    Raises:
        ConfigValidationError: If validation fails

    Schema format:
        {
            "field_name": {
                "required": bool,
                "type": type,
                "min": int/float,
                "max": int/float,
                "choices": list,
                "pattern": str (regex),
            }
        }

    Example:
        >>> schema = {
        ...     "port": {"required": True, "type": int, "min": 1, "max": 65535},
        ...     "host": {"required": True, "type": str},
        ...     "debug": {"required": False, "type": bool},
        ... }
        >>> validate_config(config, schema)
    """
    errors: List[str] = []

    for field, rules in schema.items():
        value = config.get(field)

        # Check required
        if rules.get("required", False) and value is None:
            errors.append(f"Required field '{field}' is missing")
            continue

        # Skip validation if value is None and not required
        if value is None:
            continue

        # Check type
        expected_type = rules.get("type")
        if expected_type and not isinstance(value, expected_type):
            errors.append(
                f"Field '{field}' has invalid type: "
                f"expected {expected_type.__name__}, got {type(value).__name__}"
            )
            continue

        # Check min value
        min_value = rules.get("min")
        if min_value is not None and value < min_value:
            errors.append(f"Field '{field}' value {value} is less than minimum {min_value}")

        # Check max value
        max_value = rules.get("max")
        if max_value is not None and value > max_value:
            errors.append(f"Field '{field}' value {value} is greater than maximum {max_value}")

        # Check choices
        choices = rules.get("choices")
        if choices and value not in choices:
            errors.append(f"Field '{field}' value '{value}' is not in allowed choices: {choices}")

        # Check pattern
        pattern = rules.get("pattern")
        if pattern and isinstance(value, str):
            import re

            if not re.match(pattern, value):
                errors.append(f"Field '{field}' value '{value}' does not match pattern '{pattern}'")

    if errors:
        raise ConfigValidationError(errors)


def validate_url(url: str) -> bool:
    """
    Validate URL format.

    Args:
        url: URL to validate

    Returns:
        True if valid URL
    """
    import re

    pattern = re.compile(
        r"^https?://"  # http:// or https://
        r"(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+[A-Z]{2,6}\.?|"  # domain...
        r"localhost|"  # localhost...
        r"\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})"  # ...or ip
        r"(?::\d+)?"  # optional port
        r"(?:/?|[/?]\S+)$",
        re.IGNORECASE,
    )
    return bool(pattern.match(url))


def validate_port(port: int) -> bool:
    """
    Validate port number.

    Args:
        port: Port number to validate

    Returns:
        True if valid port (1-65535)
    """
    return 1 <= port <= 65535


def validate_email(email: str) -> bool:
    """
    Validate email format.

    Args:
        email: Email address to validate

    Returns:
        True if valid email format
    """
    import re

    pattern = re.compile(r"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$")
    return bool(pattern.match(email))
