# 개발 환경 설정 가이드 (Development Setup Guide)

이 문서는 AIPX 프로젝트를 로컬 환경에서 실행하고 개발하기 위한 설정을 안내합니다.

## 1. 필수 요구 사항 (Prerequisites)

다음 도구들이 시스템에 설치되어 있어야 합니다.

### 언어 및 런타임
-   **Go**: 1.21 이상 ([설치 링크](https://go.dev/dl/))
-   **Python**: 3.11 이상 ([설치 링크](https://www.python.org/downloads/))
    -   `uv` (초고속 패키지 매니저): `curl -LsSf https://astral.sh/uv/install.sh | sh`
-   **Node.js**: 18.0 이상 (Dashboard용, [설치 링크](https://nodejs.org/))

### 인프라 및 도구
-   **Docker & Docker Compose**: 컨테이너 실행용 ([설치 링크](https://www.docker.com/products/docker-desktop/))
-   **Protoc (Protocol Buffers Compiler)**: `.proto` 파일 컴파일용
    -   Mac: `brew install protobuf`
    -   Go 플러그인:
        ```bash
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        ```

## 2. 프로젝트 초기화 (Project Initialization)

### 저장소 클론
```bash
git clone https://github.com/jjongkwann/AIPX.git
cd AIPX
```

### 마이크로서비스 초기화

#### Go 서비스 (Data Ingestion, OMS)
```bash
# Data Ingestion Service
cd services/data-ingestion-service
go mod init github.com/jjongkwann/aipx/data-ingestion
go mod tidy

# Order Management Service
cd ../order-management-service
go mod init github.com/jjongkwann/aipx/order-management
go mod tidy
```

#### Python 서비스 (Cognitive, Backtesting)
```bash
# Cognitive Service
cd services/cognitive-service
uv sync

# Backtesting Service
cd ../backtesting-service
uv sync
```

## 3. 로컬 인프라 실행 (Local Infrastructure)

프로젝트 루트에 `docker-compose.yml`을 생성하여 Kafka, Redis, PostgreSQL을 실행합니다.

```bash
docker-compose up -d
```

## 4. Protobuf 컴파일

`shared/proto`의 변경 사항을 각 서비스에 반영하려면 컴파일이 필요합니다.

```bash
# 예시: Go용 컴파일
protoc --go_out=. --go-grpc_out=. shared/proto/*.proto

# 예시: Python용 컴파일
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. shared/proto/*.proto
```
