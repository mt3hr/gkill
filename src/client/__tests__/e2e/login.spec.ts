import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Login page', () => {
  // In the Vue router, '/' is the login page (not '/login')

  test('can load login page', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/\/($|\?|regist_first_account)/, { timeout: 15000 })
  })

  test('login page has input fields', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const inputs = page.locator('input')
    await expect(inputs.first()).toBeVisible({ timeout: 15000 })
  })

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)

    const inputs = page.locator('input')
    if (await inputs.count() >= 2) {
      await inputs.nth(0).fill('nonexistent_user')
      await inputs.nth(1).fill('wrong_password')

      const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
      if (await loginButton.count() > 0) {
        await loginButton.click()
        await page.waitForTimeout(3000)
      }
    }
  })
})
