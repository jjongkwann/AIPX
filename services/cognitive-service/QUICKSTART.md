# Cognitive Service - Quick Start Guide

## Prerequisites

- Python 3.11+
- Docker & Docker Compose (for full stack)
- Anthropic API key or OpenAI API key

## Option 1: Docker Compose (Recommended)

The easiest way to run the complete stack:

```bash
# 1. Set your API key
export ANTHROPIC_API_KEY="your-api-key-here"

# 2. Start all services (app, postgres, kafka, redis)
docker-compose up -d

# 3. Check logs
docker-compose logs -f cognitive-service

# 4. Test the service
curl http://localhost:8001/health

# 5. Try the API
curl -X POST http://localhost:8001/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "I want to invest $100,000 in tech stocks for long-term growth"}'
```

## Option 2: Local Development

For development without Docker:

```bash
# 1. Install dependencies
uv sync
# or
pip install -e .

# 2. Set up environment
cp .env.example .env
# Edit .env with your settings

# 3. Start PostgreSQL (local or Docker)
docker run -d \
  --name postgres \
  -e POSTGRES_DB=aipx_cognitive \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15-alpine

# 4. Run migrations
psql -h localhost -U postgres -d aipx_cognitive -f migrations/001_cognitive.sql

# 5. Start Kafka (optional, for full functionality)
docker run -d \
  --name kafka \
  -p 9092:9092 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  apache/kafka:latest

# 6. Start Redis (optional)
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 7. Run the service
uvicorn src.main:app --reload --port 8001
```

## Testing the Service

### 1. Health Check
```bash
curl http://localhost:8001/health
```

### 2. Chat API (HTTP)
```bash
curl -X POST http://localhost:8001/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I want to invest $50,000. I am risk-averse and want long-term stable returns."
  }'
```

### 3. WebSocket Chat (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8001/api/v1/ws/chat/550e8400-e29b-41d4-a716-446655440000');

ws.onopen = () => {
  ws.send(JSON.stringify({
    message: "I have $100k to invest in tech stocks"
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Agent:', data.agent);
  console.log('Message:', data.content);
};
```

### 4. Complete Workflow Example

```bash
# Step 1: Start conversation
RESPONSE=$(curl -s -X POST http://localhost:8001/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I want to invest $100,000 in technology stocks"
  }')

echo $RESPONSE | jq .

# Step 2: Continue conversation (provide missing info)
SESSION_ID=$(echo $RESPONSE | jq -r .session_id)

curl -X POST http://localhost:8001/api/v1/chat \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"I'm moderate risk, planning for 3-5 years\",
    \"session_id\": \"$SESSION_ID\"
  }"

# Step 3: Approve strategy (when generated)
curl -X POST http://localhost:8001/api/v1/strategies/{strategy_id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approved": true,
    "reason": "Strategy looks good"
  }'
```

## Sample Conversation Flow

**User:** "I want to invest $100,000 in tech stocks"

**UserProfile Agent:** "Great! I understand you want to invest $100,000 in technology. What is your risk tolerance (conservative/moderate/aggressive) and investment timeline?"

**User:** "I'm moderate risk, planning for 3-5 years"

**MarketAnalyst Agent:** "Based on current market conditions and your moderate risk profile, here are my recommendations: Technology sector shows strong growth potential. Recommended stocks: AAPL, MSFT, GOOGL, NVDA."

**StrategyArchitect Agent:** "I've created a 'Tech Growth Moderate' strategy for you:
- Asset Allocation: AAPL 30%, MSFT 25%, GOOGL 25%, NVDA 20%
- Risk Parameters: Max Position 30%, Stop Loss 5%, Max Drawdown 15%
- Entry: RSI < 40, MACD bullish crossover
- Exit: RSI > 70, MACD bearish crossover, stop loss
Would you like to approve this strategy?"

**User:** "Yes, approved"

**System:** Strategy approved and published to Kafka for deployment!

## API Documentation

Once running, visit:
- Swagger UI: http://localhost:8001/docs
- ReDoc: http://localhost:8001/redoc
- Metrics: http://localhost:9090/metrics

## Running Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html

# View coverage report
open htmlcov/index.html
```

## Monitoring

### Logs
```bash
# Docker Compose
docker-compose logs -f cognitive-service

# Local
# Logs are output to stdout in JSON format
```

### Metrics
```bash
# Prometheus metrics
curl http://localhost:9090/metrics
```

### Database
```bash
# Connect to database
psql -h localhost -U postgres -d aipx_cognitive

# Check tables
\dt

# View user profiles
SELECT * FROM user_profiles;

# View strategies
SELECT strategy_id, user_id, name, status FROM strategies;
```

## Troubleshooting

### "LLM service not initialized"
- Check your ANTHROPIC_API_KEY is set correctly
- Verify API key has sufficient credits

### "Database connection failed"
- Ensure PostgreSQL is running
- Check credentials in .env match database

### "Kafka connection timeout"
- For development, you can disable Kafka
- Or start Kafka using docker-compose

### "Import errors"
- Make sure you installed dependencies: `uv sync` or `pip install -e .`
- Check Python version: `python --version` (should be 3.11+)

## Configuration

Key environment variables in `.env`:

```bash
# Required
ANTHROPIC_API_KEY=sk-ant-your-key-here

# Database
POSTGRES_HOST=localhost
POSTGRES_DB=aipx_cognitive
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your-password

# Optional (for full functionality)
KAFKA_BROKERS=localhost:9092
REDIS_HOST=localhost
```

## Next Steps

1. Try different conversation scenarios
2. Explore the generated strategies
3. Check Kafka events (if enabled)
4. Review the code in `src/`
5. Run the tests
6. Read the full README.md

## Support

- Issues: GitHub Issues
- Documentation: README.md
- Code: `src/` directory
- Tests: `tests/` directory
