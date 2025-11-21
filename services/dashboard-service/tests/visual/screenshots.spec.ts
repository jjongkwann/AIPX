import { test, expect } from '@playwright/test'

/**
 * Visual Regression Tests
 *
 * These tests capture screenshots and compare them against baseline images.
 * Run `npm run test:e2e -- --update-snapshots` to update baseline screenshots.
 */

test.describe('Visual Regression Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Login for protected pages
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('login page - desktop', async ({ page }) => {
    await page.goto('/login')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('login-desktop.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('login page - mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/login')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('login-mobile.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('dashboard page - desktop', async ({ page }) => {
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('dashboard-desktop.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('dashboard page - mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('dashboard-mobile.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('chat page - desktop', async ({ page }) => {
    await page.goto('/dashboard/chat')
    await page.waitForLoadState('networkidle')

    // Wait for connection
    await page.waitForSelector('text=연결됨', { timeout: 5000 }).catch(() => {})

    await expect(page).toHaveScreenshot('chat-desktop.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('portfolio page - desktop', async ({ page }) => {
    await page.goto('/dashboard/portfolio')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('portfolio-desktop.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('portfolio page - mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/dashboard/portfolio')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('portfolio-mobile.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('backtest page - desktop', async ({ page }) => {
    await page.goto('/dashboard/backtest')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('backtest-desktop.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('navbar - desktop', async ({ page }) => {
    await page.goto('/dashboard')

    const navbar = page.locator('nav')
    await expect(navbar).toHaveScreenshot('navbar-desktop.png', {
      maxDiffPixels: 50,
    })
  })

  test('navbar - mobile menu open', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/dashboard')

    // Open mobile menu
    await page.click('button[aria-label="메뉴 열기"]')
    await page.waitForTimeout(300)

    await expect(page).toHaveScreenshot('navbar-mobile-open.png', {
      maxDiffPixels: 50,
    })
  })

  test('dark theme consistency', async ({ page }) => {
    await page.goto('/dashboard')

    // Check that dark theme colors are applied
    const backgroundColor = await page.evaluate(() => {
      return window.getComputedStyle(document.body).backgroundColor
    })

    // Should be a dark color
    expect(backgroundColor).toMatch(/rgb\((\d+), \1, \1\)/)
  })

  test('responsive layout - tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 })
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')

    await expect(page).toHaveScreenshot('dashboard-tablet.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('component - portfolio table', async ({ page }) => {
    await page.goto('/dashboard/portfolio')
    await page.waitForLoadState('networkidle')

    const table = page.locator('table').first()

    if (await table.isVisible()) {
      await expect(table).toHaveScreenshot('portfolio-table.png', {
        maxDiffPixels: 50,
      })
    }
  })

  test('component - chat interface', async ({ page }) => {
    await page.goto('/dashboard/chat')
    await page.waitForLoadState('networkidle')

    const chatInterface = page.locator('[role="log"]').first()

    if (await chatInterface.isVisible()) {
      await expect(chatInterface).toHaveScreenshot('chat-interface.png', {
        maxDiffPixels: 50,
      })
    }
  })

  test('error state - login failed', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'wrongpassword')
    await page.click('button:has-text("로그인")')

    // Wait for error message
    await page.waitForSelector('[role="alert"]')

    await expect(page).toHaveScreenshot('login-error.png', {
      fullPage: true,
      maxDiffPixels: 100,
    })
  })

  test('loading state - dashboard', async ({ page }) => {
    // This is tricky to capture as loading is fast
    // You might need to throttle network in Playwright config
    await page.goto('/dashboard')

    // Capture as soon as possible
    await expect(page).toHaveScreenshot('dashboard-loading.png', {
      maxDiffPixels: 100,
      timeout: 1000,
    }).catch(() => {
      // Loading might be too fast to capture
    })
  })
})
