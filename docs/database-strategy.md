# 데이터베이스 전략 (Database Strategy)

AIPX 시스템은 데이터의 특성(속도, 무결성, 보존 기간)에 따라 저장소를 삼원화하여 운영합니다.

## 1. Hot Storage: Redis (In-Memory)
초저지연(Ultra-low latency) 접근이 필요한 **일시적 상태 데이터**를 저장합니다.

-   **용도**:
    -   **시장 데이터 캐시**: 최근 1분간의 호가/체결 데이터 (기술적 지표 계산용).
    -   **토큰 관리**: KIS API Access Token (TTL 설정).
    -   **속도 제한(Rate Limit)**: 토큰 버킷 알고리즘의 카운터.
    -   **중복 제거(Deduplication)**: 최근 처리한 메시지 ID.
-   **설정**:
    -   Cluster Mode로 구성하여 고가용성 확보.
    -   AOF(Append Only File) 활성화로 장애 시 데이터 복구.

## 2. Warm Storage: PostgreSQL (RDBMS)
데이터의 **무결성(Integrity)** 과 **관계(Relationship)** 가 중요한 정형 데이터를 저장합니다.

-   **용도**:
    -   **사용자 정보**: 계정, API 키(암호화), 프로필 설정.
    -   **매매 이력(Trade History)**: 주문 생성, 체결, 정산 내역의 영구 보관.
    -   **자산 현황(Portfolio)**: 일별/월별 수익률, 현재 보유 잔고.
    -   **전략 설정(Strategy Config)**: 활성화된 전략의 파라미터.
-   **스키마 예시**:
    -   `users`: `id`, `email`, `api_key_ref`
    -   `orders`: `id`, `user_id`, `symbol`, `price`, `status`, `created_at`
    -   `trades`: `id`, `order_id`, `exec_price`, `exec_qty`, `fee`

## 3. Cold Storage: S3 & Elasticsearch
대용량의 **비정형 데이터** 및 **로그 데이터**를 저장합니다.

-   **Amazon S3 (Data Lake)**:
    -   **Raw Market Data**: 매일 수집된 틱/호가 데이터를 Parquet 형식으로 압축 저장 (머신러닝 학습용).
    -   **Model Artifacts**: 학습된 AI 모델 파일.
-   **Elasticsearch (Log & Search)**:
    -   **시스템 로그**: 모든 마이크로서비스의 애플리케이션 로그.
    -   **에이전트 추론 로그**: "왜 이 주식을 샀는가?"에 대한 LLM의 사고 과정(Chain of Thought) 저장 및 검색.

## 요약

| 저장소 | 기술 스택 | 데이터 유형 | 주요 특징 |
| :--- | :--- | :--- | :--- |
| **Hot** | Redis | 호가, 토큰, 세션 | 초고속, 휘발성 허용 |
| **Warm** | PostgreSQL | 사용자, 주문, 잔고 | 무결성, 트랜잭션, 관계형 |
| **Cold** | S3 / ES | 틱 데이터, 로그, 모델 | 대용량, 검색, 분석용 |
