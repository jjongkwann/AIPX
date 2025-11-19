# AIPX: AI 기반 투자 성향 분석 및 고빈도 자동매매 시스템

> **AWS 하이브리드 서버리스 아키텍처**: ECS Fargate (Hot Path) + Lambda (Cold Path)로 비용 75% 절감

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Terraform](https://img.shields.io/badge/IaC-Terraform-7B42BC?logo=terraform)](https://www.terraform.io/)
[![AWS](https://img.shields.io/badge/Cloud-AWS-FF9900?logo=amazon-aws)](https://aws.amazon.com/)

---

## 📖 프로젝트 개요

**AIPX**는 투자자의 추상적인 의도를 이해하는 **인지적 유연성(Cognitive Flexibility)** 과 시장의 미세한 변동에 반응하는 **실행의 즉시성(Execution Latency)** 을 결합한 차세대 트레이딩 시스템입니다.

### 🎯 핵심 특징
- 🧠 **LangGraph 기반 에이전트**: 자연어로 투자 성향 분석 및 전략 생성
- ⚡ **하이브리드 아키텍처**: 컨테이너 + 서버리스 최적 조합
- 📊 **실시간 데이터 파이프라인**: Kafka 기반 이벤트 스트리밍
- 🚀 **초저지연 주문 실행**: gRPC 양방향 스트리밍
- 🔒 **리스크 관리**: 주문 전 실시간 검증 (팻 핑거 방지)
- 🎓 **백테스팅 엔진**: 이벤트 기반 과거 데이터 재생

---

## 🏗️ 아키텍처

### 전체 구조

```
┌─────────────────┐
│   User (Web)    │
└────────┬────────┘
         │
    ┌────▼─────┐
    │ Dashboard│◄──────┐
    └────┬─────┘       │
         │             │
    ┌────▼──────────┐  │
    │  Cognitive    │  │
    │  Service      │  │  Cold Path
    │  (Serverless) │  │  (서버리스)
    └────┬──────────┘  │
         │             │
    ┌────▼──────────┐  │
    │   Strategy    │◄─┘
    │   Worker      │
    └────┬──────────┘
         │
    ┌────▼──────────┐
    │     OMS       │
    │ (Container)   │ Hot Path
    │               │ (컨테이너)
    └────┬──────────┘
         │
    ┌────▼──────────┐
    │  KIS API      │
    └───────────────┘
```

### 레이어 구조

| 레이어 | 기술 스택 | AWS 서비스 |
|:---|:---|:---|
| **Hot Path** | Go, gRPC, WebSocket | ECS Fargate |
| **Cold Path** | Python, LangGraph | Lambda + API Gateway |
| **Event Bus** | Kafka | Amazon MSK |
| **Data Layer** | Redis, PostgreSQL, S3 | ElastiCache, RDS, S3 |

자세한 내용: [AWS 아키텍처 문서](docs/AWS-ARCHITECTURE.md)

---

## 📂 프로젝트 구조

```
AIPX/
├── docs/                          # 📚 문서
│   ├── AWS-ARCHITECTURE.md        # AWS 하이브리드 아키텍처 (최종본)
│   ├── architecture.md            # 시스템 개요 및 관계도
│   ├── api-spec.md                # API 및 Protobuf 명세
│   ├── database-strategy.md       # DB 전략
│   ├── data-flow.md               # 데이터 흐름
│   ├── development-setup.md       # 개발 환경 설정
│   ├── deployment.md              # 배포 가이드
│   └── microservices-breakdown.md # 마이크로서비스 상세
│
├── services/                      # 🚀 마이크로서비스
│   ├── data-ingestion-service/   # Go, WebSocket → Kafka
│   ├── order-management-service/ # Go, gRPC, 주문 실행
│   ├── cognitive-service/        # Python, LangGraph 에이전트
│   ├── strategy-worker/          # Python, 전략 실행
│   ├── user-service/             # Go, 인증/인가
│   ├── notification-service/     # Go/Python, 알림
│   ├── data-recorder-service/    # Go, Parquet → S3/GCS
│   ├── backtesting-service/      # Python, 이벤트 기반 백테스트
│   ├── ml-inference-service/     # Triton, AI 추론
│   └── dashboard-service/        # Next.js, 웹 UI
│
├── shared/                        # 🔗 공유 라이브러리
│   ├── proto/                    # Protobuf 정의
│   ├── go/                       # Go 공통 패키지
│   └── python/                   # Python 공통 패키지
│
├── infrastructure/                # 🛠 Infrastructure as Code
│   └── terraform/
│       ├── modules/              # 재사용 가능한 모듈
│       │   ├── hot-path/        # 컨테이너 서비스
│       │   ├── cold-path/       # 서버리스 서비스
│       │   └── data-layer/      # 데이터 레이어
│       ├── aws/                 # AWS 메인 구성
│       ├── gcp/                 # GCP 메인 구성
│       └── environments/        # 환경별 변수
│           ├── dev/
│           ├── staging/
│           └── production/
│
└── TODO/                          # ✅ 구현 로드맵
    ├── README.md                 # 전체 로드맵
    ├── PHASE-0-CLOUD-SETUP.md    # AWS 클라우드 초기 설정
    ├── PHASE-1-FOUNDATION.md     # 기초 인프라
    ├── PHASE-2-DATA-PIPELINE.md  # 데이터 파이프라인
    ├── PHASE-3-EXECUTION-LAYER.md # 실행 레이어
    ├── PHASE-4-COGNITIVE-LAYER.md # 인지 레이어
    ├── PHASE-5-TESTING-MLOPS.md  # 백테스팅 및 MLOps
    └── PHASE-6-DEPLOYMENT.md     # 배포 및 운영
```

---

## 🚀 빠른 시작

### 1. AWS 계정 설정

```bash
# AWS CLI 설치 및 구성
aws configure
# AWS Access Key ID: [YOUR_KEY]
# AWS Secret Access Key: [YOUR_SECRET]
# Default region: ap-northeast-2

# Terraform 백엔드 설정
aws s3 mb s3://aipx-terraform-state --region ap-northeast-2
aws s3api put-bucket-versioning \
  --bucket aipx-terraform-state \
  --versioning-configuration Status=Enabled
```

자세한 설정 가이드: [Phase 0: AWS 클라우드 초기 설정](TODO/PHASE-0-CLOUD-SETUP.md)

### 2. 로컬 개발 환경 구축

```bash
# 1. Protobuf 컴파일
make proto

# 2. Docker Compose로 인프라 실행
docker-compose up -d

# 3. 서비스 빌드 및 실행
cd services/data-ingestion-service
go run cmd/server/main.go
```

자세한 설정: [Phase 1: 기초 인프라](TODO/PHASE-1-FOUNDATION.md)

### 3. AWS 인프라 배포

```bash
cd infrastructure/terraform/aws

# 초기화
terraform init

# 개발 환경 배포
terraform plan -var-file=../environments/dev/aws.tfvars
terraform apply -var-file=../environments/dev/aws.tfvars

# 프로덕션 환경 배포
terraform plan -var-file=../environments/production/aws.tfvars
terraform apply -var-file=../environments/production/aws.tfvars
```

자세한 배포 가이드: [Terraform README](infrastructure/terraform/README.md)

---

## 💰 비용 예상

### 하이브리드 아키텍처 (권장)

| 클라우드 | 월 비용 (USD) | 절감률 |
|:---|---:|:---|
| **AWS** | ~$800-1,000 | 컨테이너 대비 75% 절감 |
| **GCP** | ~$600-800 | 컨테이너 대비 80% 절감 ✅ |

### 비용 최적화 팁
- 개발 환경: 최소 스펙 사용 (t3.micro, db.t3.micro)
- Spot Instances: 비프로덕션 워크로드 70% 할인
- Reserved Capacity: RDS, ElastiCache 1-3년 약정 40-60% 할인
- S3 Lifecycle: 오래된 데이터 Glacier로 자동 이동

---

## 📚 핵심 문서

### 아키텍처
- [AWS 하이브리드 아키텍처](docs/AWS-ARCHITECTURE.md) ⭐ **최종본**
- [시스템 아키텍처 개요](docs/architecture.md) - 시스템 관계도 및 핵심 철학

### 구현 가이드
- [API 명세](docs/api-spec.md) - Protobuf, Kafka 토픽, gRPC
- [데이터 흐름](docs/data-flow.md) - 실시간 데이터 및 주문 흐름
- [데이터베이스 전략](docs/database-strategy.md) - Hot/Warm/Cold 스토리지

### 로드맵
- [전체 로드맵](TODO/README.md) - Phase 0-6 구현 계획
- [Phase 0: AWS 클라우드 설정](TODO/PHASE-0-CLOUD-SETUP.md)
- [Phase 1: 기초 인프라](TODO/PHASE-1-FOUNDATION.md)

---

## 🛠 기술 스택

### Backend (Hot Path - 컨테이너)
- **Go 1.22+**: Data Ingestion, OMS, User Service
- **gRPC**: 초저지연 통신
- **WebSocket**: KIS API 실시간 연결

### Backend (Cold Path - 서버리스)
- **Python 3.11+**: Strategy Worker, Cognitive Service
- **LangGraph**: 상태 기반 에이전트 오케스트레이션
- **LangChain**: LLM 통합

### Infrastructure
- **Kafka**: 이벤트 스트리밍 (MSK / Confluent Cloud)
- **Redis**: 캐싱 및 세션 (ElastiCache / Memorystore)
- **PostgreSQL**: RDBMS (RDS / Cloud SQL)
- **S3/GCS**: 객체 스토리지

### Frontend
- **Next.js 14+**: React 기반 웹 UI
- **TradingView**: 실시간 차트
- **WebSocket**: 채팅 및 실시간 데이터

### DevOps
- **Terraform**: Infrastructure as Code
- **GitHub Actions**: CI/CD
- **Prometheus + Grafana**: 모니터링
- **ArgoCD**: GitOps (선택)

---

## 🤝 기여 가이드

현재 프로젝트는 **설계 단계**입니다. 구현을 시작하려면:

1. [Phase 0: 클라우드 환경 선택](TODO/PHASE-0-CLOUD-SETUP.md)부터 시작
2. 각 Phase의 체크리스트를 따라 구현
3. Pull Request 제출 시 [TODO](TODO/) 항목 업데이트

---

## 📄 라이선스

This project is licensed under the MIT License.

---

**Made with ❤️ by AIPX Team**
