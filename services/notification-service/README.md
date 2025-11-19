# 알림 서비스 (Notification Service)

**Notification Service**는 시스템에서 발생하는 중요한 이벤트를 사용자에게 실시간으로 전달하는 서비스입니다. Kafka 이벤트를 구독하여 메신저나 이메일로 발송합니다.

## 🛠 주요 기능 (Features)

### 1. 이벤트 구독 (Event Subscription)
-   **Kafka Consumer**: `trade.orders` (체결), `risk.alerts` (위험 감지), `system.errors` (시스템 장애) 등 다양한 토픽을 구독합니다.

### 2. 멀티 채널 지원 (Multi-Channel Support)
-   **메신저**: Slack Webhook, Telegram Bot API, 카카오톡 비즈메시지 등을 지원합니다.
-   **이메일**: SMTP 또는 AWS SES를 통해 중요 리포트를 발송합니다.

### 3. 알림 라우팅 (Notification Routing)
-   **사용자별 설정**: 사용자가 설정한 알림 수신 여부와 선호 채널에 따라 메시지를 필터링하고 라우팅합니다.
-   **템플릿 엔진**: 이벤트 데이터(JSON)를 읽기 쉬운 메시지 포맷으로 변환합니다.

## 🚀 시작하기 (Getting Started)

### 초기화
```bash
go mod tidy
```

### 실행
```bash
go run main.go
```
