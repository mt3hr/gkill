import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Plaing TimeIs Page', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    await loginAsAdmin(page)
  })

  test('can navigate to Plaing page', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/plaing/)
  })

  test('Plaing page renders app container', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('Plaing page has interactive elements', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    const buttons = page.locator('button')
    const buttonsCount = await buttons.count()
    expect(buttonsCount).toBeGreaterThan(0)
  })
})
