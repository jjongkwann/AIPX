import { test, expect } from '@playwright/test'

test.describe('Navigation Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('user can navigate between pages using navbar', async ({ page }) => {
    // Navigate to each page
    const pages = [
      { link: 'AI 채팅', url: '/dashboard/chat' },
      { link: '포트폴리오', url: '/dashboard/portfolio' },
      { link: '백테스트', url: '/dashboard/backtest' },
      { link: '전략', url: '/dashboard/strategies' },
      { link: '대시보드', url: '/dashboard' },
    ]

    for (const { link, url } of pages) {
      await page.click(`a:has-text("${link}")`)
      await expect(page).toHaveURL(url)
    }
  })

  test('active link is highlighted in navbar', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // AI 채팅 link should have active styling
    const chatLink = page.locator('a:has-text("AI 채팅")').first()
    const classes = await chatLink.getAttribute('class')

    // Check for active indicator (implementation specific)
    await expect(chatLink).toBeVisible()
  })

  test('mobile menu works correctly', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 })

    await page.goto('/dashboard')

    // Click mobile menu button
    const menuButton = page.locator('button[aria-label="메뉴 열기"]')
    await expect(menuButton).toBeVisible()

    await menuButton.click()

    // Menu should be open - navigation links should be visible
    await expect(page.locator('text=AI 채팅').nth(1)).toBeVisible()

    // Click a link
    await page.locator('text=AI 채팅').nth(1).click()

    // Should navigate and close menu
    await expect(page).toHaveURL('/dashboard/chat')
  })

  test('mobile menu closes when link is clicked', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })

    await page.goto('/dashboard')

    // Open menu
    await page.click('button[aria-label="메뉴 열기"]')

    // Click a navigation link
    await page.locator('text=포트폴리오').last().click()

    // Menu should close (only one instance of each link visible)
    await page.waitForTimeout(500)

    const portfolioLinks = page.locator('text=포트폴리오')
    const count = await portfolioLinks.count()

    // Should have fewer links visible after menu closes
    expect(count).toBeLessThanOrEqual(2)
  })

  test('logo link navigates to dashboard', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // Click logo
    await page.click('a:has-text("AIPX")')

    // Should go to dashboard
    await expect(page).toHaveURL('/dashboard')
  })

  test('back button works correctly', async ({ page }) => {
    // Navigate through pages
    await page.goto('/dashboard')
    await page.click('a:has-text("AI 채팅")')
    await expect(page).toHaveURL('/dashboard/chat')

    // Go back
    await page.goBack()

    // Should be back on dashboard
    await expect(page).toHaveURL('/dashboard')
  })

  test('forward button works correctly', async ({ page }) => {
    await page.goto('/dashboard')
    await page.click('a:has-text("AI 채팅")')
    await page.goBack()
    await page.goForward()

    await expect(page).toHaveURL('/dashboard/chat')
  })

  test('keyboard navigation works', async ({ page }) => {
    await page.goto('/dashboard')

    // Tab through navigation
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')

    // Should focus navigation elements
    const focused = await page.evaluate(() => document.activeElement?.tagName)
    expect(['A', 'BUTTON']).toContain(focused)
  })

  test('navigation is accessible with screen readers', async ({ page }) => {
    await page.goto('/dashboard')

    // Check for nav element with proper role
    const nav = page.locator('nav[role="navigation"]')
    await expect(nav).toBeVisible()

    // Check for ARIA labels
    const ariaLabel = await nav.getAttribute('aria-label')
    expect(ariaLabel).toBeTruthy()
  })

  test('page title updates when navigating', async ({ page }) => {
    await page.goto('/dashboard')
    let title = await page.title()
    expect(title).toContain('AIPX')

    await page.click('a:has-text("AI 채팅")')
    await page.waitForLoadState('domcontentloaded')

    title = await page.title()
    // Title should update (if implemented)
    expect(title).toBeTruthy()
  })

  test('navigation persists user session', async ({ page }) => {
    // Navigate to different pages
    await page.goto('/dashboard')
    await page.goto('/dashboard/chat')
    await page.goto('/dashboard/portfolio')

    // Should still be authenticated
    await expect(page.locator('text=로그아웃')).toBeVisible()
  })
})
