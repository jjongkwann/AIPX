import { test, expect } from '@playwright/test'

test.describe('Chat Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('user can navigate to chat page', async ({ page }) => {
    // Click on chat navigation
    await page.click('a:has-text("AI 채팅")')

    await expect(page).toHaveURL('/dashboard/chat')
    await expect(page.locator('text=AI 에이전트')).toBeVisible()
  })

  test('chat interface displays connection status', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // Check for connection indicator
    const connectionStatus = page.locator('text=연결됨').or(page.locator('text=재연결 중'))
    await expect(connectionStatus).toBeVisible()
  })

  test('user can type and send message', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // Wait for connection
    await page.waitForSelector('text=연결됨', { timeout: 5000 })

    // Type message
    const messageInput = page.locator('input[placeholder*="메시지"]')
    await messageInput.fill('삼성전자 주가 분석해줘')

    // Send message
    await page.click('button[aria-label="메시지 전송"]')

    // Message should appear in chat
    await expect(page.locator('text=삼성전자 주가 분석해줘')).toBeVisible()

    // Input should be cleared
    await expect(messageInput).toHaveValue('')
  })

  test('send button is disabled when input is empty', async ({ page }) => {
    await page.goto('/dashboard/chat')

    const sendButton = page.locator('button[aria-label="메시지 전송"]')

    // Should be disabled initially
    await expect(sendButton).toBeDisabled()

    // Type something
    await page.fill('input[placeholder*="메시지"]', 'test')

    // Should be enabled now
    await expect(sendButton).toBeEnabled()
  })

  test('user can send message with Enter key', async ({ page }) => {
    await page.goto('/dashboard/chat')

    await page.waitForSelector('text=연결됨', { timeout: 5000 })

    // Type and press Enter
    const messageInput = page.locator('input[placeholder*="메시지"]')
    await messageInput.fill('테스트 메시지')
    await messageInput.press('Enter')

    // Message should appear
    await expect(page.locator('text=테스트 메시지')).toBeVisible()
  })

  test('displays empty state when no messages', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // Check for empty state message
    await expect(page.locator('text=AI 에이전트에게 질문해보세요')).toBeVisible()
  })

  test('chat is disabled when disconnected', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // If disconnected, input and button should be disabled
    const reconnectingText = page.locator('text=재연결 중')

    if (await reconnectingText.isVisible()) {
      const messageInput = page.locator('input[placeholder*="메시지"]')
      const sendButton = page.locator('button[aria-label="메시지 전송"]')

      await expect(messageInput).toBeDisabled()
      await expect(sendButton).toBeDisabled()
    }
  })

  test('chat messages scroll to bottom automatically', async ({ page }) => {
    await page.goto('/dashboard/chat')

    await page.waitForSelector('text=연결됨', { timeout: 5000 })

    // Send multiple messages
    for (let i = 1; i <= 5; i++) {
      const messageInput = page.locator('input[placeholder*="메시지"]')
      await messageInput.fill(`메시지 ${i}`)
      await messageInput.press('Enter')
      await page.waitForTimeout(500)
    }

    // Last message should be visible
    await expect(page.locator('text=메시지 5')).toBeVisible()
  })

  test('displays connection status indicator', async ({ page }) => {
    await page.goto('/dashboard/chat')

    // Check for status indicator dot
    const statusDot = page.locator('div[aria-label*="연결"]')
    await expect(statusDot).toBeVisible()
  })
})
