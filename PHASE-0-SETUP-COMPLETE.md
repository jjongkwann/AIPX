# Phase 0 설정 완료 보고서

**날짜**: 2025-11-19
**작업**: AWS Terraform 인프라 코드 작성 완료

---

## ✅ 완료된 작업

### 1. Terraform 설정 파일 생성

#### 생성된 파일 목록

| 파일 | 위치 | 용도 |
|:---|:---|:---|
| `backend.tf` | `infrastructure/terraform/aws/` | S3 백엔드 및 DynamoDB 상태 잠금 설정 |
| `provider.tf` | `infrastructure/terraform/aws/` | AWS Provider 설정 및 기본 태그 |
| `variables.tf` | `infrastructure/terraform/aws/` | 모든 인프라 변수 정의 |
| `main.tf` | `infrastructure/terraform/aws/` | VPC, MSK, RDS, ElastiCache, ECS, Lambda 리소스 |
| `outputs.tf` | `infrastructure/terraform/aws/` | 리소스 엔드포인트 및 ID 출력 |
| `aws.tfvars` | `infrastructure/terraform/environments/dev/` | 개발 환경 변수 값 |
| `README.md` | `infrastructure/terraform/aws/` | 완전한 설정 가이드 |
| `setup-aws-backend.sh` | `scripts/` | S3/DynamoDB 자동 생성 스크립트 |

### 2. 주요 특징

#### backend.tf
- S3 버킷: `aipx-terraform-state`
- DynamoDB 테이블: `aipx-terraform-locks`
- 리전: `ap-northeast-2` (서울)
- 암호화 활성화

#### provider.tf
- Terraform >= 1.5.0
- AWS Provider ~> 5.0
- 모든 리소스에 자동 태그 적용 (Project, Environment, ManagedBy)

#### variables.tf (20개 변수)
- `aws_region`: AWS 리전
- `environment`: 환경 이름 (dev/staging/production)
- `vpc_cidr`: VPC CIDR 블록
- ECS Fargate: `ingestion_cpu`, `ingestion_memory`, `oms_cpu`, `oms_memory`
- MSK: `kafka_instance_type`, `kafka_broker_count`
- Redis: `redis_node_type`, `redis_num_nodes`
- RDS: `database_instance_class`, `database_allocated_storage`, `database_multi_az`
- Lambda: `lambda_memory_size`, `lambda_timeout`
- `tags`: 공통 태그

#### main.tf
1. **VPC Module** (terraform-aws-modules/vpc)
   - 3개 AZ에 걸친 퍼블릭/프라이빗 서브넷
   - NAT Gateway (dev: 단일, prod: 다중)
   - VPC Flow Logs 활성화

2. **Data Layer**
   - Amazon MSK (Kafka)
   - ElastiCache Redis
   - RDS PostgreSQL
   - S3 Data Lake

3. **Hot Path** (ECS Fargate)
   - Data Ingestion Service (WebSocket)
   - OMS (gRPC)

4. **Cold Path** (Lambda)
   - Strategy Worker
   - Cognitive Service
   - Notification Service
   - Data Recorder

#### outputs.tf (10개 출력)
- VPC: `vpc_id`, `public_subnets`, `private_subnets`
- Kafka: `kafka_bootstrap_brokers` (민감)
- Redis: `redis_endpoint` (민감)
- Database: `database_endpoint` (민감), `database_name`
- Hot Path: `oms_grpc_endpoint`, `ingestion_service_name`
- Cold Path: `strategy_worker_arn`, `cognitive_api_url`
- Storage: `data_lake_bucket`

#### dev/aws.tfvars (개발 환경 최소 스펙)
```hcl
# ECS Fargate
ingestion_cpu    = 256   # 0.25 vCPU
ingestion_memory = 512   # 512 MB
oms_cpu          = 512   # 0.5 vCPU
oms_memory       = 1024  # 1 GB

# MSK
kafka_instance_type = "kafka.t3.small"
kafka_broker_count  = 1

# Redis
redis_node_type = "cache.t3.micro"
redis_num_nodes = 1

# RDS
database_instance_class    = "db.t3.micro"
database_allocated_storage = 20
database_multi_az          = false

# Lambda
lambda_memory_size = 512
lambda_timeout     = 60
```

**예상 비용**: ~$300-400/월

#### setup-aws-backend.sh
자동화된 백엔드 설정 스크립트:
1. AWS CLI 설치 확인
2. AWS 자격 증명 확인
3. S3 버킷 생성 (암호화, 버전 관리, 퍼블릭 액세스 차단)
4. DynamoDB 테이블 생성 (PAY_PER_REQUEST)
5. 테이블 활성화 대기

---

## 🚀 다음 단계: 실제 배포

### 전제 조건 체크리스트

- [ ] AWS 계정 생성
- [ ] AWS CLI 설치
- [ ] IAM 사용자 생성 (terraform-admin)
- [ ] AWS Credentials 설정 (`aws configure`)
- [ ] Terraform 설치 (>= 1.5.0)

### Step-by-Step 배포 가이드

#### 1. AWS CLI 설치 확인

```bash
aws --version
# aws-cli/2.x.x ...
```

설치되지 않았다면:
- macOS: `brew install awscli`
- Linux: [AWS CLI 설치 가이드](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- Windows: [AWS CLI MSI Installer](https://awscli.amazonaws.com/AWSCLIV2.msi)

#### 2. AWS 자격 증명 설정

```bash
aws configure
```

입력:
```
AWS Access Key ID [None]: YOUR_ACCESS_KEY
AWS Secret Access Key [None]: YOUR_SECRET_KEY
Default region name [None]: ap-northeast-2
Default output format [None]: json
```

확인:
```bash
aws sts get-caller-identity
```

#### 3. Terraform Backend 리소스 생성

```bash
cd /Users/jk/workspace/AIPX
chmod +x scripts/setup-aws-backend.sh
./scripts/setup-aws-backend.sh
```

예상 출력:
```
🚀 AIPX AWS Terraform Backend Setup
====================================

✅ AWS CLI configured
   Account: 123456789012
   Region: ap-northeast-2

📦 Creating S3 bucket: aipx-terraform-state
   ✅ S3 bucket created

🔄 Enabling versioning on S3 bucket
   ✅ Versioning enabled

🔒 Enabling encryption on S3 bucket
   ✅ Encryption enabled (AES256)

🚫 Blocking public access on S3 bucket
   ✅ Public access blocked

🗄️  Creating DynamoDB table: aipx-terraform-locks
   ✅ DynamoDB table created
   ⏳ Waiting for table to become active...
   ✅ Table is active

✅ Terraform backend setup complete!
```

#### 4. Terraform 초기화

```bash
cd infrastructure/terraform/aws
terraform init
```

예상 출력:
```
Initializing the backend...

Successfully configured the backend "s3"! Terraform will automatically
use this backend unless the backend configuration changes.

Initializing provider plugins...
- Finding hashicorp/aws versions matching "~> 5.0"...
- Installing hashicorp/aws v5.x.x...
- Installed hashicorp/aws v5.x.x

Terraform has been successfully initialized!
```

#### 5. Terraform 실행 계획 확인

```bash
terraform plan -var-file=../environments/dev/aws.tfvars
```

출력 확인:
- 생성될 리소스 개수 확인
- 예상 비용 검토
- 에러 메시지 확인

#### 6. 인프라 배포 (주의!)

**경고**: 실제 AWS 리소스가 생성되며 비용이 발생합니다!

```bash
terraform apply -var-file=../environments/dev/aws.tfvars
```

승인:
```
Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value: yes
```

배포 시간: **약 15-20분**

#### 7. 배포 확인

```bash
# 모든 출력 확인
terraform output

# 특정 출력만 확인
terraform output vpc_id
terraform output -raw kafka_bootstrap_brokers
terraform output -raw redis_endpoint
terraform output -raw database_endpoint
```

---

## 📊 비용 분석

### 개발 환경 (dev) 월별 예상 비용

| 서비스 | 사양 | 월 비용 |
|:---|:---|---:|
| **VPC** | NAT Gateway (1개) | $32 |
| **ECS Fargate** | Data Ingestion (0.25 vCPU, 512MB) | $15 |
| **ECS Fargate** | OMS (0.5 vCPU, 1GB) | $30 |
| **Amazon MSK** | kafka.t3.small x 1 | $73 |
| **ElastiCache** | cache.t3.micro x 1 | $13 |
| **RDS PostgreSQL** | db.t3.micro (20GB) | $16 |
| **Lambda** | 512MB, 100만 요청/월 | $5 |
| **S3** | Data Lake (100GB) | $2 |
| **CloudWatch** | Logs & Monitoring | $10 |
| **데이터 전송** | 예상 | $20 |
| **총계** | | **~$216/월** |

실제 사용 패턴에 따라 **$300-400/월** 예상

### 비용 절감 팁

1. **개발 시간 외 리소스 중지**:
   ```bash
   # ECS 서비스 중지
   aws ecs update-service --cluster aipx-dev --service data-ingestion --desired-count 0
   aws ecs update-service --cluster aipx-dev --service oms --desired-count 0
   ```

2. **AWS Budgets 설정**:
   ```bash
   aws budgets create-budget \
     --account-id $(aws sts get-caller-identity --query Account --output text) \
     --budget file://budget.json
   ```

3. **불필요한 로그 삭제**:
   - CloudWatch Logs 보존 기간 설정 (7일)

---

## 🔧 트러블슈팅

### 문제 1: AWS CLI 설치 실패

**증상**:
```bash
aws --version
-bash: aws: command not found
```

**해결**:
```bash
# macOS
brew install awscli

# Linux
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

### 문제 2: AWS 자격 증명 오류

**증상**:
```
Error: error configuring Terraform AWS Provider: no valid credential sources
```

**해결**:
```bash
aws configure
# 또는
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="ap-northeast-2"
```

### 문제 3: S3 버킷 이미 존재

**증상**:
```
⚠️  S3 bucket already exists
```

**해결**: 정상입니다. 스크립트가 기존 버킷을 재사용합니다.

### 문제 4: Terraform 초기화 실패

**증상**:
```
Error: Failed to get existing workspaces: S3 bucket does not exist
```

**해결**: `scripts/setup-aws-backend.sh` 실행하여 백엔드 리소스 생성

### 문제 5: 권한 부족

**증상**:
```
Error: error creating EC2 VPC: UnauthorizedOperation
```

**해결**: IAM 사용자에 `AdministratorAccess` 정책 연결 또는 필요한 권한 부여

---

## 📝 다음 Phase 준비

Phase 0 완료 후 진행할 작업:

### Phase 1: 기초 인프라 구축 (2주)

1. **Protobuf 컴파일 파이프라인**
   - gRPC 서비스 정의
   - Python/Go 코드 생성

2. **Docker 이미지 빌드**
   - Data Ingestion Service
   - OMS
   - 각 Lambda 함수

3. **로컬 개발 환경**
   - Docker Compose
   - 로컬 Kafka, Redis, PostgreSQL

4. **CI/CD 파이프라인**
   - GitHub Actions
   - ECR 푸시
   - Lambda 배포

### Phase 2: 데이터 파이프라인 구축 (3주)

1. **Data Ingestion Service**
   - WebSocket 서버
   - Kafka Producer

2. **Kafka 토픽 설정**
   - `market-data-raw`
   - `trading-signals`
   - `order-events`

3. **Data Recorder Service**
   - Kafka Consumer (Lambda)
   - S3 Parquet 저장

---

## 🎯 현재 상태

```
✅ Phase 0: AWS 클라우드 초기 설정
   ├── ✅ Terraform 코드 작성 완료
   ├── ⏳ AWS CLI 설치 대기 (사용자 작업 필요)
   ├── ⏳ AWS 계정 설정 대기 (사용자 작업 필요)
   └── ⏳ 인프라 배포 대기

⏳ Phase 1: 기초 인프라 구축 (준비 완료)
⏳ Phase 2: 데이터 파이프라인 구축
⏳ Phase 3: 실행 레이어 구축
⏳ Phase 4: 인지 레이어 구축
⏳ Phase 5: 테스팅 및 MLOps
⏳ Phase 6: 배포 및 최적화
```

---

## 📚 참고 문서

- [infrastructure/terraform/aws/README.md](infrastructure/terraform/aws/README.md) - 상세 설정 가이드
- [docs/AWS-ARCHITECTURE.md](docs/AWS-ARCHITECTURE.md) - 전체 아키텍처 설명
- [TODO/PHASE-0-CLOUD-SETUP.md](TODO/PHASE-0-CLOUD-SETUP.md) - Phase 0 가이드
- [CLEANUP-SUMMARY.md](CLEANUP-SUMMARY.md) - 이전 정리 작업 내역

---

## 🎉 축하합니다!

Phase 0의 Terraform 코드 작성이 완료되었습니다!

이제 다음 작업을 진행하세요:

1. **AWS CLI 설치** (아직 설치 안 된 경우)
2. **AWS 계정 생성 및 자격 증명 설정**
3. **`scripts/setup-aws-backend.sh` 실행**
4. **`terraform init` 실행**
5. **`terraform plan` 으로 확인**
6. **`terraform apply` 로 배포** (비용 발생 주의!)

배포 완료 후 Phase 1을 시작할 수 있습니다.

---

**작성일**: 2025-11-19
**상태**: Phase 0 Terraform 코드 작성 완료 ✅
**다음 작업**: 사용자의 AWS 환경 설정 및 실제 배포
