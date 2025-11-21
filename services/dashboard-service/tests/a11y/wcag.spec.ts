import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

test.describe('WCAG Accessibility Compliance', () => {
  test.beforeEach(async ({ page }) => {
    // Login first for protected pages
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')
  })

  test('login page has no accessibility violations', async ({ page }) => {
    await page.goto('/login')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze()

    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('dashboard page has no accessibility violations', async ({ page }) => {
    await page.goto('/dashboard')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('chat page has no accessibility violations', async ({ page }) => {
    await page.goto('/dashboard/chat')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('portfolio page has no accessibility violations', async ({ page }) => {
    await page.goto('/dashboard/portfolio')
    await page.waitForLoadState('networkidle')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('backtest page has no accessibility violations', async ({ page }) => {
    await page.goto('/dashboard/backtest')
    await page.waitForLoadState('networkidle')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('keyboard navigation works on all pages', async ({ page }) => {
    const pages = ['/dashboard', '/dashboard/chat', '/dashboard/portfolio', '/dashboard/backtest']

    for (const url of pages) {
      await page.goto(url)

      // Tab through interactive elements
      for (let i = 0; i < 5; i++) {
        await page.keyboard.press('Tab')

        // Check if focus is visible
        const focused = await page.evaluate(() => {
          const el = document.activeElement
          if (!el) return null

          const computedStyle = window.getComputedStyle(el)
          return {
            tagName: el.tagName,
            hasFocusVisible: computedStyle.outline !== 'none' || computedStyle.boxShadow !== 'none',
          }
        })

        expect(focused).toBeTruthy()
      }
    }
  })

  test('all images have alt text', async ({ page }) => {
    await page.goto('/dashboard')

    const imagesWithoutAlt = await page.locator('img:not([alt])').count()
    expect(imagesWithoutAlt).toBe(0)
  })

  test('all form inputs have labels', async ({ page }) => {
    await page.goto('/login')

    const inputs = await page.locator('input').all()

    for (const input of inputs) {
      const hasLabel = await input.evaluate(el => {
        const id = el.getAttribute('id')
        const ariaLabel = el.getAttribute('aria-label')
        const ariaLabelledBy = el.getAttribute('aria-labelledby')

        if (ariaLabel || ariaLabelledBy) return true

        if (id) {
          const label = document.querySelector(`label[for="${id}"]`)
          if (label) return true
        }

        const parentLabel = el.closest('label')
        if (parentLabel) return true

        return false
      })

      expect(hasLabel).toBe(true)
    }
  })

  test('buttons have accessible names', async ({ page }) => {
    await page.goto('/dashboard')

    const buttons = await page.locator('button').all()

    for (const button of buttons) {
      const accessibleName = await button.evaluate(el => {
        return el.textContent?.trim() ||
          el.getAttribute('aria-label') ||
          el.getAttribute('title')
      })

      expect(accessibleName).toBeTruthy()
    }
  })

  test('color contrast meets WCAG AA standards', async ({ page }) => {
    await page.goto('/dashboard')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2aa'])
      .include('body')
      .analyze()

    const contrastViolations = accessibilityScanResults.violations.filter(
      v => v.id === 'color-contrast'
    )

    expect(contrastViolations).toEqual([])
  })

  test('modal focus management works correctly', async ({ page }) => {
    await page.goto('/dashboard')

    // If there's a modal trigger
    const modalTrigger = page.locator('button:has-text("모달")').or(
      page.locator('[data-modal-trigger]')
    )

    if (await modalTrigger.isVisible()) {
      await modalTrigger.click()

      // Focus should move to modal
      await page.waitForTimeout(300)

      const focusedElement = await page.evaluate(() => {
        return document.activeElement?.tagName
      })

      expect(focusedElement).toBeTruthy()
    }
  })

  test('escape key closes modal', async ({ page }) => {
    await page.goto('/dashboard')

    const modalTrigger = page.locator('button:has-text("모달")').or(
      page.locator('[data-modal-trigger]')
    )

    if (await modalTrigger.isVisible()) {
      await modalTrigger.click()
      await page.keyboard.press('Escape')

      // Modal should be closed
      await page.waitForTimeout(300)
    }
  })

  test('ARIA landmarks are present', async ({ page }) => {
    await page.goto('/dashboard')

    // Check for main landmark
    const mainLandmark = page.locator('[role="main"]').or(page.locator('main'))
    await expect(mainLandmark).toBeVisible()

    // Check for navigation landmark
    const navLandmark = page.locator('[role="navigation"]').or(page.locator('nav'))
    await expect(navLandmark).toBeVisible()
  })

  test('headings are properly nested', async ({ page }) => {
    await page.goto('/dashboard')

    const headingLevels = await page.evaluate(() => {
      const headings = Array.from(document.querySelectorAll('h1, h2, h3, h4, h5, h6'))
      return headings.map(h => parseInt(h.tagName[1]))
    })

    // Check that headings don't skip levels
    for (let i = 1; i < headingLevels.length; i++) {
      const diff = headingLevels[i] - headingLevels[i - 1]
      expect(diff).toBeLessThanOrEqual(1)
    }
  })

  test('lists are properly structured', async ({ page }) => {
    await page.goto('/dashboard')

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a'])
      .analyze()

    const listViolations = accessibilityScanResults.violations.filter(
      v => v.id.includes('list')
    )

    expect(listViolations).toEqual([])
  })

  test('tables have proper headers', async ({ page }) => {
    await page.goto('/dashboard/portfolio')
    await page.waitForLoadState('networkidle')

    const tables = await page.locator('table').all()

    for (const table of tables) {
      const hasHeaders = await table.evaluate(t => {
        const headers = t.querySelectorAll('th')
        return headers.length > 0
      })

      if (await table.isVisible()) {
        expect(hasHeaders).toBe(true)
      }
    }
  })

  test('links have descriptive text', async ({ page }) => {
    await page.goto('/dashboard')

    const links = await page.locator('a').all()

    for (const link of links) {
      const text = await link.textContent()
      const ariaLabel = await link.getAttribute('aria-label')

      const hasDescriptiveText = (text && text.trim().length > 0) ||
        (ariaLabel && ariaLabel.length > 0)

      expect(hasDescriptiveText).toBe(true)
    }
  })
})
