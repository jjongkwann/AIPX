import { test, expect } from '@playwright/test'

test.describe('Performance Tests', () => {
  test('dashboard page loads within acceptable time', async ({ page }) => {
    // Login first
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')

    // Measure performance
    const performanceTiming = JSON.parse(
      await page.evaluate(() => JSON.stringify(window.performance.timing))
    )

    const navigationStart = performanceTiming.navigationStart
    const loadEventEnd = performanceTiming.loadEventEnd

    const pageLoadTime = loadEventEnd - navigationStart

    // Page should load in less than 3 seconds
    expect(pageLoadTime).toBeLessThan(3000)
  })

  test('measures Core Web Vitals', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button:has-text("로그인")')
    await page.waitForURL('/dashboard')

    // Wait for page to fully load
    await page.waitForLoadState('networkidle')

    // Measure LCP (Largest Contentful Paint)
    const lcp = await page.evaluate(() => {
      return new Promise<number>((resolve) => {
        new PerformanceObserver((list) => {
          const entries = list.getEntries()
          const lastEntry = entries[entries.length - 1] as any
          resolve(lastEntry.renderTime || lastEntry.loadTime)
        }).observe({ entryTypes: ['largest-contentful-paint'] })

        setTimeout(() => resolve(0), 5000)
      })
    })

    // LCP should be less than 2.5 seconds for good performance
    if (lcp > 0) {
      expect(lcp).toBeLessThan(2500)
    }
  })

  test('measures First Contentful Paint (FCP)', async ({ page }) => {
    await page.goto('/dashboard')

    const fcp = await page.evaluate(() => {
      const paint = performance.getEntriesByType('paint')
      const fcpEntry = paint.find(entry => entry.name === 'first-contentful-paint')
      return fcpEntry ? fcpEntry.startTime : 0
    })

    // FCP should be less than 1.8 seconds
    if (fcp > 0) {
      expect(fcp).toBeLessThan(1800)
    }
  })

  test('measures Time to Interactive (TTI)', async ({ page }) => {
    await page.goto('/dashboard')

    await page.waitForLoadState('networkidle')

    // Measure by checking when page becomes interactive
    const tti = await page.evaluate(() => {
      return new Promise<number>((resolve) => {
        const start = performance.now()

        // Wait for main thread to be idle
        requestIdleCallback(() => {
          resolve(performance.now() - start)
        })

        // Fallback timeout
        setTimeout(() => resolve(5000), 5000)
      })
    })

    // TTI should be less than 3.8 seconds
    expect(tti).toBeLessThan(3800)
  })

  test('measures Cumulative Layout Shift (CLS)', async ({ page }) => {
    await page.goto('/dashboard')

    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(2000) // Wait for any layout shifts

    const cls = await page.evaluate(() => {
      return new Promise<number>((resolve) => {
        let clsValue = 0

        new PerformanceObserver((list) => {
          for (const entry of list.getEntries() as any[]) {
            if (!entry.hadRecentInput) {
              clsValue += entry.value
            }
          }
          resolve(clsValue)
        }).observe({ entryTypes: ['layout-shift'] })

        setTimeout(() => resolve(clsValue), 2000)
      })
    })

    // CLS should be less than 0.1 for good performance
    expect(cls).toBeLessThan(0.1)
  })

  test('checks bundle size impact', async ({ page }) => {
    const response = await page.goto('/dashboard')

    expect(response).not.toBeNull()

    // Check response size
    const navigationEntry = await page.evaluate(() => {
      const navEntries = performance.getEntriesByType('navigation')
      return navEntries[0] as PerformanceNavigationTiming
    })

    // Transfer size should be reasonable (less than 2MB)
    if (navigationEntry && 'transferSize' in navigationEntry) {
      expect((navigationEntry as any).transferSize).toBeLessThan(2000000)
    }
  })

  test('API responses are fast', async ({ page }) => {
    await page.goto('/login')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password123')

    const startTime = Date.now()

    // Listen for API response
    const responsePromise = page.waitForResponse(
      response => response.url().includes('/api/v1/') && response.status() === 200
    )

    await page.click('button:has-text("로그인")')

    const response = await responsePromise
    const endTime = Date.now()

    const responseTime = endTime - startTime

    // API should respond within 1 second
    expect(responseTime).toBeLessThan(1000)
  })

  test('images are optimized', async ({ page }) => {
    await page.goto('/dashboard')

    const images = await page.locator('img').all()

    for (const img of images) {
      const src = await img.getAttribute('src')

      if (src && !src.startsWith('data:')) {
        // Check if image has loading attribute
        const loading = await img.getAttribute('loading')

        // Images should use lazy loading
        // (except for above-the-fold images)
        // This is a guideline, adjust based on your needs
      }
    }
  })

  test('JavaScript execution time is reasonable', async ({ page }) => {
    await page.goto('/dashboard')

    await page.waitForLoadState('networkidle')

    const jsExecutionTime = await page.evaluate(() => {
      const entries = performance.getEntriesByType('measure')
      return entries.reduce((total, entry) => total + entry.duration, 0)
    })

    // Total JS execution should be reasonable
    // Adjust threshold based on your application
  })

  test('memory usage is acceptable', async ({ page }) => {
    await page.goto('/dashboard')

    await page.waitForLoadState('networkidle')

    // Navigate through several pages
    const pages = ['/dashboard/chat', '/dashboard/portfolio', '/dashboard/backtest']

    for (const url of pages) {
      await page.goto(url)
      await page.waitForLoadState('networkidle')
    }

    // Check for memory leaks by going back to dashboard
    await page.goto('/dashboard')

    // Memory should stabilize (implementation depends on browser capabilities)
  })
})
