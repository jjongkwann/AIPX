# Strategy Worker - AIPX Trading System

**Strategy Worker** is the execution engine of the AIPX trading system. It consumes approved trading strategies from Kafka, executes them via gRPC calls to the Order Management Service (OMS), monitors positions, and manages risk in real-time.

## Overview

The Strategy Worker acts as a bridge between the Cognitive Service (which generates strategies) and the Order Management Service (which executes orders). It implements sophisticated risk management, position monitoring, and P&L tracking to ensure safe and efficient strategy execution.

### Key Features

- **Kafka Integration**: Consumes approved strategies from `strategy.approved` topic
- **gRPC Communication**: Submits orders to OMS via gRPC streaming
- **Risk Management**: Pre-execution risk checks and position limits
- **Position Monitoring**: Real-time position tracking and P&L calculation
- **Automatic Rebalancing**: Periodic portfolio rebalancing based on strategy allocation
- **Stop Loss Management**: Automatic position closure on stop loss triggers
- **Graceful Shutdown**: Closes all positions before stopping

## Architecture

```
┌─────────────────┐
│ Cognitive       │
│ Service         │
└────────┬────────┘
         │ Kafka: strategy.approved
         ▼
┌─────────────────────────────────────────┐
│         Strategy Worker                 │
│                                         │
│  ┌──────────────┐   ┌───────────────┐  │
│  │   Kafka      │   │  Strategy     │  │
│  │   Consumer   │──▶│  Executor     │  │
│  └──────────────┘   └───────┬───────┘  │
│                             │          │
│  ┌──────────────┐   ┌───────▼───────┐  │
│  │  Risk        │◀──│  Position     │  │
│  │  Manager     │   │  Monitor      │  │
│  └──────────────┘   └───────────────┘  │
│                             │          │
│                     ┌───────▼───────┐  │
│                     │  OMS gRPC     │  │
│                     │  Client       │  │
│                     └───────┬───────┘  │
└─────────────────────────────┼─────────┘
                              │ gRPC
                              ▼
                    ┌──────────────────┐
                    │ Order Management │
                    │ Service (OMS)    │
                    └──────────────────┘
```

## Components

### 1. Strategy Consumer
- Consumes messages from Kafka `strategy.approved` topic
- Deserializes strategy configuration
- Triggers strategy execution

### 2. Strategy Executor
- Orchestrates strategy execution lifecycle
- Executes initial orders based on allocation
- Manages periodic rebalancing
- Handles position closure on stop

### 3. Risk Manager
- Pre-execution risk checks
- Position size limits
- Total exposure limits
- Daily loss limits
- Stop loss monitoring

### 4. Position Monitor
- Tracks all open positions
- Calculates realized and unrealized P&L
- Records position snapshots
- Updates trade statistics

### 5. OMS gRPC Client
- Connects to Order Management Service
- Submits orders via gRPC
- Handles order responses
- Implements retry logic with exponential backoff

### 6. Database Repository
- Persists execution state
- Records orders and fills
- Logs risk events
- Stores position snapshots

## Database Schema

### Tables

#### `strategy_executions`
Main execution tracking table:
- `execution_id`: Unique execution identifier
- `strategy_id`: Reference to strategy
- `user_id`: Owner of the strategy
- `status`: Current status (active, paused, stopped, completed, error)
- `config`: Strategy configuration (JSONB)
- `positions`: Current positions (JSONB)
- `total_pnl`, `realized_pnl`, `unrealized_pnl`: P&L metrics
- `total_trades`, `winning_trades`, `losing_trades`: Trade statistics

#### `execution_orders`
Order tracking table:
- `order_id`: Unique order identifier
- `execution_id`: Reference to execution
- `oms_order_id`: OMS order identifier
- `symbol`, `side`, `order_type`, `quantity`, `price`: Order details
- `status`: Order status (PENDING, ACCEPTED, REJECTED, FILLED, CANCELLED)
- `filled_quantity`, `filled_price`: Fill information

#### `position_snapshots`
Historical position tracking:
- `snapshot_id`: Unique snapshot identifier
- `execution_id`: Reference to execution
- `positions`: Position data (JSONB)
- `total_value`, `total_pnl`, `unrealized_pnl`: Metrics at snapshot time

#### `risk_events`
Risk event audit log:
- `event_id`: Unique event identifier
- `execution_id`: Reference to execution
- `event_type`: Type of risk event
- `severity`: INFO, WARNING, CRITICAL
- `description`: Event description
- `action_taken`: Action taken in response

## Configuration

See `config/config.yaml` for full configuration options.

### Key Settings

```yaml
kafka:
  bootstrap_servers: "localhost:9092"
  consumer_group: "strategy-worker"
  topics:
    approved: "strategy.approved"
    execution: "strategy.execution"

oms_grpc:
  host: "localhost"
  port: 50051
  timeout: 5
  retry:
    max_attempts: 3
    backoff_ms: 1000

risk:
  max_total_exposure: 1000000.00  # $1M
  max_position_size: 0.30         # 30% per symbol
  max_daily_loss: 0.05            # 5%
  max_daily_loss_amount: 50000.00 # $50k

execution:
  rebalance_interval: 300         # 5 minutes
  position_check_interval: 60     # 1 minute
```

## Kafka Message Format

### Input: `strategy.approved` Topic

```json
{
  "type": "strategy.approved",
  "strategy_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-01T00:00:00Z",
  "config": {
    "name": "Tech Growth Strategy",
    "type": "MOMENTUM",
    "symbols": ["AAPL", "MSFT", "GOOGL"],
    "allocation": {
      "AAPL": 0.35,
      "MSFT": 0.35,
      "GOOGL": 0.30
    },
    "initial_capital": 100000.00,
    "risk_params": {
      "max_position_size": 0.30,
      "stop_loss_pct": 0.05,
      "max_daily_loss": 0.05
    },
    "entry_rules": {
      "indicators": ["RSI", "MACD"],
      "rsi_threshold": 40,
      "macd_signal": "bullish"
    }
  }
}
```

### Output: `strategy.execution` Topic

```json
{
  "type": "strategy.execution.started",
  "execution_id": "770e8400-e29b-41d4-a716-446655440000",
  "strategy_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "660e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-01T00:00:00Z",
  "status": "active",
  "initial_orders": [
    {
      "symbol": "AAPL",
      "side": "BUY",
      "quantity": 200,
      "price": 175.50,
      "value": 35100.00
    }
  ]
}
```

## Risk Management

### Pre-Execution Checks

For every order, the Risk Manager performs the following checks:

1. **Order Size Limits**
   - Minimum order: $100
   - Maximum order: $100,000

2. **Position Size Limit**
   - Maximum 30% of portfolio in single symbol
   - Warning at 24% (80% of limit)

3. **Total Exposure Limit**
   - Maximum total exposure: $1M
   - Rejects orders that would exceed limit

4. **Daily Loss Limits**
   - Maximum daily loss: 5% of portfolio
   - Absolute maximum: $50,000
   - Halts trading if limit reached

5. **Stop Loss Monitoring**
   - Monitors positions against stop loss levels
   - 2% buffer to prevent whipsaws
   - Automatic position closure on trigger

### Risk Events

All risk violations are logged to the `risk_events` table with severity levels:
- **INFO**: Informational events
- **WARNING**: Approaching limits
- **CRITICAL**: Violations requiring action

## Execution Flow

### Strategy Execution Lifecycle

1. **Receive Strategy** (via Kafka)
   - Deserialize message
   - Validate required fields
   - Extract configuration

2. **Initialize Execution**
   - Create execution record in database
   - Start position monitor
   - Calculate initial orders based on allocation

3. **Execute Initial Orders**
   - For each symbol in allocation:
     - Calculate order size
     - Perform risk check
     - Submit order to OMS via gRPC
     - Update order status
     - Add position to monitor

4. **Monitor Positions**
   - Every 60 seconds:
     - Update current prices
     - Calculate P&L
     - Check stop loss levels
     - Create position snapshot

5. **Rebalance Portfolio**
   - Every 5 minutes:
     - Compare current allocation to target
     - Submit rebalancing orders if drift > 5%
     - Update positions

6. **Stop Execution** (on request or error)
   - Stop rebalancing loop
   - Close all positions
   - Stop position monitor
   - Update execution status

## Development

### Prerequisites

- Python 3.11+
- PostgreSQL 15+
- Kafka 3.0+
- Redis 7+
- Docker & Docker Compose

### Setup

```bash
# Install dependencies
pip install -e .

# Or with uv (recommended)
uv sync

# Generate gRPC stubs
./scripts/generate_protos.sh

# Run database migrations
psql -U postgres -d aipx -f migrations/001_strategy_worker.sql
```

### Environment Variables

Create a `.env` file:

```bash
ENVIRONMENT=development
DB_PASSWORD=postgres
REDIS_PASSWORD=
CONFIG_PATH=./config/config.yaml
```

### Running Locally

```bash
# Start dependencies with Docker Compose
docker-compose up -d postgres kafka redis

# Run the worker
python -m src.main
```

### Running with Docker

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f strategy-worker

# Stop services
docker-compose down
```

## Testing

### Unit Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html

# Run specific test
pytest tests/test_executor.py -v
```

### Integration Tests

```bash
# Run integration tests (requires running services)
pytest tests/integration/ -v -m integration
```

## Monitoring

### Metrics

The service exposes Prometheus metrics on port 9091:

- `strategy_executions_active`: Number of active executions
- `strategy_orders_submitted_total`: Total orders submitted
- `strategy_orders_filled_total`: Total orders filled
- `strategy_pnl_total`: Total P&L across all strategies
- `risk_violations_total`: Total risk violations

### Health Check

```bash
curl http://localhost:8080/health
```

## Production Considerations

### Scaling

- **Horizontal Scaling**: Multiple worker instances with Kafka consumer groups
- **Database Connection Pooling**: Configure pool size based on load
- **gRPC Connection Management**: Reuse connections, implement circuit breakers

### High Availability

- **Kafka Replication**: Use replicated topics
- **Database Replication**: PostgreSQL streaming replication
- **Graceful Shutdown**: Handles SIGTERM for clean termination

### Security

- **Credentials**: Use environment variables or secret management
- **TLS**: Enable TLS for gRPC and Kafka connections in production
- **Input Validation**: All strategy configs validated before execution
- **SQL Injection**: Parameterized queries via asyncpg

### Observability

- **Structured Logging**: JSON format for log aggregation
- **Distributed Tracing**: Add OpenTelemetry for request tracing
- **Alerting**: Configure alerts for risk violations and execution errors

## Troubleshooting

### Common Issues

**Consumer not receiving messages:**
- Check Kafka broker connectivity
- Verify topic exists and has messages
- Check consumer group offset

**Orders not executing:**
- Verify OMS gRPC service is running
- Check gRPC connection status
- Review risk check violations in logs

**Database connection errors:**
- Verify PostgreSQL is running
- Check connection credentials
- Review pool size settings

**High memory usage:**
- Review active execution count
- Check position snapshot frequency
- Adjust pool sizes

## Future Enhancements

- [ ] Real-time market data integration
- [ ] Advanced order types (stop-limit, trailing stop)
- [ ] Portfolio optimization algorithms
- [ ] Machine learning for rebalancing signals
- [ ] Multi-strategy execution
- [ ] Paper trading mode
- [ ] Performance attribution analysis
- [ ] WebSocket API for real-time updates

## License

MIT License - See LICENSE file

## Support

For issues or questions:
- GitHub Issues: [aipx/issues](https://github.com/aipx/issues)
- Documentation: [docs.aipx.io](https://docs.aipx.io)
- Email: support@aipx.io
