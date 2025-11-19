# 데이터 수집 서비스 (Data Ingestion Service)

**Data Ingestion Service**는 한국투자증권(KIS) API의 WebSocket 서버와 연결하여 실시간 시장 데이터를 수집하고, 이를 내부 시스템이 사용할 수 있도록 Kafka로 전송하는 **게이트웨이**입니다. Go 언어의 동시성 기능을 활용하여 고성능 처리를 보장합니다.

## 🛠 주요 기능 (Features)

### 1. 고성능 웹소켓 클라이언트
-   **Goroutine Pool**: 수천 개의 종목 데이터를 동시에 수신하더라도 지연 없이 처리할 수 있도록 경량 스레드(Goroutine)를 활용합니다.
-   **자동 재연결 (Auto-Reconnect)**: 네트워크 단절이나 서버 측 연결 종료 시, 지수 백오프(Exponential Backoff) 전략을 사용하여 자동으로 재연결을 시도합니다.

### 2. 데이터 정규화 및 직렬화
-   **JSON 파싱**: KIS API의 Raw JSON 데이터를 파싱합니다.
-   **Protobuf 변환**: 파싱된 데이터를 `TickData`, `OrderBook` Protobuf 메시지로 변환하여 데이터 크기를 줄이고 타입 안전성을 확보합니다.

### 3. Kafka 프로듀싱 (Producing)
-   **파티셔닝 (Partitioning)**: 종목 코드(`symbol`)를 키로 사용하여 동일 종목의 데이터가 항상 같은 Kafka 파티션으로 들어가도록 보장합니다 (순서 보장).
-   **백프레셔 (Backpressure) 제어**: Kafka 브로커의 부하가 높을 경우, 내부 버퍼를 활용하거나 오래된 데이터를 버리는(Conflation) 전략으로 시스템 안정성을 유지합니다.

### 4. 토큰 관리 (Token Management)
-   **Redis 연동**: KIS API 접근 토큰을 Redis에 저장하고 공유합니다.
-   **자동 갱신**: 토큰 만료 시간을 추적하여 만료 전 자동으로 갱신합니다.

## 🚀 시작하기 (Getting Started)

### 초기화
```bash
go mod tidy
```

### 실행
```bash
go run main.go
```
