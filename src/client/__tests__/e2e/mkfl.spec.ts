import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('MKFL Page', () => {
  test('can navigate to MKFL page', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/mkfl/)
  })

  test('MKFL page renders app container', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('MKFL page has interactive elements', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // MKFL is a memo/file listing page; check for UI elements
    const buttons = page.locator('button')
    const buttonsCount = await buttons.count()
    expect(buttonsCount).toBeGreaterThan(0)
  })
})
