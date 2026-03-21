import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('can navigate to settings page', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/saihate/)
  })

  test('settings page renders content', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const appContent = await page.locator('#app').innerHTML()
    expect(appContent.length).toBeGreaterThan(100)
  })

  test('settings page renders without JavaScript errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
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

  test('settings page has buttons or interactive controls', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const buttons = page.locator('button')
    const inputs = page.locator('input')
    const switches = page.locator('.v-switch, [role="switch"]')
    const totalInteractive = (await buttons.count()) + (await inputs.count()) + (await switches.count())
    // Settings page should have some controls
    expect(totalInteractive).toBeGreaterThan(0)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
