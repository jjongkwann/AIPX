# User Service

사용자 인증 및 보안 관리를 담당하는 AIPX 플랫폼의 핵심 서비스입니다.

## 목적

- 사용자 계정 관리 (등록, 로그인, 프로필 관리)
- JWT 기반 인증 및 권한 부여
- 증권사 API 키 암호화 저장
- 보안 감사 로그 기록
- 비밀번호 보안 (Argon2id)

## 보안 아키텍처

### 1. 비밀번호 보안 (Argon2id)

```
Algorithm: Argon2id
Memory: 64 MB
Iterations: 3
Parallelism: 2
Salt: 16 bytes (random)
Key Length: 32 bytes
```

**특징:**
- OWASP 권장 알고리즘
- GPU/ASIC 공격 저항성
- 타이밍 공격 방지 (constant-time comparison)
- 랜덤 솔트 사용

### 2. API 키 암호화 (AES-256-GCM)

```
Algorithm: AES-256-GCM
Key Size: 256 bits
Nonce: 12 bytes (random per encryption)
Authentication: Built-in (GCM mode)
```

**특징:**
- 인증된 암호화 (AEAD)
- 데이터 무결성 보장
- 키 로테이션 지원
- 마스터 키는 환경 변수로 관리

### 3. JWT 토큰

```
Access Token: 15분 유효
Refresh Token: 7일 유효
Algorithm: HS256
```

**특징:**
- 상태 비저장 인증
- Refresh Token DB 저장 및 취소 가능
- IP 주소 및 User-Agent 추적

## 데이터베이스 스키마

### Users 테이블
```sql
- id (UUID, PK)
- email (VARCHAR, UNIQUE)
- password_hash (VARCHAR) -- Argon2id
- name (VARCHAR)
- is_active (BOOLEAN)
- email_verified (BOOLEAN)
- created_at, updated_at (TIMESTAMPTZ)
```

### API Keys 테이블
```sql
- id (UUID, PK)
- user_id (UUID, FK -> users)
- broker (VARCHAR) -- 'KIS', 'eBest', etc
- key_encrypted (TEXT) -- AES-256-GCM
- secret_encrypted (TEXT) -- AES-256-GCM
- is_active (BOOLEAN)
- last_used_at (TIMESTAMPTZ)
- created_at, updated_at (TIMESTAMPTZ)
```

### Refresh Tokens 테이블
```sql
- id (UUID, PK)
- user_id (UUID, FK -> users)
- token_hash (VARCHAR, UNIQUE) -- SHA-256
- expires_at (TIMESTAMPTZ)
- revoked_at (TIMESTAMPTZ)
- ip_address (INET)
- user_agent (TEXT)
- created_at (TIMESTAMPTZ)
```

### Audit Logs 테이블
```sql
- id (UUID, PK)
- user_id (UUID, FK -> users)
- event_type (VARCHAR) -- 'login', 'password_change', etc
- event_data (JSONB)
- ip_address (INET)
- user_agent (TEXT)
- created_at (TIMESTAMPTZ)
```

## 환경 변수

필수 환경 변수는 `.env.example`을 참조하세요.

### 주요 환경 변수

```bash
# 서버
SERVER_PORT=8080
ENVIRONMENT=development

# 데이터베이스
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=aipx_users

# JWT 시크릿 (최소 32자)
JWT_ACCESS_SECRET=your_secure_secret_here
JWT_REFRESH_SECRET=your_secure_secret_here

# 암호화 키 (Base64 인코딩된 256비트 키)
ENCRYPTION_MASTER_KEY=your_base64_key_here
```

### 보안 키 생성

```bash
# JWT 시크릿 생성
openssl rand -base64 32

# 암호화 마스터 키 생성
openssl rand -base64 32
```

## 설치 및 실행

### 1. 의존성 설치

```bash
go mod download
```

### 2. 환경 변수 설정

```bash
cp .env.example .env
# .env 파일을 편집하여 필요한 값 설정
```

### 3. 데이터베이스 마이그레이션

```bash
# PostgreSQL 접속
psql -U postgres -h localhost

# 데이터베이스 생성
CREATE DATABASE aipx_users;

# 마이그레이션 실행
psql -U postgres -d aipx_users -f migrations/001_users.sql
```

### 4. 서비스 실행

```bash
go run cmd/server/main.go
```

또는 빌드 후 실행:

```bash
go build -o user-service cmd/server/main.go
./user-service
```

## API 엔드포인트 (T5에서 구현 예정)

### 인증
- `POST /api/v1/auth/register` - 사용자 등록
- `POST /api/v1/auth/login` - 로그인
- `POST /api/v1/auth/logout` - 로그아웃
- `POST /api/v1/auth/refresh` - 토큰 갱신
- `POST /api/v1/auth/verify-email` - 이메일 인증
- `POST /api/v1/auth/reset-password` - 비밀번호 재설정

### 사용자
- `GET /api/v1/users/me` - 현재 사용자 조회
- `PUT /api/v1/users/me` - 사용자 정보 수정
- `PUT /api/v1/users/me/password` - 비밀번호 변경
- `DELETE /api/v1/users/me` - 계정 삭제

### API 키
- `GET /api/v1/api-keys` - API 키 목록 조회
- `POST /api/v1/api-keys` - API 키 추가
- `DELETE /api/v1/api-keys/:id` - API 키 삭제
- `PUT /api/v1/api-keys/:id/activate` - API 키 활성화
- `PUT /api/v1/api-keys/:id/deactivate` - API 키 비활성화

### Health Check
- `GET /health` - 서비스 상태 확인

## 보안 고려사항

### 1. 비밀번호 정책
- 최소 8자, 최대 128자
- 대문자, 소문자, 숫자 포함 필수
- Argon2id 해싱 (OWASP 권장)

### 2. API 키 보안
- AES-256-GCM 암호화
- 마스터 키는 환경 변수로 관리 (절대 코드에 하드코딩 금지)
- 키 로테이션 지원
- 평문 API 키는 절대 로그에 기록하지 않음

### 3. JWT 토큰
- Access Token: 짧은 수명 (15분)
- Refresh Token: DB에 해시 저장, 취소 가능
- IP 주소 및 User-Agent 추적
- 로그아웃 시 토큰 즉시 취소

### 4. 감사 로깅
- 모든 인증 이벤트 기록
- 실패한 로그인 시도 추적
- API 키 생성/삭제 기록
- 비밀번호 변경 이력

### 5. Rate Limiting (TODO)
- 로그인 시도 제한
- API 호출 제한
- 이메일 전송 제한

### 6. 데이터베이스 보안
- SSL/TLS 연결 (프로덕션)
- 최소 권한 원칙
- 정기적인 백업
- 민감한 데이터 암호화

## 테스트

```bash
# 전체 테스트 실행
go test ./...

# 커버리지 포함
go test -cover ./...

# 특정 패키지 테스트
go test ./internal/auth/...
go test ./internal/crypto/...

# 벤치마크
go test -bench=. ./internal/auth/
go test -bench=. ./internal/crypto/
```

## 프로젝트 구조

```
user-service/
├── cmd/
│   └── server/
│       └── main.go              # 서버 진입점
├── internal/
│   ├── api/                     # API 핸들러 (T5에서 구현)
│   ├── auth/
│   │   ├── password.go          # Argon2id 비밀번호 해싱
│   │   └── password_test.go
│   ├── crypto/
│   │   ├── aes.go               # AES-256-GCM 암호화
│   │   └── aes_test.go
│   ├── config/
│   │   └── config.go            # 설정 관리
│   ├── models/
│   │   └── models.go            # 데이터 모델
│   └── repository/
│       ├── user_repo.go         # 사용자 저장소
│       ├── apikey_repo.go       # API 키 저장소
│       ├── token_repo.go        # 토큰 저장소
│       └── audit_repo.go        # 감사 로그 저장소
├── migrations/
│   └── 001_users.sql            # 데이터베이스 스키마
├── .env.example                 # 환경 변수 예시
├── .gitignore
├── go.mod
└── README.md
```

## 다음 단계 (T5)

1. JWT 토큰 생성/검증 구현
2. API 핸들러 구현
3. 인증 미들웨어 구현
4. 이메일 인증 기능
5. 비밀번호 재설정 기능
6. Rate Limiting 구현
7. 통합 테스트 작성

## 의존성

- **Gin**: HTTP 웹 프레임워크
- **pgx/v5**: PostgreSQL 드라이버 (연결 풀링)
- **golang.org/x/crypto**: Argon2id 암호화

## 라이선스

AIPX 프로젝트의 일부

## 기여

내부 프로젝트 - 팀 멤버만 기여 가능

## 문의

프로젝트 관련 문의는 팀 채널을 통해 주세요.
