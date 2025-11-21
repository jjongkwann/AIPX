# Quick Start Guide

AIPX Dashboard Service를 5분 안에 시작하는 가이드입니다.

## 1. 사전 요구사항

```bash
node --version  # v20.0.0 이상
npm --version   # v10.0.0 이상
```

## 2. 설치

```bash
# 디렉토리 이동
cd services/dashboard-service

# 의존성 설치
npm install
```

## 3. 환경 설정

```bash
# 환경 변수 파일 생성
cp .env.local.example .env.local

# .env.local 편집
nano .env.local
```

**필수 환경 변수:**
```env
NEXT_PUBLIC_API_BASE=http://localhost:8000
NEXT_PUBLIC_WS_BASE=ws://localhost:8001
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=development-secret-key
```

## 4. 개발 서버 실행

```bash
npm run dev
```

브라우저에서 http://localhost:3000 열기

## 5. 빌드 및 프로덕션 실행

```bash
# 프로덕션 빌드
npm run build

# 프로덕션 서버 시작
npm start
```

## 6. Docker로 실행

```bash
# 이미지 빌드
docker build -t aipx-dashboard .

# 컨테이너 실행
docker run -p 3000:3000 \
  -e NEXT_PUBLIC_API_BASE=http://localhost:8000 \
  aipx-dashboard
```

## 7. Docker Compose로 실행

```bash
# 전체 스택 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f

# 종료
docker-compose down
```

## 주요 엔드포인트

- **랜딩:** http://localhost:3000
- **로그인:** http://localhost:3000/login
- **대시보드:** http://localhost:3000/dashboard
- **AI 채팅:** http://localhost:3000/dashboard/chat
- **포트폴리오:** http://localhost:3000/dashboard/portfolio
- **백테스트:** http://localhost:3000/dashboard/backtest
- **전략:** http://localhost:3000/dashboard/strategies

## 개발 명령어

```bash
npm run dev         # 개발 서버 (포트 3000)
npm run build       # 프로덕션 빌드
npm start           # 프로덕션 서버
npm run lint        # ESLint 검사
npm run type-check  # TypeScript 타입 체크
```

## 폴더 구조 (핵심)

```
dashboard-service/
├── src/
│   ├── app/              # 페이지 (Next.js App Router)
│   ├── components/       # React 컴포넌트
│   └── lib/             # 유틸리티 (API, WebSocket, 헬퍼)
├── public/              # 정적 파일
└── Dockerfile           # Docker 설정
```

## 컴포넌트 사용 예시

### 1. TradingView 차트

```tsx
import TradingViewChart from '@/components/TradingViewChart'

<TradingViewChart
  symbol="BTC/KRW"
  data={candlestickData}
  height={400}
/>
```

### 2. AI 채팅

```tsx
import ChatInterface from '@/components/ChatInterface'

<ChatInterface userId="user-123" />
```

### 3. 포트폴리오 테이블

```tsx
import PortfolioTable from '@/components/PortfolioTable'

<PortfolioTable positions={portfolioData.positions} />
```

## API 호출 예시

```typescript
import { login, getPortfolio, getBacktests } from '@/lib/api'

// 로그인
const { access_token, user } = await login({
  email: 'user@example.com',
  password: 'password'
})

// 포트폴리오 조회
const portfolio = await getPortfolio('user-123')

// 백테스트 목록
const backtests = await getBacktests()
```

## WebSocket 사용 예시

```typescript
import { useWebSocket } from '@/lib/websocket'

const { sendMessage, lastMessage, readyState } = useWebSocket(
  'ws://localhost:8001/api/v1/ws/chat/user-123'
)

// 메시지 전송
sendMessage(JSON.stringify({ type: 'message', content: 'Hello' }))
```

## 트러블슈팅

### 포트 충돌
```bash
# 3000 포트 사용 프로세스 종료
kill -9 $(lsof -ti:3000)
```

### 의존성 문제
```bash
# node_modules 삭제 후 재설치
rm -rf node_modules package-lock.json
npm install
```

### 캐시 문제
```bash
# Next.js 캐시 삭제
rm -rf .next
npm run dev
```

### WebSocket 연결 실패
1. 백엔드 서비스 실행 확인
2. 환경 변수 NEXT_PUBLIC_WS_BASE 확인
3. 방화벽 설정 확인

## 성능 확인

```bash
# Lighthouse 점수 확인 (Chrome DevTools)
# - Performance: 90+ 목표
# - Accessibility: 95+ 목표
# - Best Practices: 95+ 목표
# - SEO: 90+ 목표
```

## 다음 단계

1. **개발 가이드**: [DEVELOPMENT.md](./DEVELOPMENT.md) 참조
2. **배포 가이드**: [DEPLOYMENT.md](./DEPLOYMENT.md) 참조
3. **전체 문서**: [README.md](./README.md) 참조

## 도움이 필요하신가요?

- 📖 [전체 문서 읽기](./README.md)
- 🐛 [이슈 제기](https://github.com/your-repo/issues)
- 💬 팀 슬랙 채널 문의

---

**Happy Coding! 🚀**
