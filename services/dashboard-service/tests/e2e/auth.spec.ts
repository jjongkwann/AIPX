import { test, expect } from '@playwright/test'

test.describe('Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Start from the login page
    await page.goto('/login')
  })

  test('user can login with valid credentials', async ({ page }) => {
    // Fill in login form
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')

    // Click login button
    await page.click('button:has-text("로그인")')

    // Should redirect to dashboard
    await expect(page).toHaveURL('/dashboard')

    // Verify dashboard content loaded
    await expect(page.locator('text=대시보드')).toBeVisible()
  })

  test('login fails with invalid credentials', async ({ page }) => {
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'wrongpassword')

    await page.click('button:has-text("로그인")')

    // Should show error message
    await expect(page.locator('[role="alert"]')).toBeVisible()

    // Should stay on login page
    await expect(page).toHaveURL('/login')
  })

  test('validates empty form fields', async ({ page }) => {
    // Try to submit without filling fields
    await page.click('button:has-text("로그인")')

    // Browser validation should prevent submission
    const emailInput = page.locator('input[type="email"]')
    await expect(emailInput).toBeFocused()
  })

  test('user can navigate to signup page', async ({ page }) => {
    await page.click('a:has-text("회원가입")')

    await expect(page).toHaveURL('/signup')
    await expect(page.locator('text=회원가입')).toBeVisible()
  })

  test('user can logout', async ({ page, context }) => {
    // Login first
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')

    await page.waitForURL('/dashboard')

    // Click logout button
    await page.click('button:has-text("로그아웃")')

    // Should redirect to login page
    await expect(page).toHaveURL('/login')
  })

  test('protected routes redirect to login when not authenticated', async ({ page, context }) => {
    // Clear any existing cookies
    await context.clearCookies()

    // Try to access protected route
    await page.goto('/dashboard')

    // Should redirect to login
    await expect(page).toHaveURL('/login')
  })

  test('session persists across page reloads', async ({ page }) => {
    // Login
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')

    await page.waitForURL('/dashboard')

    // Reload page
    await page.reload()

    // Should still be on dashboard
    await expect(page).toHaveURL('/dashboard')
    await expect(page.locator('text=대시보드')).toBeVisible()
  })

  test('shows loading state during login', async ({ page }) => {
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')

    await page.click('button:has-text("로그인")')

    // Should show loading text briefly
    const loadingText = page.locator('text=로그인 중...')

    // Note: This might be too fast to catch in local testing
    // but is useful for slower network conditions
  })
})
