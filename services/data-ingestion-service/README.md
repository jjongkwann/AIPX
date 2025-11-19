# Data Ingestion Service

실시간 시장 데이터를 수집하고 Kafka로 전송하는 서비스입니다.

## 기능

- **KIS WebSocket 연결**: 한국투자증권 WebSocket API를 통한 실시간 데이터 수신
- **자동 재연결**: 연결 끊김 시 exponential backoff를 사용한 자동 재연결 (최대 5회)
- **토큰 관리**: Redis 캐시를 활용한 인증 토큰 자동 갱신
- **메시지 파싱**: KIS JSON 메시지를 Protobuf로 변환
- **Kafka 발행**: 종목 코드 기반 파티셔닝으로 Kafka에 데이터 전송
- **Graceful Shutdown**: SIGINT/SIGTERM 시그널 처리를 통한 안전한 종료

## 아키텍처

```
┌─────────────────┐
│  KIS WebSocket  │
│      API        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐       ┌──────────────┐
│  WebSocket      │◄──────┤     Auth     │
│    Client       │       │   Manager    │
└────────┬────────┘       └──────┬───────┘
         │                       │
         │                       ▼
         │                  ┌─────────┐
         │                  │  Redis  │
         │                  │  Cache  │
         ▼                  └─────────┘
┌─────────────────┐
│    Message      │
│     Parser      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Kafka       │
│    Producer     │
└────────┬────────┘
         │
         ▼
   ┌──────────┐
   │  Kafka   │
   │  Broker  │
   └──────────┘
```

## 프로젝트 구조

```
data-ingestion-service/
├── cmd/
│   └── server/
│       └── main.go              # 메인 엔트리포인트
├── internal/
│   ├── config/
│   │   └── config.go            # 설정 관리
│   ├── kis/
│   │   ├── auth.go              # 인증 토큰 관리
│   │   ├── client.go            # WebSocket 클라이언트
│   │   └── message_parser.go   # 메시지 파싱
│   └── producer/
│       └── market_producer.go   # Kafka 프로듀서 래퍼
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

## 설정

### 환경 변수

| 변수 | 설명 | 기본값 | 필수 |
|------|------|--------|------|
| `SERVICE_NAME` | 서비스 이름 | `data-ingestion-service` | No |
| `ENVIRONMENT` | 실행 환경 (dev/staging/prod) | `development` | No |
| `LOG_LEVEL` | 로그 레벨 (debug/info/warn/error) | `info` | No |
| `LOG_FORMAT` | 로그 포맷 (json/console) | `json` | No |
| `KIS_API_KEY` | KIS API 키 | - | **Yes** |
| `KIS_API_SECRET` | KIS API 시크릿 | - | **Yes** |
| `KIS_WEBSOCKET_URL` | KIS WebSocket URL | `ws://ops.koreainvestment.com:21000` | No |
| `KIS_API_URL` | KIS REST API URL | `https://openapi.koreainvestment.com:9443` | No |
| `KIS_RECONNECT_RETRIES` | 재연결 최대 시도 횟수 | `5` | No |
| `KIS_RECONNECT_DELAY` | 재연결 초기 대기 시간 | `5s` | No |
| `KIS_HEARTBEAT_INTERVAL` | Heartbeat 간격 | `30s` | No |
| `KIS_MESSAGE_BUFFER_SIZE` | 메시지 버퍼 크기 | `1000` | No |
| `KIS_TOKEN_CACHE_TTL` | 토큰 캐시 TTL | `24h` | No |
| `KIS_TOKEN_REFRESH_BEFORE` | 토큰 갱신 사전 시간 | `10m` | No |
| `KAFKA_BROKERS` | Kafka 브로커 주소 (쉼표 구분) | `localhost:9092` | No |
| `REDIS_HOST` | Redis 호스트 | `localhost` | No |
| `REDIS_PORT` | Redis 포트 | `6379` | No |
| `REDIS_PASSWORD` | Redis 비밀번호 | - | No |

### .env 파일 예시

```bash
# Service configuration
SERVICE_NAME=data-ingestion-service
ENVIRONMENT=development
LOG_LEVEL=debug
LOG_FORMAT=console

# KIS API credentials
KIS_API_KEY=your_api_key_here
KIS_API_SECRET=your_api_secret_here
KIS_WEBSOCKET_URL=ws://ops.koreainvestment.com:21000
KIS_API_URL=https://openapi.koreainvestment.com:9443

# KIS connection settings
KIS_RECONNECT_RETRIES=5
KIS_RECONNECT_DELAY=5s
KIS_HEARTBEAT_INTERVAL=30s
KIS_MESSAGE_BUFFER_SIZE=1000
KIS_TOKEN_CACHE_TTL=24h
KIS_TOKEN_REFRESH_BEFORE=10m

# Kafka configuration
KAFKA_BROKERS=localhost:9092

# Redis configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

## 빌드 및 실행

### 로컬 개발

```bash
# 의존성 설치
go mod download

# 빌드
go build -o data-ingestion-service ./cmd/server

# 실행
./data-ingestion-service
```

### Docker

```bash
# 이미지 빌드
docker build -t aipx/data-ingestion-service:latest .

# 컨테이너 실행
docker run -d \
  --name data-ingestion-service \
  --env-file .env \
  aipx/data-ingestion-service:latest
```

### Docker Compose

```bash
# 서비스 시작
docker-compose up -d data-ingestion-service

# 로그 확인
docker-compose logs -f data-ingestion-service

# 서비스 중지
docker-compose down
```

## Kafka 토픽

서비스는 다음 토픽에 데이터를 발행합니다:

### market.tick

체결 데이터 (실시간 가격)

**메시지 형식** (Protobuf):
```protobuf
message TickData {
  string symbol = 1;        // 종목 코드
  double price = 2;         // 현재가
  int64 volume = 3;         // 체결량
  int64 timestamp = 4;      // 체결 시간 (Unix nanoseconds)
  double change = 5;        // 전일 대비 등락
  double change_rate = 6;   // 등락률
}
```

**파티셔닝**: 종목 코드 기반

### market.orderbook

호가 데이터 (매수/매도 호가)

**메시지 형식** (Protobuf):
```protobuf
message OrderBook {
  string symbol = 1;
  repeated Level bids = 2;  // 매수 호가
  repeated Level asks = 3;  // 매도 호가
  int64 timestamp = 4;
}

message Level {
  double price = 1;
  int64 quantity = 2;
}
```

**파티셔닝**: 종목 코드 기반

## 모니터링

### 로그

서비스는 구조화된 JSON 로그를 출력합니다:

```json
{
  "level": "info",
  "service": "data-ingestion-service",
  "env": "production",
  "time": "2025-11-19T10:00:00Z",
  "message": "tick data published",
  "symbol": "005930",
  "price": 70000,
  "volume": 1000
}
```

### 메트릭

프로듀서 메트릭:
- `tick_count`: 발행된 체결 데이터 수
- `orderbook_count`: 발행된 호가 데이터 수
- `error_count`: 발행 실패 횟수

WebSocket 상태:
- `StatusDisconnected`: 연결 끊김
- `StatusConnecting`: 연결 중
- `StatusConnected`: 연결됨
- `StatusReconnecting`: 재연결 중

## 에러 처리

### WebSocket 연결 오류

- **증상**: 연결이 자주 끊김
- **해결**: `KIS_RECONNECT_RETRIES` 값을 증가시키고 `KIS_RECONNECT_DELAY`를 조정

### 토큰 만료

- **증상**: 인증 오류 발생
- **해결**: Redis 연결을 확인하고 `KIS_TOKEN_REFRESH_BEFORE` 값을 증가

### Kafka 발행 실패

- **증상**: 메시지가 Kafka에 전송되지 않음
- **해결**: Kafka 브로커 연결을 확인하고 `KAFKA_BROKERS` 설정 검증

### 메시지 버퍼 오버플로우

- **증상**: "message buffer full, dropping message" 로그
- **해결**: `KIS_MESSAGE_BUFFER_SIZE`를 증가시키거나 Kafka 프로듀서 성능 튜닝

## 성능 튜닝

### Kafka 프로듀서

기본 설정은 안정성을 우선합니다:
- `RequiredAcks`: `WaitForAll` (-1)
- `EnableIdempotence`: `true`
- `Compression`: `Snappy`

높은 처리량이 필요한 경우:
```go
producerConfig.RequiredAcks = sarama.WaitForLocal
producerConfig.EnableIdempotence = false
producerConfig.Compression = sarama.CompressionLZ4
```

### 메시지 버퍼

높은 메시지 처리량:
```bash
KIS_MESSAGE_BUFFER_SIZE=10000
```

### 로그 레벨

프로덕션 환경:
```bash
LOG_LEVEL=info
LOG_FORMAT=json
```

## 테스트

### 단위 테스트

```bash
# 전체 테스트 실행
go test ./...

# 커버리지 포함
go test -cover ./...

# 특정 패키지 테스트
go test ./internal/kis/...
```

### 통합 테스트

```bash
# Docker Compose로 인프라 시작
docker-compose up -d redis kafka

# 테스트 실행
go test -tags=integration ./...
```

## 문제 해결

### 디버그 로그 활성화

```bash
LOG_LEVEL=debug ./data-ingestion-service
```

### Redis 연결 확인

```bash
redis-cli -h localhost -p 6379 ping
```

### Kafka 연결 확인

```bash
kafka-topics.sh --bootstrap-server localhost:9092 --list
```

### WebSocket 연결 확인

```bash
# 수동 WebSocket 연결 테스트
wscat -c ws://ops.koreainvestment.com:21000
```

## 참고 자료

- [KIS API 문서](https://apiportal.koreainvestment.com/)
- [한국투자증권 OpenAPI](https://apiportal.koreainvestment.com/apiservice/)
- [Kafka Producer 최적화](https://kafka.apache.org/documentation/#producerconfigs)
- [Protobuf 가이드](https://developers.google.com/protocol-buffers)

## 라이선스

AIPX Project - Proprietary
