import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('MKFL Page', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    await loginAsAdmin(page)
  })

  test('can navigate to MKFL page', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/mkfl/)
  })

  test('MKFL page renders app container', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('MKFL page has interactive elements', async ({ page }) => {
    await page.goto('/mkfl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    const buttons = page.locator('button')
    const buttonsCount = await buttons.count()
    expect(buttonsCount).toBeGreaterThan(0)
  })
})
