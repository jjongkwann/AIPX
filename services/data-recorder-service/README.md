# 데이터 레코더 서비스 (Data Recorder Service)

**Data Recorder Service**는 실시간으로 흘러가는 시장 데이터를 영구 저장소에 기록하여, 추후 백테스팅과 AI 모델 학습에 사용할 수 있도록 하는 **아카이빙** 서비스입니다.

## 🛠 주요 기능 (Features)

### 1. 고속 데이터 수집
-   **Kafka Consumer Group**: `market.tick`, `market.orderbook` 토픽을 전용 컨슈머 그룹으로 구독하여 실시간 처리와 독립적으로 데이터를 수집합니다.

### 2. 시계열 데이터베이스 저장
-   **TimescaleDB / PostgreSQL**: 시계열 데이터에 최적화된 DB에 틱 데이터를 저장합니다.
-   **배치 삽입 (Batch Insert)**: DB 부하를 줄이기 위해 일정량의 데이터를 메모리에 모았다가 한 번에 저장합니다.

### 3. 콜드 스토리지 아카이빙
-   **S3 업로드**: 오래된 데이터나 대용량 틱 데이터는 Parquet 형식으로 압축하여 AWS S3와 같은 객체 스토리지로 이관합니다.

## 🚀 시작하기 (Getting Started)

### 초기화
```bash
go mod tidy
```

### 실행
```bash
go run main.go
```
