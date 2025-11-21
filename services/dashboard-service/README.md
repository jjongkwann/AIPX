# Dashboard Service

**Dashboard Service**는 AIPX 트레이딩 시스템의 웹 인터페이스로, Next.js 14를 기반으로 구축된 모던 React 애플리케이션입니다.

## 주요 기능

### 1. AI 채팅 인터페이스
- WebSocket 기반 실시간 채팅
- GPT-4 기반 AI 에이전트와 대화
- 투자 전략 상담 및 시장 분석

### 2. 포트폴리오 모니터링
- 실시간 자산 현황
- 보유 포지션 상세 정보
- 손익 분석 및 차트

### 3. 백테스트 결과 시각화
- 전략 성과 지표
- 자산 곡선 차트
- 거래 내역 및 분석

### 4. 전략 관리
- 활성/비활성 전략 관리
- 전략 성과 모니터링
- 새로운 전략 등록

### 5. 실시간 시장 차트
- TradingView 스타일 캔들스틱 차트
- 다양한 시간대 지원
- 인터랙티브 차트 조작

## 기술 스택

- **Framework**: Next.js 14 (App Router)
- **UI Library**: React 18
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Components**: shadcn/ui
- **Charts**:
  - lightweight-charts (캔들스틱)
  - recharts (메트릭 시각화)
- **State Management**: React Hooks
- **API Client**: Axios
- **WebSocket**: Native WebSocket API

## 프로젝트 구조

```
dashboard-service/
├── src/
│   ├── app/                    # Next.js App Router
│   │   ├── (auth)/            # 인증 페이지
│   │   │   ├── login/
│   │   │   └── signup/
│   │   ├── dashboard/         # 대시보드 페이지
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx       # 메인 대시보드
│   │   │   ├── chat/          # AI 채팅
│   │   │   ├── portfolio/     # 포트폴리오
│   │   │   ├── backtest/      # 백테스트
│   │   │   └── strategies/    # 전략 관리
│   │   ├── layout.tsx         # 루트 레이아웃
│   │   ├── page.tsx           # 랜딩 페이지
│   │   └── globals.css        # 글로벌 스타일
│   ├── components/            # React 컴포넌트
│   │   ├── ui/               # shadcn/ui 컴포넌트
│   │   ├── TradingViewChart.tsx
│   │   ├── ChatInterface.tsx
│   │   ├── PortfolioTable.tsx
│   │   ├── BacktestResults.tsx
│   │   └── Navbar.tsx
│   └── lib/                   # 유틸리티 및 라이브러리
│       ├── api.ts            # API 클라이언트
│       ├── websocket.ts      # WebSocket 훅
│       └── utils.ts          # 헬퍼 함수
├── public/                    # 정적 파일
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── next.config.js
├── tailwind.config.ts
├── tsconfig.json
└── package.json
```

## 시작하기

자세한 개발 환경 설정은 [DEVELOPMENT.md](./DEVELOPMENT.md)를 참조하세요.

### 빠른 시작

```bash
# 의존성 설치
npm install

# 환경 변수 설정
cp .env.local.example .env.local

# 개발 서버 실행
npm run dev
```

브라우저에서 http://localhost:3000을 열어 확인하세요.

## 배포

프로덕션 배포 가이드는 [DEPLOYMENT.md](./DEPLOYMENT.md)를 참조하세요.

### Docker로 실행

```bash
# 이미지 빌드
docker build -t aipx-dashboard .

# 컨테이너 실행
docker run -p 3000:3000 aipx-dashboard
```

## 주요 컴포넌트

### TradingViewChart
실시간 시장 데이터를 TradingView 스타일 차트로 시각화합니다.

```tsx
<TradingViewChart
  symbol="BTC/KRW"
  data={candlestickData}
  height={400}
/>
```

### ChatInterface
AI 에이전트와 실시간 채팅 인터페이스를 제공합니다.

```tsx
<ChatInterface userId="user-123" />
```

### PortfolioTable
포트폴리오 포지션을 테이블 형태로 표시합니다.

```tsx
<PortfolioTable positions={portfolioData.positions} />
```

### BacktestResults
백테스트 결과를 차트와 메트릭으로 시각화합니다.

```tsx
<BacktestResults report={backtestReport} />
```

## 성능 최적화

### 1. 컴포넌트 최적화
- React.memo를 사용한 불필요한 리렌더링 방지
- useMemo/useCallback을 통한 계산 최적화
- 코드 분할 및 동적 import

### 2. 이미지 최적화
- Next.js Image 컴포넌트 사용
- WebP/AVIF 포맷 지원
- 레이지 로딩

### 3. 번들 최적화
- Tree shaking
- 코드 스플리팅
- 프리페칭

## 접근성 (Accessibility)

- WCAG 2.1 Level AA 준수
- 시맨틱 HTML 사용
- ARIA 레이블 및 역할
- 키보드 네비게이션 지원
- 스크린 리더 호환

## 보안

- HTTPS 강제
- CSRF 보호
- XSS 방지
- 보안 헤더 설정
- Rate limiting (nginx)

## 브라우저 지원

- Chrome (최신 2개 버전)
- Firefox (최신 2개 버전)
- Safari (최신 2개 버전)
- Edge (최신 2개 버전)

## 라이선스

MIT License

## 기여

기여는 언제나 환영합니다! Pull Request를 보내주세요.

## 문의

문제가 발생하면 GitHub Issues에 등록해주세요.
