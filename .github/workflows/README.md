# GitHub Actions Workflows

이 디렉토리는 AIPX 프로젝트의 CI/CD 워크플로우를 포함합니다.

## 워크플로우 개요

### 1. Proto Lint (`proto-lint.yml`)

**목적**: Protobuf 파일의 린트, 형식 검사 및 컴파일 검증

**트리거**:
- `main` 브랜치로의 push
- `main` 브랜치로의 PR (변경사항: `shared/proto/**`)

**주요 작업**:
- Buf를 사용한 proto 파일 린트
- PR에서 breaking changes 검사
- proto 형식 검증
- Go 및 Python용 proto 코드 컴파일 검증

**소요 시간**: ~5-10분

**실패 시 조치**:
```bash
# 로컬에서 proto 파일 검증
cd shared/proto
buf lint
buf format -w
buf generate
```

---

### 2. Go Tests (`go-test.yml`)

**목적**: Go 코드의 린트, 테스트 및 빌드 검증

**트리거**:
- `main` 브랜치로의 push
- `main` 브랜치로의 PR (변경사항: `**/*.go`, `go.mod`, `go.sum`)

**주요 작업**:
- **Lint Job**: `go vet`, `golangci-lint` 실행
- **Test Job**: Go 1.21, 1.22에서 테스트 실행
  - Race detector 활성화
  - Coverage 리포트 생성
  - Codecov 업로드
- **Build Job**: 모든 Go 서비스 빌드 검증

**Go 버전**: 1.21, 1.22 (매트릭스)

**소요 시간**: ~10-15분

**캐싱**:
- Go 모듈 캐시
- 빌드 캐시

**실패 시 조치**:
```bash
# 린트 문제 해결
cd shared/go
go vet ./...
golangci-lint run

# 테스트 실행
go test -v -race ./...

# 빌드 확인
cd services/your-service
go build -v ./...
```

---

### 3. Python Tests (`python-test.yml`)

**목적**: Python 코드의 린트, 타입 체크, 테스트 및 보안 스캔

**트리거**:
- `main` 브랜치로의 push
- `main` 브랜치로의 PR (변경사항: `**/*.py`, `requirements*.txt`, `pyproject.toml`)

**주요 작업**:
- **Lint Job**:
  - Ruff (린터 + 포매터)
  - Black (코드 포매터)
  - isort (import 정렬)
  - mypy (타입 체크)
- **Test Job**: Python 3.11, 3.12에서 테스트 실행
  - pytest with coverage
  - HTML 커버리지 리포트
  - Codecov 업로드
- **Test Services Job**: 각 Python 서비스별 테스트
- **Security Job**:
  - Safety (의존성 취약점 스캔)
  - Bandit (보안 린터)

**Python 버전**: 3.11, 3.12 (매트릭스)

**소요 시간**: ~15-20분

**캐싱**:
- pip 패키지 캐시

**실패 시 조치**:
```bash
# 린트 및 포매팅
cd shared/python
ruff check aipx/ --fix
ruff format aipx/
black aipx/
isort aipx/

# 타입 체크
mypy aipx/

# 테스트 실행
pytest tests/ -v --cov=aipx

# 보안 스캔
safety check -r requirements.txt
bandit -r aipx/
```

---

### 4. Docker Build (`docker-build.yml`)

**목적**: Docker 이미지 빌드, 보안 스캔 및 레지스트리 푸시

**트리거**:
- `main` 브랜치로의 push
- `main` 브랜치로의 PR (변경사항: `**/Dockerfile`, `docker-compose.yml`, `services/**`, `shared/**`)

**주요 작업**:
- **Docker Compose Build**: 모든 서비스 빌드 및 시작 검증
- **Security Scan**: Trivy를 사용한 취약점 스캔
  - PostgreSQL, Redis, Kafka 이미지 스캔
  - GitHub Security에 결과 업로드
- **Build and Push** (main 브랜치만):
  - GitHub Container Registry로 푸시
  - Multi-platform 빌드 (amd64, arm64)
  - 자동 태깅 (latest, sha, semver)
- **Base Images**: Go 및 Python 베이스 이미지 빌드
- **Size Report**: 이미지 크기 리포트 생성

**레지스트리**: `ghcr.io` (GitHub Container Registry)

**소요 시간**: ~20-30분

**캐싱**:
- Docker Buildx 캐시
- GitHub Actions 캐시

**실패 시 조치**:
```bash
# 로컬에서 빌드 테스트
docker-compose build --parallel

# 서비스 시작 확인
docker-compose up -d
docker-compose ps
docker-compose logs

# Trivy 스캔 실행
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image your-image:tag

# 정리
docker-compose down -v
```

---

## 공통 기능

### Concurrency Control

모든 워크플로우는 동일한 PR/브랜치에서 진행 중인 실행을 자동으로 취소합니다:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
```

### Timeout Limits

모든 작업은 15분 타임아웃이 설정되어 있습니다:

```yaml
timeout-minutes: 15
```

### Fail-Fast Strategy

테스트 작업은 `fail-fast: false`로 설정되어 모든 매트릭스 조합을 실행합니다.

---

## 상태 배지

README.md에 다음 배지를 추가하세요:

```markdown
![Proto Lint](https://github.com/YOUR_USERNAME/AIPX/workflows/Proto%20Lint/badge.svg)
![Go Tests](https://github.com/YOUR_USERNAME/AIPX/workflows/Go%20Tests/badge.svg)
![Python Tests](https://github.com/YOUR_USERNAME/AIPX/workflows/Python%20Tests/badge.svg)
![Docker Build](https://github.com/YOUR_USERNAME/AIPX/workflows/Docker%20Build/badge.svg)
[![codecov](https://codecov.io/gh/YOUR_USERNAME/AIPX/branch/main/graph/badge.svg)](https://codecov.io/gh/YOUR_USERNAME/AIPX)
```

---

## 필수 시크릿 설정

GitHub 저장소 Settings > Secrets and variables > Actions에서 다음 시크릿을 설정하세요:

### 선택 사항 (Coverage 리포트용):
- `CODECOV_TOKEN`: Codecov 토큰 (https://codecov.io에서 생성)

### 자동 제공 (설정 불필요):
- `GITHUB_TOKEN`: GitHub Actions 자동 제공

---

## Dependabot 설정

`.github/dependabot.yml` 파일은 다음 의존성을 자동으로 업데이트합니다:

- **Go modules** (월요일 09:00)
- **Python packages** (월요일 09:00)
- **Docker images** (화요일 09:00)
- **GitHub Actions** (수요일 09:00)
- **Terraform** (목요일 09:00)

Dependabot은 매주 자동으로 PR을 생성하며, 각 PR은 자동으로 CI 워크플로우를 실행합니다.

---

## 워크플로우 디버깅

### 로컬에서 워크플로우 테스트

[act](https://github.com/nektos/act)를 사용하여 로컬에서 GitHub Actions를 테스트할 수 있습니다:

```bash
# act 설치 (macOS)
brew install act

# 워크플로우 실행
act -j lint  # 특정 job 실행
act pull_request  # PR 이벤트 시뮬레이션

# 시크릿 사용
act -s CODECOV_TOKEN=your-token
```

### GitHub Actions 로그 확인

1. GitHub 저장소 > Actions 탭
2. 실패한 워크플로우 클릭
3. 실패한 작업 클릭
4. 상세 로그 확인

### 재실행

실패한 워크플로우는 다음과 같이 재실행할 수 있습니다:

1. Actions 탭에서 워크플로우 선택
2. "Re-run jobs" > "Re-run all jobs" 또는 "Re-run failed jobs"

---

## 성능 최적화

### 캐싱 전략

각 워크플로우는 다음을 캐시합니다:

- **Go**: 모듈 및 빌드 캐시
- **Python**: pip 패키지 캐시
- **Docker**: Buildx 레이어 캐시
- **Proto**: Buf 모듈 캐시

### 병렬 실행

- 매트릭스 전략으로 여러 버전 동시 테스트
- Docker 서비스 병렬 빌드
- 독립적인 작업 동시 실행

### 조건부 실행

- PR에서만 breaking changes 체크
- main 브랜치에서만 이미지 푸시
- 변경된 파일 경로 기반 트리거

---

## 트러블슈팅

### 일반적인 문제

#### 1. Go 테스트 실패
```bash
# 로컬 재현
cd shared/go
go mod tidy
go test -v ./...
```

#### 2. Python 린트 실패
```bash
# 자동 수정
cd shared/python
ruff check aipx/ --fix
ruff format aipx/
black aipx/
isort aipx/
```

#### 3. Docker 빌드 실패
```bash
# 캐시 없이 빌드
docker-compose build --no-cache

# 개별 서비스 빌드
docker-compose build service-name
```

#### 4. 캐시 문제
- GitHub Actions 페이지에서 캐시 수동 삭제
- 워크플로우 파일에서 캐시 키 변경

---

## 모범 사례

### PR 작성 시

1. 작은 단위로 커밋
2. 커밋 메시지 규칙 준수 (conventional commits)
3. PR 전에 로컬에서 테스트 실행
4. Draft PR로 CI 먼저 확인

### 워크플로우 수정 시

1. 작은 변경으로 시작
2. `act`로 로컬 테스트
3. Draft PR로 실제 환경 테스트
4. 타임아웃 설정 확인

### 의존성 업데이트 시

1. Dependabot PR 검토
2. CHANGELOG 확인
3. Breaking changes 주의
4. 단계적 업데이트 (major 버전은 신중히)

---

## 추가 리소스

- [GitHub Actions 문서](https://docs.github.com/en/actions)
- [Dependabot 문서](https://docs.github.com/en/code-security/dependabot)
- [Docker Buildx 문서](https://docs.docker.com/buildx/working-with-buildx/)
- [Buf 문서](https://buf.build/docs)
- [Codecov 문서](https://docs.codecov.com/)

---

## 연락처

워크플로우 관련 문제나 제안사항은 이슈를 생성해 주세요.
