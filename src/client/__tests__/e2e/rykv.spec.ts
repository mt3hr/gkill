import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('RYKV Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('can navigate to RYKV page', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/rykv/)
  })

  test('RYKV page renders app container', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('RYKV page has interactive elements', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // RYKV is a record viewer; check for buttons or navigation elements
    const buttons = page.locator('button')
    const buttonsCount = await buttons.count()
    expect(buttonsCount).toBeGreaterThan(0)
  })

  test('RYKV page renders without JavaScript errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(5000)
    // Filter out known benign errors
    const criticalErrors = errors.filter(e =>
      !e.includes('ResizeObserver') &&
      !e.includes('devtools') &&
      !e.includes('[hmr]') &&
      !e.includes('Failed to fetch') &&
      !e.includes('Unexpected end of JSON input')
    )
    expect(criticalErrors.length).toBe(0)
  })

  test('RYKV page app content is substantial', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(5000)
    const textContent = await page.locator('#app').textContent()
    expect(textContent!.length).toBeGreaterThan(0)
  })

  test('RYKV page responds to mobile viewport', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    await page.setViewportSize({ width: 375, height: 812 })
    await page.waitForTimeout(1000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    await page.setViewportSize({ width: 1280, height: 720 })
  })

  test('RYKV page navigation does not lose URL', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/rykv/)
    // Reload and verify URL is maintained
    await page.reload({ waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/rykv/)
  })
})
