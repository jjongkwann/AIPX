# Order Management Service (OMS)

Order Management Service handles order lifecycle management, risk validation, and broker integration for the AIPX trading platform.

## Overview

The OMS is responsible for:
- Receiving order requests from strategy engines via gRPC
- Validating orders against risk management rules
- Submitting orders to broker APIs (Korea Investment & Securities)
- Tracking order status and execution details
- Publishing order events to Kafka for downstream services

## Architecture

```
┌─────────────────┐
│ Strategy Engine │
└────────┬────────┘
         │ gRPC
         ▼
┌─────────────────────────────────────────┐
│     Order Management Service (OMS)      │
│                                         │
│  ┌──────────┐  ┌──────────┐           │
│  │ gRPC     │  │ Risk     │           │
│  │ Server   │─▶│ Engine   │           │
│  └──────────┘  └──────────┘           │
│       │              │                 │
│       ▼              ▼                 │
│  ┌──────────┐  ┌──────────┐           │
│  │ Rate     │  │ Order    │           │
│  │ Limiter  │◀─│ Repo     │           │
│  └──────────┘  └──────────┘           │
│       │              │                 │
│       ▼              ▼                 │
│  ┌──────────┐  ┌──────────┐           │
│  │ KIS      │  │ Kafka    │           │
│  │ Client   │  │ Producer │           │
│  └──────────┘  └──────────┘           │
└─────────────────────────────────────────┘
         │              │
         ▼              ▼
    ┌────────┐    ┌────────┐
    │ Broker │    │ Kafka  │
    │  API   │    │        │
    └────────┘    └────────┘
         │
         ▼
    ┌────────┐
    │ Postgres│
    │  (DB)  │
    └────────┘
```

## Features

### Core Functionality
- **Order Lifecycle Management**: Create, track, update, and cancel orders
- **Risk Validation**: Pre-trade risk checks to prevent fat-finger errors
- **Rate Limiting**: Token bucket algorithm to respect broker API rate limits
- **Broker Integration**: Korea Investment & Securities (KIS) REST API client
- **Event Publishing**: Kafka integration for order events and executions

### Database Schema

#### Orders Table
```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    strategy_id VARCHAR(50),
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(4) NOT NULL,  -- BUY, SELL
    order_type VARCHAR(10) NOT NULL,  -- LIMIT, MARKET
    price NUMERIC(15, 2),
    quantity INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,  -- PENDING, SENT, FILLED, REJECTED, CANCELLED
    broker_order_id VARCHAR(50),
    filled_price NUMERIC(15, 2),
    filled_quantity INTEGER,
    reject_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Order Audit Log
```sql
CREATE TABLE order_audit_log (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id),
    status VARCHAR(20) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Configuration

### Environment Variables

#### Server Configuration
- `OMS_GRPC_PORT`: gRPC server port (default: 50051)
- `OMS_HTTP_PORT`: HTTP server port for health checks (default: 8083)
- `SERVER_SHUTDOWN_TIMEOUT`: Graceful shutdown timeout (default: 30s)
- `SERVER_READ_TIMEOUT`: Request read timeout (default: 10s)
- `SERVER_WRITE_TIMEOUT`: Response write timeout (default: 10s)

#### Database Configuration
- `POSTGRES_HOST`: PostgreSQL host (default: localhost)
- `POSTGRES_PORT`: PostgreSQL port (default: 5432)
- `POSTGRES_USER`: Database user (default: aipx)
- `POSTGRES_PASSWORD`: Database password
- `POSTGRES_DB`: Database name (default: aipx)
- `POSTGRES_MAX_CONNECTIONS`: Connection pool max size (default: 20)
- `POSTGRES_MIN_CONNECTIONS`: Connection pool min size (default: 5)

#### Redis Configuration
- `REDIS_HOST`: Redis host (default: localhost)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password
- `REDIS_DB`: Redis database number (default: 0)
- `REDIS_POOL_SIZE`: Connection pool size (default: 10)

#### Kafka Configuration
- `KAFKA_BROKERS`: Comma-separated list of Kafka brokers
- `KAFKA_TOPIC_ORDER_EVENTS`: Order events topic (default: order-events)
- `KAFKA_TOPIC_EXECUTIONS`: Executions topic (default: executions)
- `KAFKA_CONSUMER_GROUP`: Consumer group ID (default: oms-consumer-group)

#### KIS API Configuration
- `KIS_BASE_URL`: KIS API base URL
- `KIS_APP_KEY`: KIS application key
- `KIS_APP_SECRET`: KIS application secret
- `KIS_ACCOUNT_NO`: Trading account number
- `KIS_RATE_LIMIT_RPS`: Rate limit in requests per second (default: 20)

#### Risk Management Configuration
- `MAX_POSITION_SIZE`: Maximum position size (default: 1000000)
- `MAX_DAILY_LOSS_PERCENT`: Maximum daily loss percentage (default: 5.0)
- `MAX_LOSS_PER_TRADE_PERCENT`: Maximum loss per trade (default: 2.0)
- `MAX_ORDERS_PER_MINUTE`: Order rate limit per user (default: 60)
- `MAX_ORDERS_PER_DAY`: Daily order limit per user (default: 500)
- `ENABLE_RISK_CHECKS`: Enable/disable risk checks (default: true)

#### Logging Configuration
- `LOG_LEVEL`: Log level (debug, info, warn, error) (default: info)
- `LOG_FORMAT`: Log format (json, console) (default: json)
- `LOG_OUTPUT`: Log output (stdout, stderr, file path) (default: stdout)

## Getting Started

### Prerequisites
- Go 1.24 or higher
- PostgreSQL 15+
- Redis 7+
- Kafka 3.5+

### Installation

1. Clone the repository:
```bash
cd services/order-management-service
```

2. Install dependencies:
```bash
go mod download
```

3. Run database migrations:
```bash
psql -U aipx -d aipx -f migrations/001_orders.sql
```

### Running the Service

#### Development Mode
```bash
go run cmd/server/main.go
```

#### Production Build
```bash
go build -o bin/oms cmd/server/main.go
./bin/oms
```

### Testing

Run unit tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

## Project Structure

```
order-management-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Service configuration
│   ├── grpc/
│   │   └── server.go            # gRPC server (TODO: T4)
│   ├── risk/
│   │   └── engine.go            # Risk validation engine (TODO: T4)
│   ├── ratelimit/
│   │   └── limiter.go           # Rate limiting (TODO: T4)
│   ├── broker/
│   │   └── kis_client.go        # KIS broker client (TODO: T4)
│   └── repository/
│       └── order_repo.go        # Order data access layer
├── migrations/
│   └── 001_orders.sql           # Database schema
├── go.mod                        # Go module definition
├── go.sum                        # Dependency checksums
└── README.md                     # This file
```

## Development Status

### Phase 1: Basic Infrastructure (Current - T1) ✅
- [x] Project structure setup
- [x] Database schema and migrations
- [x] Order repository with pgx
- [x] Service configuration
- [x] Main application with graceful shutdown
- [x] Go module initialization

### Phase 1: Implementation (Next - T4) 🚧
- [ ] gRPC server implementation
- [ ] Risk Engine implementation
- [ ] Rate Limiter implementation
- [ ] KIS Broker Client implementation
- [ ] Kafka event publishing
- [ ] Integration tests

## API Reference

### gRPC Service (To be implemented in T4)

```protobuf
service OrderService {
  rpc CreateOrder(OrderRequest) returns (OrderResponse);
  rpc GetOrder(GetOrderRequest) returns (OrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (OrderResponse);
  rpc GetUserOrders(GetUserOrdersRequest) returns (OrderListResponse);
}
```

## Contributing

Please follow Go best practices and coding standards:
- Use `gofmt` for code formatting
- Write unit tests for new functionality
- Document exported functions with godoc comments
- Handle errors explicitly

## License

Copyright (c) 2024 AIPX
