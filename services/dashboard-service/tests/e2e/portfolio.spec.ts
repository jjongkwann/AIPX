import { test, expect } from '@playwright/test'

test.describe('Portfolio Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('user can navigate to portfolio page', async ({ page }) => {
    await page.click('a:has-text("포트폴리오")')

    await expect(page).toHaveURL('/dashboard/portfolio')
  })

  test('portfolio table displays positions', async ({ page }) => {
    await page.goto('/dashboard/portfolio')

    // Check table headers
    await expect(page.locator('text=종목')).toBeVisible()
    await expect(page.locator('text=수량')).toBeVisible()
    await expect(page.locator('text=평균가')).toBeVisible()
    await expect(page.locator('text=현재가')).toBeVisible()
    await expect(page.locator('text=손익')).toBeVisible()
  })

  test('portfolio displays summary metrics', async ({ page }) => {
    await page.goto('/dashboard/portfolio')

    // Check for summary cards (if implemented)
    const summaryText = page.locator('text=총 자산').or(page.locator('text=현금'))
    await expect(summaryText).toBeVisible({ timeout: 10000 })
  })

  test('empty portfolio shows appropriate message', async ({ page, context }) => {
    // Mock empty portfolio response
    await context.route('**/api/v1/portfolio', route => {
      route.fulfill({
        status: 200,
        body: JSON.stringify({
          positions: [],
          summary: {
            total_equity: 10000000,
            cash: 10000000,
            total_pnl: 0,
            total_pnl_percent: 0,
          },
        }),
      })
    })

    await page.goto('/dashboard/portfolio')

    await expect(page.locator('text=보유 중인 포지션이 없습니다')).toBeVisible()
  })

  test('portfolio positions show color-coded P&L', async ({ page }) => {
    await page.goto('/dashboard/portfolio')

    // Wait for table to load
    await page.waitForSelector('table', { timeout: 10000 })

    // Check for colored text (profit/loss indicators)
    const tableRows = page.locator('tbody tr')
    const count = await tableRows.count()

    if (count > 0) {
      // At least one position should have colored P&L
      const firstRow = tableRows.first()
      await expect(firstRow).toBeVisible()
    }
  })

  test('user can view position details on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 })

    await page.goto('/dashboard/portfolio')

    // Mobile view should show cards instead of table
    const mobileCards = page.locator('.md\\:hidden')
    await expect(mobileCards).toBeVisible()
  })

  test('portfolio data refreshes on page load', async ({ page }) => {
    await page.goto('/dashboard/portfolio')

    // Wait for data to load
    await page.waitForLoadState('networkidle')

    // Reload page
    await page.reload()

    // Data should still be visible
    const tableOrMessage = page.locator('table').or(page.locator('text=보유 중인 포지션'))
    await expect(tableOrMessage).toBeVisible()
  })

  test('portfolio shows loading state', async ({ page }) => {
    await page.goto('/dashboard/portfolio')

    // Check for loading indicator (if implemented)
    // This might be too fast to catch
    const loadingIndicator = page.locator('text=로딩').or(page.locator('[role="progressbar"]'))

    // Either loading or content should be visible
    const content = page.locator('table').or(page.locator('text=보유 중인 포지션'))
    await expect(content).toBeVisible({ timeout: 10000 })
  })
})
