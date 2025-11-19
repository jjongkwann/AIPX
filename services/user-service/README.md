# 사용자 서비스 (User Service)

**User Service**는 AIPX 시스템의 보안과 사용자 관리를 담당하는 핵심 서비스입니다. 회원가입, 로그인, API 키 관리 등 모든 인증/인가 프로세스를 처리합니다.

## 🛠 주요 기능 (Features)

### 1. 인증 (Authentication)
-   **회원가입 및 로그인**: 이메일/비밀번호 기반 또는 OAuth2(구글, 카카오) 로그인을 지원합니다.
-   **JWT (JSON Web Token)**: 로그인 성공 시 JWT Access/Refresh Token을 발급하여, 다른 마이크로서비스 요청 시 인증 수단으로 사용합니다.

### 2. 보안 (Security)
-   **API 키 암호화**: 사용자가 입력한 증권사 API Key와 Secret을 AES-256 등으로 암호화하여 DB에 저장합니다.
-   **비밀번호 해싱**: 사용자 비밀번호는 Argon2 또는 Bcrypt로 단방향 해싱하여 저장합니다.

### 3. 사용자 관리 (User Management)
-   **프로필 관리**: 닉네임, 알림 설정 등 사용자 개인화 정보를 관리합니다.
-   **권한 관리 (RBAC)**: 일반 사용자, 관리자 등 역할에 따른 접근 권한을 제어합니다.

## 🚀 시작하기 (Getting Started)

### 초기화
```bash
go mod tidy
```

### 실행
```bash
go run main.go
```
