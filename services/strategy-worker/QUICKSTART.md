# Strategy Worker - Quick Start Guide

Get the Strategy Worker running in under 10 minutes.

## Prerequisites

- Python 3.11+
- Docker & Docker Compose
- Git

## Quick Setup

### 1. Clone and Navigate

```bash
cd services/strategy-worker
```

### 2. Start Infrastructure

Start all required services with Docker Compose:

```bash
docker-compose up -d postgres kafka redis
```

Wait for services to be healthy (about 30 seconds):

```bash
docker-compose ps
```

### 3. Setup Database

Run migrations:

```bash
docker-compose exec postgres psql -U postgres -d aipx -f /docker-entrypoint-initdb.d/001_strategy_worker.sql
```

Or manually:

```bash
psql -h localhost -U postgres -d aipx -f migrations/001_strategy_worker.sql
```

### 4. Install Python Dependencies

Using uv (recommended):

```bash
uv sync
```

Or using pip:

```bash
pip install -e .
pip install -e ".[dev]"  # For development
```

### 5. Generate gRPC Stubs

```bash
chmod +x scripts/generate_protos.sh
./scripts/generate_protos.sh
```

### 6. Configure Environment

Create `.env` file:

```bash
cat > .env << EOF
ENVIRONMENT=development
DB_PASSWORD=postgres
REDIS_PASSWORD=
CONFIG_PATH=./config/config.yaml
EOF
```

### 7. Run the Worker

```bash
python -m src.main
```

You should see:

```
{"event": "Starting Strategy Worker", "version": "0.1.0", "environment": "development"}
{"event": "Database pool connected", "host": "localhost", "database": "aipx"}
{"event": "OMS gRPC client connected", "host": "localhost", "port": 50051}
{"event": "Kafka consumer started", "topic": "strategy.approved"}
{"event": "Strategy Worker started successfully"}
```

## Testing the Worker

### 1. Produce Test Strategy Message

Create a test strategy and publish to Kafka:

```bash
# Install kafka CLI tools
pip install kafka-python

# Run test producer
python << 'EOF'
import json
from kafka import KafkaProducer

producer = KafkaProducer(
    bootstrap_servers=['localhost:29092'],
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

strategy_message = {
    "type": "strategy.approved",
    "strategy_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2024-01-01T00:00:00Z",
    "config": {
        "name": "Test Tech Strategy",
        "type": "MOMENTUM",
        "symbols": ["AAPL", "MSFT"],
        "allocation": {
            "AAPL": 0.50,
            "MSFT": 0.50
        },
        "initial_capital": 10000.00,
        "risk_params": {
            "max_position_size": 0.30,
            "stop_loss_pct": 0.05,
            "max_daily_loss": 0.05
        }
    }
}

producer.send('strategy.approved', value=strategy_message)
producer.flush()
print("✓ Strategy message sent")
EOF
```

### 2. Check Worker Logs

You should see the worker processing the strategy:

```
{"event": "Processing strategy message", "topic": "strategy.approved"}
{"event": "Starting strategy execution", "execution_id": "..."}
{"event": "Executing initial orders", "symbols": ["AAPL", "MSFT"]}
{"event": "Strategy execution started successfully"}
```

### 3. Query Database

Check execution was created:

```bash
psql -h localhost -U postgres -d aipx -c "SELECT execution_id, strategy_id, status FROM strategy_executions;"
```

Check orders:

```bash
psql -h localhost -U postgres -d aipx -c "SELECT order_id, symbol, side, quantity, status FROM execution_orders;"
```

## Development Workflow

### Running Tests

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# With coverage
pytest --cov=src --cov-report=html

# Open coverage report
open htmlcov/index.html
```

### Code Quality

```bash
# Format code
black src/ tests/

# Lint
ruff check src/ tests/

# Type check
mypy src/
```

### Hot Reload Development

Use `watchfiles` for auto-reload during development:

```bash
pip install watchfiles

watchfiles --filter python "python -m src.main" src/
```

## Docker Development

### Build and Run with Docker

```bash
# Build image
docker-compose build strategy-worker

# Run all services
docker-compose up

# View logs
docker-compose logs -f strategy-worker

# Rebuild and restart
docker-compose up -d --build strategy-worker
```

### Debug in Docker

```bash
# Execute shell in container
docker-compose exec strategy-worker sh

# View logs
docker-compose logs --tail=100 -f strategy-worker

# Check health
docker-compose exec strategy-worker python -c "import asyncio; print('healthy')"
```

## Monitoring

### Check Service Status

```bash
# View active executions
psql -h localhost -U postgres -d aipx -c "SELECT * FROM strategy_executions WHERE status = 'active';"

# View metrics
psql -h localhost -U postgres -d aipx -c "SELECT * FROM strategy_metrics;"

# View recent risk events
psql -h localhost -U postgres -d aipx -c "SELECT * FROM risk_events ORDER BY created_at DESC LIMIT 10;"
```

### Prometheus Metrics

```bash
# If monitoring is enabled
curl http://localhost:9091/metrics
```

## Common Commands

### Stop Execution

To stop an execution, you'll need to implement a control mechanism (e.g., REST API or Kafka command topic).

For now, you can update the database:

```bash
psql -h localhost -U postgres -d aipx -c "
UPDATE strategy_executions
SET status = 'stopped', stopped_at = NOW()
WHERE execution_id = 'YOUR_EXECUTION_ID';
"
```

### View Position Snapshots

```bash
psql -h localhost -U postgres -d aipx -c "
SELECT
    snapshot_id,
    execution_id,
    total_value,
    total_pnl,
    snapshot_at
FROM position_snapshots
ORDER BY snapshot_at DESC
LIMIT 10;
"
```

### Check Risk Events

```bash
psql -h localhost -U postgres -d aipx -c "
SELECT
    event_type,
    severity,
    description,
    created_at
FROM risk_events
ORDER BY created_at DESC
LIMIT 10;
"
```

## Troubleshooting

### Issue: Kafka Connection Error

```bash
# Check Kafka is running
docker-compose ps kafka

# Check Kafka logs
docker-compose logs kafka

# Test connectivity
docker-compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

### Issue: Database Connection Error

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Test connection
psql -h localhost -U postgres -d aipx -c "SELECT 1;"

# Check connection pool in config
cat config/config.yaml | grep -A 5 "database:"
```

### Issue: gRPC Connection Error

The OMS service needs to be running. For development, you can:

1. Mock the OMS service (included in docker-compose as `oms-mock`)
2. Temporarily disable gRPC calls in the code
3. Run the actual OMS service

### Issue: No Messages Consumed

```bash
# Check topic exists
docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Create topic if missing
docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 \
  --create --topic strategy.approved --partitions 3 --replication-factor 1

# Check consumer group
docker-compose exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group strategy-worker --describe
```

## Next Steps

1. **Integrate with Cognitive Service**: Connect to the actual Cognitive Service Kafka output
2. **Implement OMS Integration**: Complete gRPC integration with Order Management Service
3. **Add Market Data**: Integrate real-time market data for position monitoring
4. **Implement Control API**: Add REST API for execution control (start/stop/pause)
5. **Setup Monitoring**: Configure Prometheus and Grafana dashboards
6. **Production Deployment**: Deploy to Kubernetes or cloud platform

## Resources

- [Main README](./README.md) - Full documentation
- [Configuration Guide](./config/config.yaml) - All config options
- [Database Schema](./migrations/001_strategy_worker.sql) - Schema details
- [AIPX Documentation](../../docs/) - Overall system documentation

## Getting Help

If you encounter issues:

1. Check logs: `docker-compose logs -f strategy-worker`
2. Verify all services are healthy: `docker-compose ps`
3. Review configuration: `cat config/config.yaml`
4. Check database state: `psql -h localhost -U postgres -d aipx`
5. Open an issue on GitHub

Happy trading! 🚀
