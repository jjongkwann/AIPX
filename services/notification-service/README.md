# Notification Service

Notification Service는 AIPX 플랫폼의 다채널 알림 전송 서비스입니다. Kafka 이벤트를 구독하여 사용자 선호도에 따라 Slack, Telegram, Email 등의 채널로 알림을 전송합니다.

## Features

- **Multi-Channel Support**: Slack, Telegram, Email 채널 지원
- **Event-Driven Architecture**: Kafka 기반 비동기 이벤트 처리
- **Template-Based Messages**: 템플릿 엔진을 통한 메시지 커스터마이징
- **User Preferences**: 사용자별 채널 선호도 관리
- **Notification History**: 전송 이력 추적 및 감사
- **Retry Logic**: 실패한 알림 자동 재시도
- **Rate Limiting**: 채널별 전송 속도 제한

## Architecture

```
┌──────────────┐
│ Kafka Topics │
│ - trade.orders│
│ - risk.alerts│
│ - system.alerts│
└──────┬───────┘
       │
       v
┌──────────────────────────────┐
│ Notification Consumer        │
│ - Event Processing           │
│ - Template Rendering         │
│ - Channel Routing            │
└──────┬───────────────────────┘
       │
       v
┌──────────────────────────────┐
│ Notification Channels        │
│ - Slack (Webhook/Bot)        │
│ - Telegram (Bot API)         │
│ - Email (SMTP)               │
└──────┬───────────────────────┘
       │
       v
┌──────────────────────────────┐
│ PostgreSQL                   │
│ - User Preferences           │
│ - Notification History       │
└──────────────────────────────┘
```

## Supported Channels

### 1. Slack
- Webhook URL 기반 메시지 전송
- Block Kit 형식 지원 (TODO: Phase 2)
- Bot OAuth 지원 (TODO: Phase 2)

### 2. Telegram
- Bot API 기반 메시지 전송
- Markdown/HTML 포맷 지원
- Inline Keyboard 지원 (TODO: Phase 2)

### 3. Email
- SMTP 프로토콜 지원
- HTML/Plain Text 멀티파트 메시지
- TLS/StartTLS 암호화 지원

## Event Types

| Event Type | Description | Source Topic |
|------------|-------------|--------------|
| `order_filled` | 주문 체결 완료 | trade.orders |
| `order_rejected` | 주문 거부 | trade.orders |
| `order_cancelled` | 주문 취소 | trade.orders |
| `risk_alert` | 리스크 경고 | risk.alerts |
| `position_opened` | 포지션 오픈 | trade.orders |
| `position_closed` | 포지션 종료 | trade.orders |
| `system_alert` | 시스템 알림 | system.alerts |
| `maintenance_notice` | 시스템 점검 | system.alerts |

## Database Schema

### notification_preferences
사용자의 알림 채널 선호도 설정
```sql
- id: UUID (PK)
- user_id: UUID
- channel: VARCHAR(20) [slack, telegram, email]
- enabled: BOOLEAN
- config: JSONB (channel-specific config)
- created_at, updated_at: TIMESTAMPTZ
```

### notification_history
알림 전송 이력 및 감사 로그
```sql
- id: UUID (PK)
- user_id: UUID
- channel: VARCHAR(20)
- event_type: VARCHAR(50)
- title, message: TEXT
- payload: JSONB
- status: VARCHAR(20) [sent, failed, pending, retry]
- error_message: TEXT
- retry_count: INT
- sent_at, created_at: TIMESTAMPTZ
```

## Configuration

### Environment Variables

#### Kafka
```bash
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=notification-service
KAFKA_TOPIC_TRADE_ORDERS=trade.orders
KAFKA_TOPIC_RISK_ALERTS=risk.alerts
KAFKA_TOPIC_SYSTEM_ALERTS=system.alerts
```

#### Database
```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=aipx
POSTGRES_PASSWORD=aipx_dev_password
POSTGRES_DB=aipx
POSTGRES_MAX_CONNECTIONS=20
```

#### Slack
```bash
SLACK_ENABLED=true
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_SIGNING_SECRET=your-signing-secret
```

#### Telegram
```bash
TELEGRAM_ENABLED=true
TELEGRAM_BOT_TOKEN=your:bot:token
TELEGRAM_PARSE_MODE=Markdown
```

#### Email
```bash
EMAIL_ENABLED=true
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM_EMAIL=noreply@aipx.io
EMAIL_FROM_NAME=AIPX Trading Platform
SMTP_USE_STARTTLS=true
```

## Templates

템플릿 파일은 `internal/templates/*.tmpl` 또는 `templates/*.tmpl`에 위치합니다.

### Template Format
```
Title
---
Message body with {{.variables}}
```

### Available Functions
- `formatFloat`: 소수점 2자리 포맷팅
- `formatPercent`: 퍼센트 포맷팅
- `upper/lower`: 대소문자 변환
- `title`: 타이틀 케이스 변환

### Example Template (order_filled.tmpl)
```
Order Filled: {{.symbol}} {{.side}}
---
Your {{.side}} order for {{.symbol}} has been filled.

Order Details:
- Symbol: {{.symbol}}
- Side: {{upper .side}}
- Quantity: {{.quantity}}
- Price: {{formatFloat .price}}
```

## Development

### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Kafka 3.0+

### Setup
```bash
cd services/notification-service

# Install dependencies
go mod download

# Run migrations
psql -h localhost -U aipx -d aipx -f migrations/001_notifications.sql

# Build
go build -o bin/notification-service cmd/server/main.go

# Run
./bin/notification-service
```

### Testing
```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/consumer
```

## Deployment

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o notification-service cmd/server/main.go

FROM alpine:latest
COPY --from=builder /app/notification-service /usr/local/bin/
CMD ["notification-service"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: notification-service
        image: aipx/notification-service:latest
        env:
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        - name: POSTGRES_HOST
          value: "postgres"
```

## Monitoring

### Metrics (TODO: Phase 2)
- `notifications_sent_total`: 전송된 알림 수
- `notifications_failed_total`: 실패한 알림 수
- `notification_duration_seconds`: 전송 소요 시간
- `channel_health_status`: 채널 헬스 상태

### Logs
```json
{
  "level": "info",
  "service": "notification-service",
  "channel": "slack",
  "user_id": "uuid",
  "event_type": "order_filled",
  "msg": "Notification sent successfully"
}
```

## Phase 1 Status (Current)

- [x] Project structure setup
- [x] Database schema and migrations
- [x] Channel interface definition
- [x] Stub implementations (Slack, Telegram, Email)
- [x] Kafka consumer implementation
- [x] Template engine
- [x] Repository layer
- [x] Configuration management
- [x] Main server placeholder

## Phase 2 Roadmap (T6)

- [ ] Slack webhook integration
- [ ] Telegram Bot API integration
- [ ] SMTP email integration
- [ ] Rate limiting implementation
- [ ] Retry queue with DLQ
- [ ] Health check endpoints
- [ ] Prometheus metrics
- [ ] User preference API
- [ ] Notification history API
- [ ] Unit and integration tests

## API Endpoints (TODO: Phase 2)

### User Preferences
```
GET    /api/v1/preferences/:user_id
POST   /api/v1/preferences
PUT    /api/v1/preferences/:id
DELETE /api/v1/preferences/:id
```

### Notification History
```
GET    /api/v1/history/:user_id
GET    /api/v1/history/:user_id/:notification_id
```

### Health Check
```
GET    /health
GET    /health/channels
GET    /metrics
```

## Contributing

1. Feature 브랜치 생성
2. 변경사항 커밋
3. 테스트 실행 및 통과
4. Pull Request 생성

## License

Copyright © 2025 AIPX. All rights reserved.
