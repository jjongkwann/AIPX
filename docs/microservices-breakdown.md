# 마이크로서비스 상세 명세

## 1. 인지 서비스 (Cognitive Service - `services/cognitive-service`)
-   **언어**: Python
-   **프레임워크**: LangGraph, FastAPI
-   **책임**:
    -   LangGraph 에이전트 런타임 호스팅.
    -   UI와 에이전트 간 대화를 위한 REST/WebSocket 엔드포인트 제공.
    -   대화 상태(Conversation State) 저장.
    -   `StrategyConfig` JSON 생성 및 Kafka/Database 발행.

## 2. 데이터 수집 서비스 (Data Ingestion Service - `services/data-ingestion-service`)
-   **언어**: Go
-   **프레임워크**: Native Go concurrency, Gorilla WebSocket
-   **책임**:
    -   KIS API와 WebSocket 연결 유지.
    -   인증 처리 (Redis를 통한 토큰 관리).
    -   KIS로부터 수신한 Raw JSON 메시지 파싱.
    -   Protobuf(`TickData`, `OrderBook`)로 직렬화.
    -   Kafka 토픽으로 발행 (종목 코드로 파티셔닝).

## 3. 주문 관리 서비스 (Order Management Service - `services/order-management-service`)
-   **언어**: Go
-   **프레임워크**: gRPC, Gin (REST Admin용, 선택사항)
-   **책임**:
    -   전략 엔진으로부터 gRPC 스트림으로 `OrderRequest` 수신.
    -   Risk Engine 규칙에 따라 주문 유효성 검사.
    -   속도 제한(Rate Limiting, 토큰 버킷) 적용.
    -   KIS API로 HTTP/REST 주문 요청 전송.
    -   `OrderResponse` (체결/거부)를 스트림으로 반환.

## 4. 백테스팅 서비스 (Backtesting Service - `services/backtesting-service`)
-   **언어**: Python
-   **프레임워크**: Custom Event Loop
-   **책임**:
    -   S3/DB에서 과거 데이터 재생(Replay).
    -   매칭 엔진 시뮬레이션 (지연 시간, 슬리피지 반영).
    -   샌드박스 환경에서 전략 로직 실행.
    -   성과 보고서 생성 (CAGR, MDD, Sharpe 등).

## 5. ML 추론 서비스 (ML Inference Service - `services/ml-inference-service`)
-   **언어**: Python (모델), C++ (서버)
-   **프레임워크**: NVIDIA Triton Inference Server
-   **책임**:
    -   딥러닝 모델(LSTM, Transformer) 서빙.
    -   고성능 추론 요청 처리.
    -   동적 배칭 및 앙상블 파이프라인 지원.

## 6. Dashboard Service
-   **기술 스택**: Next.js (React), TypeScript, Tailwind CSS.
-   **역할**: 사용자 인터페이스 제공.
-   **주요 기능**:
    -   **Chat Interface**: Cognitive Service와 WebSocket 통신.
    -   **Real-time Chart**: TradingView 라이브러리 연동.
    -   **Portfolio View**: 자산 현황 및 주문 내역 표시.

## 7. User Service (사용자 관리)
-   **기술 스택**: Go (Gin/Echo), PostgreSQL.
-   **역할**: 사용자 인증 및 권한 관리.
-   **주요 기능**:
    -   **회원가입/로그인**: JWT 기반 인증.
    -   **API 키 관리**: 증권사 API 키 암호화 저장.
    -   **프로필 관리**: 사용자 설정 및 선호도 저장.

## 8. Notification Service (알림)
-   **기술 스택**: Go 또는 Python.
-   **역할**: 주요 이벤트 발생 시 사용자에게 알림 발송.
-   **주요 기능**:
    -   **이벤트 구독**: Kafka `trade.orders`, `risk.alerts` 토픽 구독.
    -   **멀티 채널 발송**: Slack, Telegram, Email, SMS 등 다양한 채널 지원.
    -   **알림 설정**: 사용자별 수신 동의 및 채널 설정 적용.

## 9. Data Recorder Service (데이터 저장)
-   **기술 스택**: Go.
-   **역할**: 실시간 시장 데이터 영구 저장.
-   **주요 기능**:
    -   **고속 저장**: Kafka `market.tick` 데이터를 TimescaleDB 또는 S3(Parquet)에 배치 저장.
    -   **데이터 정합성**: 누락된 데이터 감지 및 복구 메커니즘.
    -   **쿼리 API**: 백테스팅 서비스를 위한 과거 데이터 조회 API 제공.
