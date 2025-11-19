# Scripts Documentation

이 디렉토리는 AIPX 프로젝트의 자동화 스크립트를 포함합니다.

## 📄 Available Scripts

### 1. proto-compile.sh

Protobuf 파일을 Go와 Python 코드로 컴파일하는 스크립트입니다.

#### 기능

- 모든 `.proto` 파일을 자동으로 찾아 컴파일
- Go 코드 생성 (*.pb.go, *_grpc.pb.go)
- Python 코드 생성 (*_pb2.py, *_pb2_grpc.py)
- 자동 에러 처리 및 상태 메시지 출력
- 이전 생성 파일 자동 정리

#### 사전 요구사항

**필수 (Go 컴파일용):**
```bash
# Protocol Buffers 컴파일러
brew install protobuf  # macOS
# 또는
sudo apt-get install protobuf-compiler  # Ubuntu

# Go 플러그인
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# PATH에 추가 (필요한 경우)
export PATH="$PATH:$(go env GOPATH)/bin"
```

**선택사항 (Python 컴파일용):**
```bash
pip install grpcio-tools
```

#### 사용법

**직접 실행:**
```bash
./scripts/proto-compile.sh
```

**Make 명령어 사용 (권장):**
```bash
# Protobuf 컴파일
make proto

# 생성된 파일 정리
make clean
```

#### 출력 위치

**Go 코드:**
- 위치: `shared/go/pkg/pb/`
- 파일: `*.pb.go`, `*_grpc.pb.go`
- Import: `import "github.com/jjongkwann/aipx/shared/go/pkg/pb"`

**Python 코드:**
- 위치: `shared/python/common/pb/`
- 파일: `*_pb2.py`, `*_pb2_grpc.py`
- Import: `from common.pb import order_pb2`

#### 컴파일되는 Proto 파일

1. **market_data.proto** - 시장 데이터 (틱, 호가)
2. **order.proto** - 주문 관리
3. **user.proto** - 사용자 인증
4. **strategy.proto** - 전략 관리

#### 예제 사용

**Go에서 사용:**
```go
package main

import (
    "github.com/jjongkwann/aipx/shared/go/pkg/pb"
)

func main() {
    order := &pb.OrderRequest{
        Symbol: "005930",
        Side: pb.Side_BUY,
        Type: pb.OrderType_LIMIT,
        Price: 70000,
        Quantity: 10,
    }
}
```

**Python에서 사용:**
```python
from common.pb import order_pb2

order = order_pb2.OrderRequest(
    symbol="005930",
    side=order_pb2.BUY,
    type=order_pb2.LIMIT,
    price=70000,
    quantity=10
)
```

#### 트러블슈팅

**문제: protoc를 찾을 수 없음**
```bash
# macOS
brew install protobuf

# Ubuntu
sudo apt-get update
sudo apt-get install protobuf-compiler
```

**문제: protoc-gen-go를 찾을 수 없음**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# PATH에 GOPATH/bin이 있는지 확인
echo $PATH | grep "$(go env GOPATH)/bin"

# 없다면 추가
export PATH="$PATH:$(go env GOPATH)/bin"
```

**문제: Python 컴파일 실패**
```bash
# grpcio-tools 설치
pip install grpcio-tools

# 또는 requirements.txt를 통해
pip install -r shared/python/requirements.txt
```

**문제: Import 경로 오류**
- Go: go.mod 파일의 module 경로 확인
- Python: PYTHONPATH 환경 변수 설정 확인

#### 스크립트 옵션

스크립트는 다음과 같은 상황을 자동으로 처리합니다:
- ✅ 필수 도구 설치 여부 확인
- ✅ 출력 디렉토리 자동 생성
- ✅ 이전 생성 파일 자동 정리
- ✅ 컴파일 성공/실패 상태 표시
- ✅ Python 도구 미설치 시 자동 스킵
- ✅ 적절한 exit code 반환

### 2. init-db.sql

PostgreSQL 데이터베이스 초기화 SQL 스크립트입니다.

#### 사용법

```bash
# Docker 컨테이너 내에서 자동 실행됨
docker-compose up -d

# 또는 수동 실행
psql -U aipx -d aipx_db -f scripts/init-db.sql
```

### 3. setup-aws-backend.sh

Terraform AWS 백엔드 설정을 위한 스크립트입니다.

#### 사용법

```bash
./scripts/setup-aws-backend.sh
```

## 🔄 일반적인 워크플로우

### Proto 파일 수정 후

1. Proto 파일 수정 (`shared/proto/*.proto`)
2. 컴파일 실행: `make proto`
3. 생성된 코드 확인
4. 서비스에서 import하여 사용

### 개발 환경 시작

```bash
# 1. 인프라 시작
make docker-up

# 2. Proto 컴파일
make proto

# 3. 의존성 설치
make install

# 4. 개발 시작
```

## 📚 추가 자료

- [Protocol Buffers 공식 문서](https://protobuf.dev/)
- [gRPC Go 튜토리얼](https://grpc.io/docs/languages/go/)
- [gRPC Python 튜토리얼](https://grpc.io/docs/languages/python/)

## ❓ 문제 해결

문제가 발생하면:
1. 스크립트의 상태 메시지 확인
2. 필수 도구 설치 여부 확인
3. 경로 및 권한 확인
4. GitHub Issues에 문의

---

**마지막 업데이트:** 2025-11-19
