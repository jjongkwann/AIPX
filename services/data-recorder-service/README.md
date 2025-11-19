# Data Recorder Service

고성능 시장 데이터 수집 및 저장 서비스입니다. Kafka에서 시장 데이터를 실시간으로 소비하여 TimescaleDB(핫 패스)와 S3(콜드 패스)에 동시에 저장합니다.

## 주요 기능

### 데이터 처리
- **실시간 Kafka 소비**: `market.tick`, `market.orderbook` 토픽 구독
- **듀얼 저장 경로**:
  - 핫 패스: TimescaleDB (실시간 쿼리용)
  - 콜드 패스: S3 Parquet (장기 보관 및 분석용)
- **배치 버퍼링**: 메모리 효율적인 배치 처리
- **자동 플러시**: 크기 또는 시간 기준 자동 플러시

### 성능 최적화
- **병렬 처리**: 다중 consumer 고루틴
- **연결 풀링**: TimescaleDB pgx 연결 풀
- **COPY 명령**: PostgreSQL COPY를 이용한 대량 삽입
- **Parquet 압축**: Snappy 압축으로 S3 저장 공간 절약
- **시간 기반 파티셔닝**: S3에 year/month/day/hour 파티션

### 안정성
- **재시도 로직**: 실패 시 자동 재시도
- **그레이스풀 셧다운**: 종료 시 모든 버퍼 플러시
- **헬스체크 엔드포인트**: `/health`, `/metrics`
- **구조화된 로깅**: zerolog를 이용한 JSON 로깅

## 아키텍처

```
Kafka (market.tick, market.orderbook)
    │
    ▼
Market Consumer
    │
    ├─▶ Tick Buffer ──▶ TimescaleDB (tick_data)
    │                      │
    │                      └─▶ Continuous Aggregates (1m, 5m, 1h)
    │
    ├─▶ OrderBook Buffer ──▶ TimescaleDB (orderbook)
    │
    ├─▶ S3 Tick Buffer ──▶ S3 (parquet/tick/year=.../data.parquet)
    │
    └─▶ S3 OrderBook Buffer ──▶ S3 (parquet/orderbook/year=.../data.parquet)
```

## 설치 및 실행

### 사전 요구사항
- Go 1.22+
- TimescaleDB (PostgreSQL 확장)
- Kafka
- AWS S3 (선택사항)

### 환경 변수 설정

`.env.example`을 `.env`로 복사하고 수정:

```bash
cp .env.example .env
```

주요 환경 변수:

```bash
# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=data-recorder-group
KAFKA_TOPICS=market.tick,market.orderbook

# TimescaleDB
TIMESCALEDB_HOST=localhost
TIMESCALEDB_PORT=5433
TIMESCALEDB_USER=aipx
TIMESCALEDB_PASSWORD=your_password
TIMESCALEDB_DB=aipx_timeseries

# S3
S3_BUCKET=aipx-data-lake
S3_REGION=ap-northeast-2
S3_WRITE_ENABLED=true

# 배치 설정
BATCH_SIZE=1000              # DB 배치 크기
BATCH_TIMEOUT_MS=5000        # DB 배치 타임아웃 (5초)
S3_BATCH_SIZE=100000         # S3 배치 크기 (100k)
S3_BATCH_TIMEOUT_SEC=3600    # S3 배치 타임아웃 (1시간)
```

### 데이터베이스 초기화

```bash
# TimescaleDB 접속
psql -h localhost -p 5433 -U aipx -d aipx_timeseries

# 마이그레이션 실행
\i migrations/001_create_tables.sql
```

### 로컬 실행

```bash
# 의존성 설치
go mod download

# 실행
go run cmd/server/main.go
```

### Docker 실행

```bash
# 이미지 빌드
docker build -t data-recorder-service .

# 컨테이너 실행
docker run -d \
  --name data-recorder \
  --env-file .env \
  -p 8080:8080 \
  data-recorder-service
```

### Docker Compose 실행

전체 AIPX 스택과 함께 실행:

```bash
cd ../../
docker-compose up -d data-recorder-service
```

## API 엔드포인트

### Health Check
```bash
curl http://localhost:8080/health
```

응답:
```
OK
consumer: healthy
timescaledb: healthy
s3: healthy
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

응답:
```
# Consumer Metrics
total_messages: 150234
total_errors: 3
tick_count: 120450
orderbook_count: 29784

# Buffer Metrics
tick_buffer_size: 234
orderbook_buffer_size: 45

# TimescaleDB Metrics
timescale_inserts: 149876
timescale_errors: 0

# S3 Metrics
s3_uploads: 12
s3_errors: 0
s3_bytes: 45678901
```

## 성능 튜닝

### 처리량 증가

```bash
# Consumer 수 증가
NUM_CONSUMERS=8

# DB 워커 증가
DB_WRITE_WORKERS=8

# 배치 크기 증가
BATCH_SIZE=2000
```

### 메모리 사용량 감소

```bash
# 버퍼 크기 제한
MAX_BUFFER_SIZE=50000

# 배치 크기 감소
BATCH_SIZE=500
```

### S3 비용 최적화

```bash
# S3 배치 크기 증가 (더 적은 파일)
S3_BATCH_SIZE=200000

# S3 배치 타임아웃 증가
S3_BATCH_TIMEOUT_SEC=7200  # 2시간
```

## TimescaleDB 쿼리 예제

### 최근 틱 데이터 조회
```sql
SELECT * FROM tick_data
WHERE symbol = '005930'
  AND time >= NOW() - INTERVAL '1 hour'
ORDER BY time DESC
LIMIT 100;
```

### 1분 OHLCV 조회
```sql
SELECT * FROM tick_1min
WHERE symbol = '005930'
  AND bucket >= NOW() - INTERVAL '1 day'
ORDER BY bucket DESC;
```

### 호가창 조회
```sql
SELECT
  time,
  symbol,
  bids->0->>'price' AS best_bid_price,
  asks->0->>'price' AS best_ask_price
FROM orderbook
WHERE symbol = '005930'
  AND time >= NOW() - INTERVAL '5 minutes'
ORDER BY time DESC;
```

## S3 데이터 구조

### 파티션 구조
```
s3://bucket-name/
  market-data/
    tick/
      year=2025/
        month=11/
          day=19/
            hour=16/
              data_20251119161234.parquet
              data_20251119162345.parquet
    orderbook/
      year=2025/
        month=11/
          day=19/
            hour=16/
              data_20251119161234.parquet
```

### Parquet 스키마

**Tick Data:**
- timestamp: INT64 (nanoseconds)
- symbol: STRING
- price: DOUBLE
- volume: INT64
- change: DOUBLE
- change_rate: DOUBLE

**OrderBook:**
- timestamp: INT64 (nanoseconds)
- symbol: STRING
- bids: STRING (JSON)
- asks: STRING (JSON)

## 모니터링

### 로그 확인
```bash
# Docker 로그
docker logs -f data-recorder

# 로컬 실행 로그
# JSON 포맷으로 stdout에 출력
```

### 메트릭 수집

Prometheus를 사용한 메트릭 수집:

```yaml
scrape_configs:
  - job_name: 'data-recorder'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## 트러블슈팅

### 메시지 처리 지연

1. Consumer 수 증가
2. 배치 크기 증가
3. DB 연결 수 증가

### 메모리 부족

1. 버퍼 크기 제한 설정
2. 배치 크기 감소
3. S3 플러시 주기 단축

### DB 연결 실패

1. TimescaleDB 연결 정보 확인
2. 방화벽 설정 확인
3. 연결 수 제한 확인

### S3 업로드 실패

1. AWS 자격 증명 확인
2. 버킷 권한 확인
3. 네트워크 연결 확인

## 개발

### 테스트 실행
```bash
go test ./...
```

### 벤치마크
```bash
go test -bench=. ./internal/buffer/...
```

### 린팅
```bash
golangci-lint run
```

## 라이센스

MIT License
