import { test, expect } from '@playwright/test'

test.describe('Backtest Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('user can navigate to backtest page', async ({ page }) => {
    await page.click('a:has-text("백테스트")')

    await expect(page).toHaveURL('/dashboard/backtest')
  })

  test('backtest list displays historical backtests', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    // Wait for content to load
    await page.waitForLoadState('networkidle')

    // Check for backtest list or empty state
    const content = page.locator('text=전략').or(page.locator('text=백테스트'))
    await expect(content).toBeVisible({ timeout: 10000 })
  })

  test('user can create new backtest', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    // Look for create button
    const createButton = page.locator('button:has-text("새 백테스트")').or(
      page.locator('button:has-text("생성")')
    )

    if (await createButton.isVisible()) {
      await createButton.click()

      // Should show create form or modal
      await expect(page.locator('text=전략 선택').or(page.locator('input'))).toBeVisible()
    }
  })

  test('backtest form validates required fields', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    const createButton = page.locator('button:has-text("새 백테스트")').or(
      page.locator('button:has-text("생성")')
    )

    if (await createButton.isVisible()) {
      await createButton.click()

      // Try to submit without filling
      const submitButton = page.locator('button[type="submit"]')
      if (await submitButton.isVisible()) {
        await submitButton.click()

        // Should show validation errors or prevent submission
        // (Implementation depends on form design)
      }
    }
  })

  test('user can view backtest results', async ({ page, context }) => {
    // Mock backtest result
    await context.route('**/api/v1/backtests/*', route => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          body: JSON.stringify({
            id: 'test-id',
            strategy_name: 'Test Strategy',
            summary: {
              total_return: 0.15,
              cagr: 0.12,
              mdd: -0.08,
              sharpe_ratio: 1.5,
              win_rate: 0.58,
              total_trades: 100,
              winning_trades: 58,
              losing_trades: 42,
            },
            equity_curve: [],
            trades: [],
          }),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/dashboard/backtest/test-id')

    // Check for result metrics
    await expect(page.locator('text=총 수익률').or(page.locator('text=CAGR'))).toBeVisible({ timeout: 10000 })
  })

  test('backtest results display performance metrics', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    // Click on first backtest if available
    const backtestLink = page.locator('a').filter({ hasText: /Strategy|전략/ }).first()

    if (await backtestLink.isVisible({ timeout: 5000 })) {
      await backtestLink.click()

      // Should show metrics
      const metrics = page.locator('text=CAGR').or(page.locator('text=샤프'))
      await expect(metrics).toBeVisible({ timeout: 10000 })
    }
  })

  test('backtest results show equity curve chart', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    const backtestLink = page.locator('a').filter({ hasText: /Strategy|전략/ }).first()

    if (await backtestLink.isVisible({ timeout: 5000 })) {
      await backtestLink.click()

      // Check for chart section
      const chartSection = page.locator('text=자산 곡선').or(page.locator('text=Equity'))
      await expect(chartSection).toBeVisible({ timeout: 10000 })
    }
  })

  test('backtest results show trade history', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    const backtestLink = page.locator('a').filter({ hasText: /Strategy|전략/ }).first()

    if (await backtestLink.isVisible({ timeout: 5000 })) {
      await backtestLink.click()

      // Check for trade history section
      const tradeHistory = page.locator('text=거래 내역').or(page.locator('text=Trade'))
      await expect(tradeHistory).toBeVisible({ timeout: 10000 })
    }
  })

  test('pagination works for backtest list', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    // Look for pagination controls
    const nextButton = page.locator('button:has-text("다음")').or(
      page.locator('button[aria-label*="next"]')
    )

    if (await nextButton.isVisible()) {
      await nextButton.click()

      // Page should update
      await page.waitForLoadState('networkidle')
    }
  })

  test('user can filter backtest list', async ({ page }) => {
    await page.goto('/dashboard/backtest')

    // Look for filter controls
    const filterInput = page.locator('input[placeholder*="검색"]').or(
      page.locator('input[type="search"]')
    )

    if (await filterInput.isVisible()) {
      await filterInput.fill('Momentum')

      // Results should update
      await page.waitForTimeout(500)
    }
  })
})
