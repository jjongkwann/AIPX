# 인지 서비스 (Cognitive Service)

**Cognitive Service**는 사용자의 자연어 입력을 이해하고, 투자 성향을 분석하여 맞춤형 트레이딩 전략을 생성하는 시스템의 **두뇌**입니다. LangGraph를 활용하여 상태 기반의 에이전트 워크플로우를 관리합니다.

## 🛠 주요 기능 (Features)

### 1. 사용자 프로파일링 (User Profiling)
-   **대화형 인터페이스**: 사용자와의 자연어 대화를 통해 정보를 수집합니다.
-   **Slot Filling**: `UserProfile` 스키마(리스크 성향, 자본금, 투자 기간 등)에 정의된 필수 정보를 추출합니다.
-   **능동적 질문**: 누락된 정보가 있을 경우, 에이전트가 스스로 판단하여 사용자에게 되묻습니다.

### 2. 전략 생성 (Strategy Architect)
-   **JSON 설정 생성**: 확정된 사용자 프로파일을 바탕으로 실행 가능한 `StrategyConfig.json`을 생성합니다.
-   **파라미터 최적화**: 사용자의 성향에 맞춰 기술적 지표(RSI, MACD 등)의 임계값과 자금 관리 규칙을 설정합니다.

### 3. 멀티 에이전트 협업 (Multi-Agent Collaboration)
-   **Supervisor Agent**: 전체 워크플로우를 관장하며 하위 에이전트에게 작업을 할당합니다.
-   **Analyst Agent**: 뉴스 및 거시 경제 데이터를 검색(RAG)하여 시장 상황을 분석합니다.
-   **Quant Agent**: 수치적 데이터를 기반으로 기술적 분석을 수행합니다.
-   **Risk Agent**: 생성된 전략의 예상 변동성과 리스크를 검증합니다.

## 🚀 시작하기 (Getting Started)

### 의존성 설치
```bash
uv sync
```

### 실행
```bash
uv run uvicorn main:app --reload
```
