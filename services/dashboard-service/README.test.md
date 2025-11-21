# Dashboard Service Testing Guide

## Overview

Comprehensive test suite for the AIPX Dashboard Service with 70%+ code coverage.

## Test Structure

```
services/dashboard-service/
├── src/__tests__/          # Unit tests (co-located with source)
│   ├── components/         # Component tests
│   │   ├── TradingViewChart.test.tsx
│   │   ├── ChatInterface.test.tsx
│   │   ├── Navbar.test.tsx
│   │   ├── BacktestResults.test.tsx
│   │   └── PortfolioTable.test.tsx
│   └── pages/             # Page tests
│       └── login.test.tsx
├── tests/
│   ├── setup.ts           # Test setup and global mocks
│   ├── utils/             # Test utilities
│   │   └── test-utils.tsx
│   ├── mocks/             # Mock data and MSW handlers
│   │   ├── data.ts
│   │   ├── handlers.ts
│   │   └── server.ts
│   ├── integration/       # Integration tests
│   │   └── test_api_client.test.ts
│   ├── e2e/              # End-to-end tests (Playwright)
│   │   ├── auth.spec.ts
│   │   ├── chat.spec.ts
│   │   ├── portfolio.spec.ts
│   │   ├── backtest.spec.ts
│   │   └── navigation.spec.ts
│   ├── a11y/             # Accessibility tests
│   │   └── wcag.spec.ts
│   ├── performance/       # Performance tests
│   │   └── load.spec.ts
│   └── visual/           # Visual regression tests
│       └── screenshots.spec.ts
```

## Running Tests

### All Tests

```bash
# Run all tests (unit + integration)
npm test

# Run tests in CI mode with coverage
npm run test:ci

# Run tests with coverage report
npm run test:coverage
```

### Unit & Integration Tests

```bash
# Watch mode for development
npm test

# Run once
npm run test:ci
```

### E2E Tests

```bash
# Run all E2E tests
npm run test:e2e

# Run E2E tests with UI mode
npm run test:e2e:ui

# Run specific test file
npx playwright test tests/e2e/auth.spec.ts

# Run tests in headed mode (see browser)
npx playwright test --headed

# Debug specific test
npx playwright test --debug tests/e2e/auth.spec.ts
```

### Accessibility Tests

```bash
# Run accessibility tests
npm run test:a11y

# Or run with Playwright
npx playwright test tests/a11y/
```

### Performance Tests

```bash
# Run performance tests
npm run test:perf
```

### Visual Regression Tests

```bash
# Run visual tests
npx playwright test tests/visual/

# Update baseline screenshots
npx playwright test tests/visual/ --update-snapshots
```

## Test Coverage

Current coverage thresholds (enforced in CI):

- **Branches**: 70%
- **Functions**: 70%
- **Lines**: 70%
- **Statements**: 70%

View coverage report:

```bash
npm run test:coverage
open coverage/lcov-report/index.html
```

## Writing Tests

### Component Test Example

```typescript
import { render, screen } from '@/tests/utils/test-utils'
import MyComponent from '@/components/MyComponent'

describe('MyComponent', () => {
  it('renders correctly', () => {
    render(<MyComponent title="Test" />)
    expect(screen.getByText('Test')).toBeInTheDocument()
  })
})
```

### E2E Test Example

```typescript
import { test, expect } from '@playwright/test'

test('user can login', async ({ page }) => {
  await page.goto('/login')
  await page.fill('input[type="email"]', 'test@example.com')
  await page.fill('input[type="password"]', 'password123')
  await page.click('button:has-text("로그인")')
  await expect(page).toHaveURL('/dashboard')
})
```

## Mocking

### API Mocking (MSW)

Mock API responses in tests:

```typescript
import { server } from '@/tests/mocks/server'
import { http, HttpResponse } from 'msw'

test('handles API error', async () => {
  server.use(
    http.get('/api/v1/portfolio', () => {
      return HttpResponse.json({ error: 'Failed' }, { status: 500 })
    })
  )

  // Test error handling
})
```

### WebSocket Mocking

```typescript
import { MockWebSocket } from '@/tests/utils/test-utils'

// Use MockWebSocket in tests
global.WebSocket = MockWebSocket as any
```

## Best Practices

### Unit Tests

1. **Test behavior, not implementation**
   ```typescript
   // Good: Test what the user sees
   expect(screen.getByText('Welcome')).toBeInTheDocument()

   // Avoid: Testing internal state
   expect(component.state.isLoading).toBe(false)
   ```

2. **Use data-testid sparingly**
   - Prefer accessible queries (getByRole, getByLabelText)
   - Only use data-testid when no accessible alternative exists

3. **Keep tests isolated**
   - Each test should be independent
   - Clean up after each test

### E2E Tests

1. **Use real user workflows**
   - Test complete user journeys
   - Avoid testing implementation details

2. **Wait for elements properly**
   ```typescript
   // Good: Wait for specific condition
   await page.waitForSelector('text=Loaded')

   // Avoid: Arbitrary timeouts
   await page.waitForTimeout(5000)
   ```

3. **Handle flaky tests**
   - Use proper wait conditions
   - Retry failed tests in CI
   - Investigate and fix root cause

### Accessibility Tests

1. **Test with real assistive technologies**
2. **Ensure keyboard navigation works**
3. **Verify ARIA labels are meaningful**
4. **Check color contrast ratios**

## Debugging Tests

### Unit Tests

```bash
# Run specific test file
npm test -- TradingViewChart.test.tsx

# Run tests matching pattern
npm test -- --testNamePattern="renders correctly"

# Debug with Node inspector
node --inspect-brk node_modules/.bin/jest --runInBand
```

### E2E Tests

```bash
# Debug mode (opens inspector)
npx playwright test --debug

# Headed mode (see browser)
npx playwright test --headed

# Slow mo (slow down actions)
npx playwright test --headed --slow-mo=1000

# Generate trace
npx playwright test --trace on
npx playwright show-trace trace.zip
```

## CI/CD Integration

Tests run automatically on:
- Push to main/develop branches
- Pull requests
- Pre-commit hooks (optional)

## Troubleshooting

### Common Issues

1. **Tests timeout**
   - Increase timeout in test or config
   - Check for network issues
   - Verify mock data is correct

2. **Flaky tests**
   - Add proper wait conditions
   - Avoid timing-dependent logic
   - Check for race conditions

3. **Coverage not meeting threshold**
   - Add tests for uncovered code
   - Remove dead code
   - Adjust threshold if justified

### Getting Help

- Check test output for detailed errors
- Review Playwright trace for E2E issues
- Check MSW handlers for API mocking issues
- Review component props and types

## Performance Benchmarks

Target metrics:
- **LCP** (Largest Contentful Paint): < 2.5s
- **FCP** (First Contentful Paint): < 1.8s
- **CLS** (Cumulative Layout Shift): < 0.1
- **TTI** (Time to Interactive): < 3.8s

Run performance tests:
```bash
npm run test:perf
```

## Visual Regression

Update screenshots when UI changes:
```bash
npx playwright test tests/visual/ --update-snapshots
```

Review visual diffs:
```bash
npx playwright show-report
```

## Resources

- [Jest Documentation](https://jestjs.io/)
- [React Testing Library](https://testing-library.com/react)
- [Playwright Documentation](https://playwright.dev/)
- [MSW Documentation](https://mswjs.io/)
- [axe-core Accessibility](https://github.com/dequelabs/axe-core)
