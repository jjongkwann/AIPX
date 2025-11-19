# AIPX Python Shared Packages - Quick Start Guide

## Installation

```bash
cd /Users/jk/workspace/AIPX/shared/python

# Install core dependencies
pip install -r requirements.txt

# Or install in development mode
pip install -e .

# Or with all extras
pip install -e ".[dev,test]"
```

## Environment Setup

Create a `.env` file in your service directory:

```bash
# Application
APP_NAME=my-service
APP_VERSION=0.1.0
APP_ENVIRONMENT=development
APP_DEBUG=false
APP_PORT=8000
APP_LOG_LEVEL=INFO

# Database
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=aipx
DB_USERNAME=postgres
DB_PASSWORD=secret

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
```

## Quick Start Examples

### 1. Basic FastAPI Service with All Packages

```python
from fastapi import FastAPI, HTTPException
from common.kafka import KafkaProducer, KafkaConfig
from common.redis import RedisClient, RedisCache, RedisLock, RedisConfig
from common.logger import get_logger, setup_logging, LoggingMiddleware
from common.config import Settings

# Initialize
setup_logging(level="INFO", use_json=True)
logger = get_logger("my-service")
settings = Settings.from_env()

app = FastAPI(title=settings.app.name, version=settings.app.version)
app.add_middleware(LoggingMiddleware)

# Global state
kafka_producer = None
redis_client = None
redis_cache = None

@app.on_event("startup")
async def startup():
    global kafka_producer, redis_client, redis_cache

    # Initialize Kafka
    kafka_config = KafkaConfig.from_env()
    kafka_producer = KafkaProducer(kafka_config)
    logger.info("Kafka producer initialized")

    # Initialize Redis
    redis_config = RedisConfig.from_env()
    redis_client = RedisClient(redis_config)
    await redis_client.connect()
    logger.info("Redis client initialized")

    # Initialize Cache
    redis_cache = RedisCache(redis_config, default_ttl=300)
    await redis_cache.connect()
    logger.info("Redis cache initialized")

@app.on_event("shutdown")
async def shutdown():
    await kafka_producer.close()
    await redis_client.close()
    await redis_cache.close()
    logger.info("Service shutdown complete")

@app.get("/health")
async def health():
    return {"status": "healthy", "version": settings.app.version}

@app.post("/users")
async def create_user(user_data: dict):
    user_id = user_data["id"]

    # Use distributed lock
    lock = RedisLock(redis_client, f"user:{user_id}", ttl=30)

    async with lock:
        # Check cache first
        cached_user = await redis_cache.get(f"user:{user_id}")
        if cached_user:
            logger.info("User found in cache", user_id=user_id)
            return cached_user

        # Store in cache
        await redis_cache.set(f"user:{user_id}", user_data, ttl=3600)

        # Publish event
        await kafka_producer.send(
            "user-events",
            {
                "type": "user_created",
                "user_id": user_id,
                "data": user_data,
            },
        )

        logger.info("User created", user_id=user_id)

    return {"status": "created", "user_id": user_id}

@app.get("/users/{user_id}")
async def get_user(user_id: int):
    # Try cache first
    user = await redis_cache.get(f"user:{user_id}")

    if not user:
        logger.warning("User not found", user_id=user_id)
        raise HTTPException(status_code=404, detail="User not found")

    logger.info("User retrieved", user_id=user_id)
    return user

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host=settings.app.host, port=settings.app.port)
```

### 2. Kafka Consumer Service

```python
import asyncio
from common.kafka import KafkaConsumer, KafkaConfig
from common.logger import get_logger, setup_logging

setup_logging(level="INFO", use_json=True)
logger = get_logger("consumer-service")

async def process_user_event(message):
    """Process user events from Kafka."""
    logger.info(
        "Processing user event",
        topic=message["topic"],
        offset=message["offset"],
        value=message["value"],
    )

    event_data = message["value"]
    event_type = event_data.get("type")

    if event_type == "user_created":
        logger.info("User created event", user_id=event_data.get("user_id"))
        # Process user creation
    elif event_type == "user_updated":
        logger.info("User updated event", user_id=event_data.get("user_id"))
        # Process user update
    else:
        logger.warning("Unknown event type", event_type=event_type)

async def error_handler(error, message):
    """Handle consumer errors."""
    logger.error(
        "Error processing message",
        error=str(error),
        error_type=type(error).__name__,
        topic=message.get("topic"),
        offset=message.get("offset"),
    )

async def main():
    config = KafkaConfig.from_env()

    async with KafkaConsumer(config, "user-event-consumer-group") as consumer:
        logger.info("Starting consumer")
        await consumer.consume(
            topics=["user-events"],
            handler=process_user_event,
            error_handler=error_handler,
        )

if __name__ == "__main__":
    asyncio.run(main())
```

### 3. Cached API Calls

```python
from common.redis import RedisCache, RedisConfig
from common.logger import get_logger

logger = get_logger("api-service")

# Initialize cache
cache = RedisCache(RedisConfig.from_env(), default_ttl=300)

@cache.cached(ttl=600, key_prefix="user")
async def get_user_from_db(user_id: int):
    """
    Fetch user from database.
    Result will be cached for 10 minutes.
    """
    logger.info("Fetching user from database", user_id=user_id)
    # Simulate database call
    return {"id": user_id, "name": "John Doe", "email": "john@example.com"}

@cache.cached(ttl=3600, key_prefix="stats")
async def get_user_statistics(user_id: int):
    """
    Calculate user statistics.
    Result will be cached for 1 hour.
    """
    logger.info("Calculating user statistics", user_id=user_id)
    # Expensive calculation
    return {"user_id": user_id, "total_orders": 42, "lifetime_value": 1234.56}

# Usage
async def main():
    await cache.connect()

    # First call hits database
    user = await get_user_from_db(123)

    # Second call returns from cache
    user = await get_user_from_db(123)

    # Clear specific cache
    await cache.delete("user:123")

    # Clear pattern
    await cache.clear_pattern("user:*")

    await cache.close()
```

### 4. Distributed Lock for Critical Sections

```python
from common.redis import RedisClient, RedisLock, RedisConfig
from common.logger import get_logger

logger = get_logger("order-service")

async def process_order(order_id: int):
    """Process order with distributed lock."""
    config = RedisConfig.from_env()

    async with RedisClient(config) as client:
        lock = RedisLock(client, f"order:{order_id}", ttl=30)

        # Try to acquire lock with timeout
        if await lock.acquire(timeout=5.0):
            try:
                logger.info("Lock acquired, processing order", order_id=order_id)

                # Critical section - only one process can execute this
                # ... process order logic ...

                # Extend lock if needed
                await lock.extend(additional_ttl=10)

            finally:
                # Always release lock
                await lock.release()
                logger.info("Lock released", order_id=order_id)
        else:
            logger.warning("Could not acquire lock", order_id=order_id)
            raise Exception("Order is being processed by another instance")

# Or use context manager
async def process_order_v2(order_id: int):
    config = RedisConfig.from_env()

    async with RedisClient(config) as client:
        lock = RedisLock(client, f"order:{order_id}", ttl=30)

        async with lock:  # Automatically acquires and releases
            logger.info("Processing order", order_id=order_id)
            # Critical section
```

### 5. Structured Logging with Context

```python
from common.logger import get_logger, log_context, setup_logging

setup_logging(level="INFO", use_json=True)
logger = get_logger("api-service")

async def handle_request(request_id: str, user_id: int):
    """Handle request with logging context."""

    # Set context for this request
    with log_context(logger, request_id=request_id, user_id=user_id):
        logger.info("Request started")  # Includes request_id and user_id

        # All logs in this context include request_id and user_id
        logger.info("Validating input")
        logger.info("Querying database")
        logger.info("Request completed")

    # Context is removed
    logger.info("Outside context")  # No request_id or user_id

# Direct context setting
logger.set_context(service="api-service", version="1.0.0")
logger.info("Service initialized")  # Includes service and version

# Clear context
logger.clear_context()
```

### 6. Configuration Management

```python
from common.config import Settings, EnvConfig, validate_config

# Load all settings
settings = Settings.from_env(".env")

# Access nested settings
print(settings.app.name)
print(settings.app.port)
print(settings.database.get_connection_string())
print(settings.redis.host)
print(settings.kafka.bootstrap_servers)

# Type-safe environment access
env = EnvConfig(".env")

# String
api_key = env.get_str("API_KEY", required=True)

# Integer
port = env.get_int("PORT", default=8000)
max_connections = env.get_int("MAX_CONNECTIONS", default=100)

# Boolean
debug = env.get_bool("DEBUG", default=False)

# List
allowed_hosts = env.get_list("ALLOWED_HOSTS", separator=",")

# Dictionary (all vars with prefix)
db_config = env.get_dict("DB_")  # DB_HOST, DB_PORT, etc.

# Validation
config = {
    "port": 8000,
    "host": "0.0.0.0",
    "debug": True,
}

schema = {
    "port": {"required": True, "type": int, "min": 1, "max": 65535},
    "host": {"required": True, "type": str},
    "debug": {"required": False, "type": bool},
}

try:
    validate_config(config, schema)
    print("Configuration valid!")
except Exception as e:
    print(f"Configuration error: {e}")
```

## Testing Your Service

```python
import pytest
from httpx import AsyncClient
from main import app  # Your FastAPI app

@pytest.mark.asyncio
async def test_create_user():
    async with AsyncClient(app=app, base_url="http://test") as client:
        response = await client.post(
            "/users",
            json={"id": 123, "name": "Test User"},
        )
        assert response.status_code == 200
        assert response.json()["status"] == "created"

@pytest.mark.asyncio
async def test_get_user():
    async with AsyncClient(app=app, base_url="http://test") as client:
        response = await client.get("/users/123")
        assert response.status_code == 200
        assert response.json()["id"] == 123
```

## Running Your Service

```bash
# Development
uvicorn main:app --reload --host 0.0.0.0 --port 8000

# Production
uvicorn main:app --host 0.0.0.0 --port 8000 --workers 4

# With Gunicorn
gunicorn main:app -w 4 -k uvicorn.workers.UvicornWorker
```

## Docker Integration

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Copy shared packages
COPY shared/python /app/shared/python
RUN pip install -e /app/shared/python

# Copy service code
COPY services/my-service /app
RUN pip install -r requirements.txt

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

## Common Patterns

### Pattern 1: Service Initialization

```python
async def init_service():
    """Initialize all service components."""
    # Config
    settings = Settings.from_env()

    # Logging
    setup_logging(level=settings.app.log_level, use_json=True)
    logger = get_logger(settings.app.name)

    # Redis
    redis_config = RedisConfig.from_env()
    redis_client = RedisClient(redis_config)
    await redis_client.connect()

    # Kafka
    kafka_config = KafkaConfig.from_env()
    producer = KafkaProducer(kafka_config)

    logger.info(
        "Service initialized",
        name=settings.app.name,
        version=settings.app.version,
    )

    return {
        "settings": settings,
        "logger": logger,
        "redis": redis_client,
        "kafka": producer,
    }
```

### Pattern 2: Error Handling

```python
from common.kafka import KafkaProducerError
from common.redis import RedisConnectionError

async def safe_publish_event(event_data):
    """Publish event with error handling."""
    try:
        await kafka_producer.send("events", event_data)
        logger.info("Event published", event_type=event_data.get("type"))
    except KafkaProducerError as e:
        logger.error(
            "Failed to publish event",
            error=str(e),
            event_data=event_data,
            exc_info=True,
        )
        # Handle failure (retry, dead letter queue, etc.)
```

### Pattern 3: Graceful Shutdown

```python
import signal
import asyncio

shutdown_event = asyncio.Event()

def signal_handler(sig, frame):
    logger.info("Shutdown signal received")
    shutdown_event.set()

async def main():
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Initialize components
    components = await init_service()

    # Run until shutdown signal
    await shutdown_event.wait()

    # Cleanup
    await components["redis"].close()
    await components["kafka"].close()
    logger.info("Shutdown complete")

if __name__ == "__main__":
    asyncio.run(main())
```

## Troubleshooting

### Issue: Kafka Connection Error

```python
# Solution: Check bootstrap servers and network
config = KafkaConfig(
    bootstrap_servers="kafka:9092",  # Use correct host
    # Add timeout
)
```

### Issue: Redis Lock Timeout

```python
# Solution: Increase timeout or TTL
lock = RedisLock(client, "resource", ttl=60)  # Longer TTL
acquired = await lock.acquire(timeout=10.0)  # Longer timeout
```

### Issue: Logs Not Showing

```python
# Solution: Check log level
setup_logging(level="DEBUG", use_json=False)  # More verbose, plain text
```

## Additional Resources

- Full Documentation: `/Users/jk/workspace/AIPX/shared/python/README.md`
- Implementation Details: `/Users/jk/workspace/AIPX/shared/python/IMPLEMENTATION_SUMMARY.md`
- Package Source: `/Users/jk/workspace/AIPX/shared/python/common/`
