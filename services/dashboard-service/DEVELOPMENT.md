# Development Guide

Dashboard Service 개발 환경 설정 및 개발 가이드입니다.

## 사전 요구사항

- Node.js 20.x 이상
- npm 10.x 이상
- Git

## 개발 환경 설정

### 1. 저장소 클론 및 의존성 설치

```bash
cd services/dashboard-service
npm install
```

### 2. 환경 변수 설정

`.env.local` 파일을 생성하고 다음 변수를 설정하세요:

```env
# API Configuration
NEXT_PUBLIC_API_BASE=http://localhost:8000
NEXT_PUBLIC_WS_BASE=ws://localhost:8001

# NextAuth Configuration
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=your-development-secret-key
```

### 3. 개발 서버 실행

```bash
npm run dev
```

브라우저에서 http://localhost:3000을 열어 확인하세요.

## 프로젝트 구조

### App Router (src/app/)

Next.js 14의 App Router를 사용합니다:

- `page.tsx`: 라우트 페이지
- `layout.tsx`: 레이아웃 컴포넌트
- `loading.tsx`: 로딩 UI
- `error.tsx`: 에러 UI

### 컴포넌트 (src/components/)

재사용 가능한 React 컴포넌트를 관리합니다:

- `ui/`: shadcn/ui 기반 기본 UI 컴포넌트
- 나머지: 비즈니스 로직 컴포넌트

### 라이브러리 (src/lib/)

유틸리티 함수와 공통 로직:

- `api.ts`: API 클라이언트 및 엔드포인트
- `websocket.ts`: WebSocket 훅
- `utils.ts`: 헬퍼 함수

## 개발 워크플로우

### 1. 새로운 페이지 추가

```bash
# 예: /dashboard/settings 페이지 추가
mkdir -p src/app/dashboard/settings
touch src/app/dashboard/settings/page.tsx
```

### 2. 새로운 컴포넌트 추가

```tsx
// src/components/MyComponent.tsx
'use client' // 클라이언트 컴포넌트인 경우

import { memo } from 'react'

interface MyComponentProps {
  // props 정의
}

const MyComponent = memo(function MyComponent(props: MyComponentProps) {
  return (
    // JSX
  )
})

export default MyComponent
```

### 3. API 엔드포인트 추가

```typescript
// src/lib/api.ts
export async function getNewData(): Promise<NewDataType> {
  const response = await apiClient.get<NewDataType>('/api/v1/new-endpoint')
  return response.data
}
```

### 4. 스타일링

Tailwind CSS를 사용합니다:

```tsx
<div className="bg-gray-900 p-4 rounded-lg hover:bg-gray-800">
  Content
</div>
```

## 코딩 컨벤션

### TypeScript

- 명시적 타입 정의 사용
- `interface` 우선, `type`은 필요시만
- any 사용 금지

```typescript
// Good
interface User {
  id: string
  name: string
}

// Bad
type User = {
  id: any
  name: any
}
```

### React 컴포넌트

- 함수형 컴포넌트 사용
- Hooks 사용
- memo로 최적화

```tsx
// Good
const MyComponent = memo(function MyComponent({ data }: Props) {
  const [state, setState] = useState()
  return <div>{data}</div>
})

// Bad
class MyComponent extends React.Component {
  render() {
    return <div>{this.props.data}</div>
  }
}
```

### 파일 명명 규칙

- 컴포넌트: `PascalCase.tsx`
- 유틸리티: `camelCase.ts`
- 페이지: `page.tsx`
- 레이아웃: `layout.tsx`

## 테스트

### 단위 테스트

```bash
npm run test
```

### E2E 테스트

```bash
npm run test:e2e
```

### 타입 체크

```bash
npm run type-check
```

### 린팅

```bash
npm run lint
```

## 디버깅

### React DevTools

Chrome 확장 프로그램 설치:
- React Developer Tools
- Redux DevTools (상태 관리 사용시)

### VSCode 디버깅

`.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Next.js: debug server-side",
      "type": "node-terminal",
      "request": "launch",
      "command": "npm run dev"
    }
  ]
}
```

## 성능 프로파일링

### Next.js 빌드 분석

```bash
npm run build
npm run analyze
```

### React Profiler

개발 도구에서 React Profiler 탭 사용

## 일반적인 문제 해결

### 포트 충돌

```bash
# 3000 포트를 사용하는 프로세스 확인
lsof -ti:3000

# 프로세스 종료
kill -9 $(lsof -ti:3000)
```

### 캐시 문제

```bash
# Next.js 캐시 삭제
rm -rf .next

# node_modules 재설치
rm -rf node_modules package-lock.json
npm install
```

### WebSocket 연결 실패

1. 백엔드 서비스가 실행 중인지 확인
2. CORS 설정 확인
3. 방화벽 설정 확인

## Hot Reload

파일 저장시 자동으로 브라우저가 새로고침됩니다:

- 컴포넌트 변경: Fast Refresh
- 페이지 변경: Full Reload
- 설정 파일 변경: 서버 재시작 필요

## 환경별 설정

### Development

```env
NODE_ENV=development
NEXT_PUBLIC_API_BASE=http://localhost:8000
```

### Staging

```env
NODE_ENV=production
NEXT_PUBLIC_API_BASE=https://staging-api.aipx.com
```

### Production

```env
NODE_ENV=production
NEXT_PUBLIC_API_BASE=https://api.aipx.com
```

## 추가 리소스

- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)

## 도움말

문제가 있으면 팀 슬랙 채널에 문의하거나 GitHub Issues에 등록하세요.
