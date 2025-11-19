# 데이터 흐름 (Data Flow)

## 1. 실시간 시장 데이터 흐름
1.  **KIS API**가 WebSocket 메시지(JSON)를 푸시합니다.
2.  **Ingestion Service (Go)** 가 JSON을 수신합니다.
3.  **Ingestion Service**가 파싱 후 **Protobuf** (`TickData`)로 변환합니다.
4.  **Ingestion Service**가 **Kafka** 토픽 `market.tick`에 푸시합니다 (종목별 파티셔닝).
5.  **Strategy Engine (Python)** 이 Kafka에서 데이터를 소비(Consume)합니다.
6.  **Dashboard (Next.js)** 가 Kafka 데이터를 소비하여 UI를 업데이트합니다 (WebSocket 게이트웨이 등 경유).

## 2. 주문 실행 흐름
1.  **Strategy Engine**이 매수/매도를 결정합니다.
2.  **Strategy Engine**이 `OrderRequest`를 **gRPC Stream**을 통해 **OMS (Go)** 로 전송합니다.
3.  **OMS**가 **Risk Engine**을 확인합니다 (사전 거래 점검).
4.  **OMS**가 **Rate Limiter**를 확인합니다.
5.  **OMS**가 **KIS REST API**를 호출하여 실제 주문을 넣습니다.
6.  **OMS**가 KIS로부터 응답을 받습니다.
7.  **OMS**가 `OrderResponse`를 gRPC를 통해 **Strategy Engine**으로 반환합니다.

## 3. 사용자 프로파일링 흐름
1.  **사용자**가 UI에서 채팅을 합니다.
2.  **UI**가 메시지를 **Cognitive Service (FastAPI)** 로 전송합니다.
3.  **LangGraph Agent**가 메시지를 처리하고 상태를 업데이트합니다.
4.  **Agent**가 누락된 슬롯(정보)이 있으면 추가 질문을 합니다.
5.  **Agent**가 `UserProfile`을 확정합니다.
6.  **Strategy Architect**가 `StrategyConfig.json`을 생성합니다.
7.  **시스템**이 설정에 기반하여 새로운 Strategy Worker를 배포합니다.
