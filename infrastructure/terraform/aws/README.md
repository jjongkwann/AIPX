# AIPX AWS Infrastructure - Terraform

AWS 하이브리드 서버리스 아키텍처를 위한 Terraform 인프라 코드입니다.

## 아키텍처 개요

- **Hot Path**: ECS Fargate (Data Ingestion, OMS)
- **Cold Path**: Lambda (Strategy Worker, Cognitive, Notification, Recorder)
- **Data Layer**: MSK, ElastiCache Redis, RDS PostgreSQL, S3

## 디렉터리 구조

```
aws/
├── backend.tf        # Terraform 상태 백엔드 설정 (S3 + DynamoDB)
├── provider.tf       # AWS Provider 설정
├── variables.tf      # 변수 정의
├── main.tf          # 메인 리소스 정의
├── outputs.tf       # 출력 값 정의
└── README.md        # 이 파일
```

## 사전 요구사항

### 1. AWS CLI 설치

**macOS (Homebrew)**:
```bash
brew install awscli
```

**Linux**:
```bash
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

**Windows**:
```powershell
msiexec.exe /i https://awscli.amazonaws.com/AWSCLIV2.msi
```

설치 확인:
```bash
aws --version
# aws-cli/2.x.x ...
```

### 2. AWS 계정 설정

AWS 계정이 없다면 [AWS 회원가입](https://portal.aws.amazon.com/billing/signup)

### 3. IAM 사용자 생성 (Terraform용)

AWS 콘솔에서 IAM 사용자 생성:

1. IAM > 사용자 > 사용자 추가
2. 사용자 이름: `terraform-admin`
3. 액세스 유형: 프로그래밍 방식 액세스
4. 권한: `AdministratorAccess` 정책 연결 (프로덕션에서는 최소 권한 원칙 적용)
5. 액세스 키 ID와 비밀 액세스 키 저장

### 4. AWS Credentials 설정

```bash
aws configure
```

입력 정보:
```
AWS Access Key ID: YOUR_ACCESS_KEY
AWS Secret Access Key: YOUR_SECRET_KEY
Default region name: ap-northeast-2
Default output format: json
```

설정 확인:
```bash
aws sts get-caller-identity
```

출력 예시:
```json
{
    "UserId": "AIDAXXXXXXXXXXXXXXXXX",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/terraform-admin"
}
```

### 5. Terraform 설치

**macOS (Homebrew)**:
```bash
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
```

**Linux (Ubuntu/Debian)**:
```bash
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform
```

**Windows (Chocolatey)**:
```powershell
choco install terraform
```

설치 확인:
```bash
terraform version
# Terraform v1.5.0 or later
```

## 초기 설정 (Phase 0)

### Step 1: Terraform Backend 리소스 생성

Terraform 상태 파일을 저장할 S3 버킷과 상태 잠금을 위한 DynamoDB 테이블을 생성합니다.

```bash
# 프로젝트 루트로 이동
cd /path/to/AIPX

# 백엔드 설정 스크립트 실행
chmod +x scripts/setup-aws-backend.sh
./scripts/setup-aws-backend.sh
```

스크립트가 자동으로 생성하는 리소스:
- S3 버킷: `aipx-terraform-state` (암호화, 버전 관리 활성화)
- DynamoDB 테이블: `aipx-terraform-locks` (상태 잠금용)

### Step 2: Terraform 초기화

```bash
cd infrastructure/terraform/aws
terraform init
```

예상 출력:
```
Initializing the backend...
Successfully configured the backend "s3"!
...
Terraform has been successfully initialized!
```

### Step 3: Terraform 실행 계획 확인

```bash
# 개발 환경
terraform plan -var-file=../environments/dev/aws.tfvars

# 프로덕션 환경
terraform plan -var-file=../environments/production/aws.tfvars
```

### Step 4: 인프라 배포

**주의**: 실제 AWS 리소스가 생성되며 비용이 발생합니다!

```bash
# 개발 환경 배포
terraform apply -var-file=../environments/dev/aws.tfvars

# 승인 후 yes 입력
```

배포 시간: 약 15-20분 (VPC, MSK, RDS 등 생성)

### Step 5: 배포 확인

```bash
# 출력 값 확인
terraform output

# 특정 출력 값만 확인
terraform output vpc_id
terraform output kafka_bootstrap_brokers
```

## 환경별 설정

### 개발 환경 (dev)

```bash
terraform plan -var-file=../environments/dev/aws.tfvars
terraform apply -var-file=../environments/dev/aws.tfvars
```

비용 최소화 설정:
- Kafka: `kafka.t3.small` x 1 브로커
- Redis: `cache.t3.micro` x 1 노드
- RDS: `db.t3.micro`, 단일 AZ
- ECS Fargate: 최소 스펙 (0.25 vCPU, 512MB)

예상 비용: **~$300-400/월**

### 스테이징 환경 (staging)

```bash
terraform plan -var-file=../environments/staging/aws.tfvars
terraform apply -var-file=../environments/staging/aws.tfvars
```

### 프로덕션 환경 (production)

```bash
terraform plan -var-file=../environments/production/aws.tfvars
terraform apply -var-file=../environments/production/aws.tfvars
```

고가용성 설정:
- Kafka: `kafka.m5.large` x 3 브로커 (Multi-AZ)
- Redis: `cache.r6g.large` x 6 노드 (클러스터 모드)
- RDS: `db.r6g.xlarge`, Multi-AZ
- ECS Fargate: 충분한 리소스 (1 vCPU, 2GB)

예상 비용: **~$1,500-1,800/월** (최적화 후)

## 리소스 삭제

**경고**: 모든 데이터가 영구적으로 삭제됩니다!

```bash
# 개발 환경 삭제
terraform destroy -var-file=../environments/dev/aws.tfvars

# 승인 후 yes 입력
```

## 주요 출력 값

| 출력 값 | 설명 |
|:---|:---|
| `vpc_id` | VPC ID |
| `public_subnets` | 퍼블릭 서브넷 ID 목록 |
| `private_subnets` | 프라이빗 서브넷 ID 목록 |
| `kafka_bootstrap_brokers` | MSK 브로커 엔드포인트 (민감 정보) |
| `redis_endpoint` | ElastiCache Redis 엔드포인트 (민감 정보) |
| `database_endpoint` | RDS PostgreSQL 엔드포인트 (민감 정보) |
| `oms_grpc_endpoint` | OMS gRPC 내부 엔드포인트 |
| `cognitive_api_url` | Cognitive Service API Gateway URL |
| `data_lake_bucket` | S3 Data Lake 버킷 이름 |

민감 정보 출력:
```bash
terraform output -raw kafka_bootstrap_brokers
terraform output -raw redis_endpoint
terraform output -raw database_endpoint
```

## 상태 관리

### 상태 파일 위치
- S3: `s3://aipx-terraform-state/aws/terraform.tfstate`
- 자동 암호화 및 버전 관리

### 상태 잠금
- DynamoDB 테이블: `aipx-terraform-locks`
- 동시 실행 방지

### 상태 확인
```bash
terraform state list
terraform state show module.vpc.aws_vpc.this[0]
```

## 트러블슈팅

### 1. Backend 초기화 실패

```
Error: Failed to get existing workspaces: S3 bucket does not exist.
```

**해결**: `scripts/setup-aws-backend.sh` 실행하여 S3 버킷 생성

### 2. 권한 에러

```
Error: error creating EC2 VPC: UnauthorizedOperation
```

**해결**: IAM 사용자에 충분한 권한 부여

### 3. 리소스 충돌

```
Error: error creating MSK Cluster: ConflictException
```

**해결**: 기존 리소스 확인 및 삭제 또는 다른 이름 사용

### 4. 리전 에러

```
Error: Error launching source instance: InvalidAMIID.NotFound
```

**해결**: `variables.tf`와 `aws.tfvars`에서 리전 확인

## 보안 Best Practices

### 1. Secrets 관리
- **절대 커밋 금지**: `.tfvars` 파일에 민감 정보 포함 시 `.gitignore` 추가
- **AWS Secrets Manager 사용**: RDS 비밀번호, API 키 등
- **환경 변수**: `TF_VAR_` 접두사 사용

예시:
```bash
export TF_VAR_database_password="your-secure-password"
terraform apply
```

### 2. IAM 최소 권한 원칙
개발 환경에서는 편의상 `AdministratorAccess`를 사용하지만, 프로덕션에서는 필요한 권한만 부여:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:*",
        "ecs:*",
        "rds:*",
        "elasticache:*",
        "kafka:*",
        "s3:*",
        "lambda:*"
      ],
      "Resource": "*"
    }
  ]
}
```

### 3. VPC 보안
- 프라이빗 서브넷에 민감한 리소스 배치 (RDS, ElastiCache, MSK)
- Security Group으로 최소 접근 제어
- VPC Flow Logs 활성화 (이미 main.tf에 포함됨)

### 4. 암호화
- S3: 서버 측 암호화 (AES-256)
- RDS: 저장 데이터 암호화
- 전송 중 데이터: TLS/SSL

## 비용 모니터링

### AWS Budgets 설정

```bash
aws budgets create-budget \
  --account-id $(aws sts get-caller-identity --query Account --output text) \
  --budget file://budget.json \
  --notifications-with-subscribers file://notifications.json
```

`budget.json`:
```json
{
  "BudgetName": "AIPX Monthly Budget",
  "BudgetLimit": {
    "Amount": "500",
    "Unit": "USD"
  },
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST"
}
```

### 비용 최적화 팁

1. **개발 환경 자동 종료**:
   ```bash
   # 업무 시간 외 리소스 중지
   aws ecs update-service --cluster aipx-dev --service data-ingestion --desired-count 0
   ```

2. **Spot Instances 활용**: ECS Fargate Spot 사용 (최대 70% 절감)

3. **Reserved Instances**: 프로덕션 RDS, ElastiCache (40-60% 할인)

4. **Savings Plans**: Lambda, Fargate 1년 약정 (17% 할인)

## 다음 단계

Phase 0 완료 후:

1. **Phase 1**: 기초 인프라 구축
   - Protobuf 컴파일 파이프라인
   - Docker 이미지 빌드 및 ECR 푸시
   - 로컬 개발 환경 (Docker Compose)

2. **Phase 2**: 데이터 파이프라인 구축
   - Data Ingestion Service 구현
   - Kafka 토픽 설정
   - Data Recorder Service 구현

3. **Phase 3**: 실행 레이어 구축
   - OMS 구현
   - Strategy Worker 구현

자세한 내용은 `TODO/` 디렉터리 참조

## 참고 문서

- [AWS-ARCHITECTURE.md](../../../docs/AWS-ARCHITECTURE.md) - 전체 아키텍처 상세 설명
- [PHASE-0-CLOUD-SETUP.md](../../../TODO/PHASE-0-CLOUD-SETUP.md) - Phase 0 가이드
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [AWS MSK Documentation](https://docs.aws.amazon.com/msk/)
- [AWS ECS Fargate Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html)

## 문의 및 지원

이슈나 질문이 있으면 프로젝트 이슈 트래커에 등록해주세요.
