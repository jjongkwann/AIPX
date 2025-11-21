# Cognitive Service - AI Investment Strategy Generation

**Cognitive Service** is the "brain" of the AIPX trading system. It uses LangGraph-based multi-agent workflows to understand user intent, analyze market conditions, and generate personalized investment strategies.

## Architecture

### Multi-Agent System

The service implements a supervisor-agent pattern with four specialized agents:

1. **Supervisor Agent**: Orchestrates the workflow and routes to appropriate agents
2. **UserProfile Agent**: Extracts investment profile from natural language
3. **MarketAnalyst Agent**: Analyzes market conditions and recommends sectors
4. **StrategyArchitect Agent**: Generates executable trading strategies

### LangGraph Workflow

```
Entry → Supervisor → [Conditional Routing]
                      ├─→ UserProfile Agent → Supervisor
                      ├─→ MarketAnalyst Agent → Supervisor
                      ├─→ StrategyArchitect Agent → Supervisor
                      └─→ END
```

## Features

### User Profiling
- Natural language conversation interface
- Slot filling for risk tolerance, capital, sectors, and horizon
- Automatic follow-up questions for missing information
- Profile completeness tracking

### Market Analysis
- Current market sentiment analysis (bullish/bearish/neutral)
- Sector recommendations based on market conditions
- Risk-adjusted recommendations
- Integration-ready for real-time data feeds

### Strategy Generation
- Automated strategy configuration creation
- Asset allocation optimization
- Risk parameter tuning (stop loss, drawdown, position sizing)
- Entry/exit rule definition
- Technical indicator configuration

### Multi-Channel Support
- HTTP REST API
- WebSocket real-time chat
- Session management
- Message history

## Tech Stack

- **Framework**: FastAPI + Uvicorn
- **AI/LLM**: LangChain + LangGraph with Claude 3.5 Sonnet / GPT-4
- **Database**: PostgreSQL (asyncpg)
- **Message Queue**: Kafka (aiokafka)
- **Cache**: Redis
- **Monitoring**: Prometheus metrics
- **Logging**: Structured logging (structlog)

## Quick Start

### Prerequisites

- Python 3.11+
- PostgreSQL 15+
- Kafka
- Redis
- Anthropic API key (or OpenAI API key)

### Installation

```bash
# Install dependencies with uv
uv sync

# Or with pip
pip install -e .
```

### Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
# Edit .env with your API keys and database credentials
```

### Database Setup

```bash
# Run migrations
psql -U postgres -d aipx_cognitive -f migrations/001_cognitive.sql
```

### Run the Service

```bash
# Development
uvicorn src.main:app --reload --port 8001

# Production
uvicorn src.main:app --host 0.0.0.0 --port 8001 --workers 4
```

### Using Docker Compose

```bash
# Start all services (app, postgres, kafka, redis)
docker-compose up -d

# View logs
docker-compose logs -f cognitive-service

# Stop services
docker-compose down
```

## API Documentation

### Chat Endpoints

#### HTTP Chat
```bash
POST /api/v1/chat
```

**Request:**
```json
{
  "message": "I want to invest $100,000 in tech stocks for long-term growth",
  "session_id": "optional-session-id"
}
```

**Response:**
```json
{
  "message": "Great! I understand you want to invest $100,000 in technology stocks...",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "agent": "user_profile",
  "profile_completeness": 0.66,
  "missing_fields": ["risk_tolerance"],
  "strategy_ready": false
}
```

#### WebSocket Chat
```javascript
const ws = new WebSocket('ws://localhost:8001/api/v1/ws/chat/{user_id}');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.content);
};

ws.send(JSON.stringify({
  message: "I want to invest $100k in tech"
}));
```

### Strategy Endpoints

#### Create Strategy
```bash
POST /api/v1/strategies?user_id={user_id}
```

#### Get Strategy
```bash
GET /api/v1/strategies/{strategy_id}
```

#### Approve Strategy
```bash
POST /api/v1/strategies/{strategy_id}/approve
```

**Request:**
```json
{
  "approved": true,
  "reason": "Strategy looks good"
}
```

## Project Structure

```
cognitive-service/
├── src/
│   ├── agents/              # LangGraph agent implementations
│   │   ├── user_profile_agent.py
│   │   ├── market_analyst.py
│   │   ├── strategy_architect.py
│   │   └── supervisor.py
│   ├── graph/               # LangGraph workflow
│   │   ├── state.py         # State definitions
│   │   └── workflow.py      # Workflow creation
│   ├── routers/             # FastAPI routers
│   │   ├── chat.py
│   │   └── strategy.py
│   ├── schemas/             # Pydantic models
│   │   ├── user_profile.py
│   │   ├── strategy_config.py
│   │   └── message.py
│   ├── services/            # External service integrations
│   │   ├── llm_service.py
│   │   ├── db_service.py
│   │   └── kafka_service.py
│   ├── config.py            # Configuration management
│   └── main.py              # FastAPI application
├── tests/                   # Test suite
│   ├── test_workflow.py
│   └── test_agents.py
├── migrations/              # Database migrations
│   └── 001_cognitive.sql
├── pyproject.toml           # Dependencies
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Development

### Running Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html

# Run specific test file
pytest tests/test_workflow.py -v
```

### Code Quality

```bash
# Format code
black src/ tests/

# Lint
ruff check src/ tests/

# Type checking
mypy src/
```

## Workflow Example

Here's a typical conversation flow:

**User:** "I want to invest $100,000 in technology stocks"

**Supervisor** → Routes to **UserProfile Agent**

**UserProfile Agent** → Extracts: capital=$100k, sectors=[Technology], missing: risk_tolerance, horizon

**Assistant:** "Great! I see you want to invest $100,000 in technology. What is your risk tolerance (conservative/moderate/aggressive) and investment timeline?"

**User:** "I'm moderate risk, planning for 3-5 years"

**Supervisor** → Profile complete → Routes to **MarketAnalyst Agent**

**MarketAnalyst** → Analyzes market, recommends: AAPL, MSFT, GOOGL, NVDA

**Supervisor** → Routes to **StrategyArchitect Agent**

**StrategyArchitect** → Generates strategy with asset allocation, risk params, entry/exit rules

**Assistant:** "I've created a 'Tech Growth Moderate' strategy for you:
- AAPL: 30%, MSFT: 25%, GOOGL: 25%, NVDA: 20%
- Stop Loss: 5%, Max Drawdown: 15%
- Entry: RSI < 40, MACD bullish
Would you like to approve this strategy?"

**User:** "Yes, approved"

**Supervisor** → Strategy approved → Publishes to Kafka → END

## Integration with Other Services

### Kafka Events

**Published Events:**
- `strategy.created` - When a strategy is generated
- `strategy.approved` - When user approves a strategy
- `strategy.rejected` - When user rejects a strategy

**Event Schema:**
```json
{
  "type": "strategy.approved",
  "strategy_id": "uuid",
  "user_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": { /* strategy config */ }
}
```

### Database Schema

- `user_profiles` - User investment profiles
- `chat_sessions` - Conversation sessions
- `strategies` - Generated strategy configurations
- `chat_messages` - Message history

## Monitoring

### Metrics Endpoint

```bash
curl http://localhost:9090/metrics
```

### Health Check

```bash
curl http://localhost:8001/health
```

## Environment Variables

See `.env.example` for all configuration options:

- **LLM**: API keys, model selection, parameters
- **Database**: PostgreSQL connection
- **Kafka**: Broker addresses, topics
- **Redis**: Connection settings
- **Logging**: Log level, format
- **Monitoring**: Metrics configuration

## Performance Considerations

- **Async/Await**: All I/O operations are asynchronous
- **Connection Pooling**: Database and Redis connections pooled
- **LLM Caching**: Conversation state cached in LangGraph memory
- **Structured Output**: Uses LLM structured output for reliable parsing
- **Rate Limiting**: LLM service includes retry logic with exponential backoff

## Security

- **Input Validation**: All inputs validated with Pydantic
- **SQL Injection**: Uses parameterized queries (asyncpg)
- **CORS**: Configurable allowed origins
- **API Keys**: Stored in environment variables
- **Non-root Container**: Docker runs as non-root user

## Future Enhancements

- [ ] RAG integration for market news analysis
- [ ] Real-time market data integration
- [ ] Backtesting integration before approval
- [ ] Multi-language support
- [ ] Voice interface
- [ ] Strategy performance monitoring
- [ ] A/B testing for different prompts
- [ ] User feedback loop for strategy improvement

## Contributing

1. Follow Python best practices (PEP 8, type hints)
2. Add tests for new features (>90% coverage)
3. Update documentation
4. Use structured logging
5. Handle errors gracefully

## License

MIT License - See LICENSE file

## Support

For issues or questions:
- GitHub Issues: [aipx/issues](https://github.com/aipx/issues)
- Documentation: [docs.aipx.io](https://docs.aipx.io)
- Email: support@aipx.io
