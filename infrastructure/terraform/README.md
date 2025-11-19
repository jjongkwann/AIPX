# AIPX Infrastructure as Code

이 디렉터리는 AIPX 시스템의 AWS 클라우드 인프라를 Terraform으로 관리합니다.

## 📁 디렉터리 구조

```
terraform/
├── modules/              # 재사용 가능한 Terraform 모듈
│   ├── hot-path/        # ECS Fargate 기반 저지연 서비스
│   │   └── aws/         # Data Ingestion, OMS
│   ├── cold-path/       # Lambda 기반 서버리스 서비스
│   │   └── aws/         # Strategy Worker, Cognitive, Notification, Recorder
│   └── data-layer/      # 데이터 레이어
│       ├── kafka/aws/   # Amazon MSK
│       ├── redis/aws/   # ElastiCache
│       ├── database/aws/# RDS PostgreSQL
│       └── storage/aws/ # S3
├── aws/                 # AWS 메인 구성
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
└── environments/        # 환경별 변수 파일
    ├── dev/
    │   └── aws.tfvars
    ├── staging/
    │   └── aws.tfvars
    └── production/
        └── aws.tfvars
```

## 🎯 모듈 설명

### Hot Path Module
**목적**: 저지연이 필수인 핵심 서비스 (ECS Fargate)

- **Data Ingestion Service**: WebSocket 지속 연결, KIS API 실시간 데이터 수집
- **Order Management Service**: gRPC 양방향 스트리밍, 초저지연 주문 실행

**AWS 서비스**:
- ECS Fargate (컨테이너 서버리스)
- Application Load Balancer (gRPC 지원)
- Service Discovery (Cloud Map)

### Cold Path Module
**목적**: 이벤트 기반 간헐적 실행 (Lambda 서버리스)

- **Strategy Worker**: MSK 트리거, 전략 실행 및 주문 생성
- **Cognitive Service**: API Gateway WebSocket, LangGraph 에이전트
- **Notification Service**: EventBridge 트리거, Slack/Telegram 알림
- **Data Recorder**: MSK 트리거, Parquet → S3 저장

**AWS 서비스**:
- Lambda (컨테이너 이미지)
- API Gateway (WebSocket)
- EventBridge (이벤트 라우팅)

### Data Layer Modules
**목적**: 관리형 데이터 서비스

- **Kafka**: Amazon MSK (완전 관리형 Kafka)
- **Redis**: ElastiCache (클러스터 모드, Multi-AZ)
- **PostgreSQL**: RDS (Multi-AZ, 자동 백업)
- **Object Storage**: S3 (Data Lake, Lifecycle 정책)

## 🚀 사용법

### 1. 백엔드 초기화

```bash
# S3 버킷 생성 (Terraform 상태 저장)
aws s3 mb s3://aipx-terraform-state --region ap-northeast-2
aws s3api put-bucket-versioning \
  --bucket aipx-terraform-state \
  --versioning-configuration Status=Enabled

# 암호화 활성화
aws s3api put-bucket-encryption \
  --bucket aipx-terraform-state \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "AES256"
      }
    }]
  }'

# DynamoDB 테이블 생성 (상태 잠금)
aws dynamodb create-table \
  --table-name aipx-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-2
```

### 2. Terraform 초기화 및 배포

```bash
cd infrastructure/terraform/aws

# 초기화
terraform init

# 개발 환경 배포
terraform plan -var-file=../environments/dev/aws.tfvars
terraform apply -var-file=../environments/dev/aws.tfvars

# 스테이징 환경 배포
terraform plan -var-file=../environments/staging/aws.tfvars
terraform apply -var-file=../environments/staging/aws.tfvars

# 프로덕션 환경 배포
terraform plan -var-file=../environments/production/aws.tfvars
terraform apply -var-file=../environments/production/aws.tfvars
```

### 3. 환경 전환

```bash
# 개발 → 스테이징
terraform workspace select staging
terraform apply -var-file=../environments/staging/aws.tfvars

# 스테이징 → 프로덕션
terraform workspace select production
terraform apply -var-file=../environments/production/aws.tfvars
```

### 4. 리소스 확인

```bash
# 현재 상태 확인
terraform show

# 특정 리소스 정보
terraform state show module.hot_path.aws_ecs_service.oms

# 출력 값 확인
terraform output
```

### 5. 리소스 삭제

```bash
# 개발 환경 전체 삭제
terraform destroy -var-file=../environments/dev/aws.tfvars
```

## 🔒 보안 모범 사례

### Secrets 관리
```bash
# AWS Secrets Manager
aws secretsmanager create-secret \
  --name aipx/dev/kis-credentials \
  --secret-string '{"app_key":"xxx","app_secret":"yyy"}'

# GCP Secret Manager
echo -n "your-secret" | gcloud secrets create kis-app-key --data-file=-
```

### IAM 최소 권한 원칙
- Lambda/Cloud Run에 필요한 권한만 부여
- Service Account 분리 (서비스별)
- VPC 내부 통신 (public IP 최소화)

## 💰 비용 최적화

### 개발 환경 (~$800-1,000/월)
- **ECS Fargate**: 최소 CPU/메모리 (256 CPU, 512 MB)
- **MSK**: kafka.t3.small × 1 브로커
- **RDS**: db.t3.micro, Single-AZ
- **ElastiCache**: cache.t3.micro × 1
- **Lambda**: 기본 설정 (512MB 메모리)

### 프로덕션 환경 최적화 (~$1,500-1,800/월)
- **Savings Plans**: Lambda, Fargate 1년 약정 → 17% 할인
- **Reserved Instances**: RDS, ElastiCache 1-3년 → 40-60% 할인
- **Spot Instances**: 개발/테스트 ECS 태스크 → 70% 할인
- **S3 Intelligent-Tiering**: 자동 아카이빙
- **Lambda 메모리 최적화**: AWS Compute Optimizer 권장사항
- **CloudWatch Logs**: 7일 보존 (개발), 30일 (프로덕션)

## 📊 모니터링

### Terraform Cloud (선택)
```bash
# Terraform Cloud 연동
terraform login

# 워크스페이스 생성
terraform workspace new aipx-dev
```

### Cost Explorer
- AWS Cost Explorer로 일일 비용 추적
- GCP Billing Reports로 예산 알림 설정

## 🔧 트러블슈팅

### 상태 파일 복구
```bash
# S3에서 특정 버전 복원
aws s3api list-object-versions \
  --bucket aipx-terraform-state \
  --prefix aws/terraform.tfstate

# 버전 복원
terraform state pull > backup.tfstate
```

### 리소스 Import
```bash
# 기존 AWS 리소스를 Terraform으로 가져오기
terraform import aws_ecs_cluster.aipx aipx-cluster
terraform import aws_msk_cluster.aipx arn:aws:kafka:ap-northeast-2:123456789012:cluster/aipx-kafka/...
terraform import aws_db_instance.aipx aipx-postgres
```

## 📚 참고 문서
- [AWS 하이브리드 아키텍처](../../docs/AWS-ARCHITECTURE.md)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [AWS ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [AWS Lambda Best Practices](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
