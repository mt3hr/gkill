import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Register First Account Page', () => {
  test('can navigate to register first account page', async ({ page }) => {
    await page.goto('/regist_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/regist_first_account/)
  })

  test('register first account page renders app container', async ({ page }) => {
    await page.goto('/regist_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('register first account page has input fields', async ({ page }) => {
    await page.goto('/regist_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // Registration page should have input fields for account creation
    const inputs = page.locator('input')
    const inputCount = await inputs.count()
    expect(inputCount).toBeGreaterThan(0)
  })
})
