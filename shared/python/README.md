# AIPX Common Shared Packages

Production-ready Python shared packages for AIPX microservices platform.

## Packages

### 1. Kafka Package (`common.kafka`)

High-level Kafka producer and consumer with JSON serialization, error handling, and automatic offset management.

**Features:**
- Async/await support
- JSON serialization/deserialization
- Automatic reconnection
- Context manager support
- Batch operations
- Comprehensive error handling

**Usage:**

```python
from common.kafka import KafkaProducer, KafkaConsumer, KafkaConfig

# Producer
config = KafkaConfig.from_env()
async with KafkaProducer(config) as producer:
    await producer.send(
        "user-events",
        {"user_id": 123, "action": "login"},
        key="user-123"
    )

# Consumer
async def handle_message(message):
    print(f"Received: {message['value']}")

async with KafkaConsumer(config, "my-service-group") as consumer:
    await consumer.consume("user-events", handle_message)
```

### 2. Redis Package (`common.redis`)

Async Redis client with caching, distributed locking, and automatic serialization.

**Features:**
- Async/await support
- JSON serialization
- Distributed locking with TTL
- Caching decorator
- Connection pooling
- Hash operations

**Usage:**

```python
from common.redis import RedisClient, RedisCache, RedisLock, RedisConfig

# Client
config = RedisConfig.from_env()
async with RedisClient(config) as client:
    await client.set("user:123", {"name": "John", "age": 30})
    user = await client.get("user:123")

# Cache
async with RedisCache(config, default_ttl=300) as cache:
    # Direct caching
    await cache.set("key", {"data": "value"}, ttl=600)
    data = await cache.get("key")

    # Decorator
    @cache.cached(ttl=600, key_prefix="user")
    async def get_user(user_id: int):
        return {"id": user_id, "name": "John"}

# Distributed Lock
async with RedisClient(config) as client:
    lock = RedisLock(client, "resource:123", ttl=30)
    async with lock:
        # Critical section - only one process can execute
        await process_resource()
```

### 3. Logger Package (`common.logger`)

Structured JSON logging with context management and FastAPI middleware.

**Features:**
- JSON structured logging
- Context management
- Request/response logging middleware
- Thread-safe context variables
- Exception tracking
- ELK/Splunk compatible

**Usage:**

```python
from common.logger import get_logger, log_context, LoggingMiddleware, setup_logging

# Setup global logging
setup_logging(level="INFO", use_json=True)

# Get logger
logger = get_logger("my-service")

# Basic logging
logger.info("User logged in", user_id=123, ip="192.168.1.1")
logger.error("Database error", error=str(e), exc_info=True)

# Context logging
with log_context(logger, request_id="abc-123", user_id=456):
    logger.info("Processing request")  # Includes context
    logger.info("Request completed")   # Still includes context

# FastAPI middleware
from fastapi import FastAPI

app = FastAPI()
app.add_middleware(LoggingMiddleware)
```

### 4. Config Package (`common.config`)

Environment-based configuration management with validation.

**Features:**
- Type-safe environment variable access
- Configuration validation
- Settings aggregation
- .env file support
- Database/Redis/Kafka presets

**Usage:**

```python
from common.config import Settings, EnvConfig, get_env_config, validate_config

# Load all settings from environment
settings = Settings.from_env(".env")
print(settings.app.name)
print(settings.database.get_connection_string())
print(settings.redis.host)

# Type-safe environment access
env = get_env_config(".env")
port = env.get_int("PORT", default=8000)
debug = env.get_bool("DEBUG", default=False)
allowed_hosts = env.get_list("ALLOWED_HOSTS")

# Configuration validation
schema = {
    "port": {"required": True, "type": int, "min": 1, "max": 65535},
    "host": {"required": True, "type": str},
}
validate_config(config, schema)
```

## Installation

### Using pip

```bash
# Install core dependencies
pip install -r requirements.txt

# Install with development dependencies
pip install -r requirements-dev.txt

# Install with test dependencies
pip install -r requirements-test.txt
```

### Using Poetry

```bash
# Install core dependencies
poetry install

# Install with all optional dependencies
poetry install --with dev,test
```

### As a package

```bash
# Install in editable mode
pip install -e .

# Install with extras
pip install -e ".[dev,test]"
```

## Environment Variables

Create a `.env` file with the following variables:

```bash
# Application
APP_NAME=my-service
APP_VERSION=0.1.0
APP_ENVIRONMENT=development
APP_DEBUG=false
APP_HOST=0.0.0.0
APP_PORT=8000
APP_LOG_LEVEL=INFO

# Database
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=aipx
DB_USERNAME=postgres
DB_PASSWORD=secret
DB_POOL_SIZE=10
DB_MAX_OVERFLOW=20

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=
REDIS_SSL=false

# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
KAFKA_SASL_MECHANISM=
KAFKA_SASL_USERNAME=
KAFKA_SASL_PASSWORD=
```

## Testing

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=common --cov-report=html

# Run specific test file
pytest tests/test_kafka.py

# Run with verbosity
pytest -v
```

## Code Quality

```bash
# Format code
black .

# Lint code
ruff check .

# Type check
mypy common/

# Run all quality checks
black . && ruff check . && mypy common/
```

## Development

### Project Structure

```
shared/python/
├── common/                 # Main package
│   ├── __init__.py
│   ├── kafka/             # Kafka package
│   │   ├── __init__.py
│   │   ├── producer.py
│   │   ├── consumer.py
│   │   ├── config.py
│   │   └── exceptions.py
│   ├── redis/             # Redis package
│   │   ├── __init__.py
│   │   ├── client.py
│   │   ├── cache.py
│   │   ├── lock.py
│   │   ├── config.py
│   │   └── exceptions.py
│   ├── logger/            # Logger package
│   │   ├── __init__.py
│   │   ├── logger.py
│   │   ├── context.py
│   │   └── middleware.py
│   └── config/            # Config package
│       ├── __init__.py
│       ├── settings.py
│       ├── env.py
│       └── validators.py
├── tests/                 # Test files
├── pyproject.toml         # Package configuration
├── requirements.txt       # Core dependencies
├── requirements-dev.txt   # Dev dependencies
└── README.md             # This file
```

### Adding a New Package

1. Create package directory under `common/`
2. Add `__init__.py` with exports
3. Implement modules with type hints and docstrings
4. Add tests in `tests/`
5. Update `common/__init__.py` to export new package
6. Update this README

## Performance Considerations

- **Kafka**: Uses connection pooling and batch operations
- **Redis**: Connection pooling with async support
- **Logger**: Async-safe with minimal overhead
- **Config**: Loaded once at startup

## Security

- Credentials from environment variables only
- No hardcoded secrets
- SSL/TLS support for Redis and Kafka
- Input validation on all config
- Safe serialization/deserialization

## License

MIT License - See LICENSE file for details
