# 배포 및 인프라 (Deployment & Infrastructure)

## 컨테이너화 (Containerization)
모든 서비스는 Docker화됩니다.
-   **Python 서비스**: `python:3.11-slim` 또는 GPU 워크로드를 위한 `nvidia/cuda` 베이스 이미지 사용.
-   **Go 서비스**: 멀티 스테이지 빌드(Multi-stage build), 최소한의 풋프린트를 위해 `distroless` 이미지 사용.

## 오케스트레이션 (Kubernetes/EKS)
-   **네임스페이스**: `aipx-prod`, `aipx-dev`.
-   **ConfigMaps/Secrets**: 환경 변수 및 KIS API 키 관리 (External Secrets).
-   **오토스케일링**: CPU/Memory 및 Kafka Lag 기반의 HPA (Horizontal Pod Autoscaler).

## CI/CD
-   **GitHub Actions**: PR 시 테스트(단위, 통합) 실행.
-   **Spinnaker**: 배포 파이프라인 관리.
    -   **카나리 배포 (Canary Deployment)**: 새 버전을 1% 트래픽에 배포하고 메트릭 모니터링 후 승격.
-   **Gitploy**: 개발자가 GitHub UI에서 배포를 트리거할 수 있도록 지원.

## 관측 가능성 (Observability)
-   **Prometheus**: Go/Python 서비스에서 메트릭 수집.
-   **Grafana**: 시스템 상태, 손익(P&L), 지연 시간(Latency) 대시보드.
-   **Loki**: 중앙 집중식 로깅.
-   **Datadog**: 분산 트레이싱 (APM).
