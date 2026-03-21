import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Shared Mi Page', () => {
  test('shared mi page loads without crashing', async ({ page }) => {
    // Navigate without a share ID parameter; page should still load without crashing
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
  })

  test('shared mi page renders app container', async ({ page }) => {
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('shared mi page does not show fatal error', async ({ page }) => {
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // Verify no uncaught JS errors caused a blank page
    const appContent = await page.locator('#app').innerHTML()
    expect(appContent.length).toBeGreaterThan(0)
  })
})
