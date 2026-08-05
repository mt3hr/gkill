import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Register First Account Page', () => {
  test('can navigate to register first account page', async ({ page }) => {
    await page.goto('/register_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/register_first_account/)
  })

  test('legacy /regist_first_account redirects to /register_first_account', async ({ page }) => {
    // 旧パスはブックマークや古い資料から来る。reset_token を落とすと初回セットアップが通らない
    await page.goto('/regist_first_account?reset_token=dummy_token', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/register_first_account/)
    await expect(page).toHaveURL(/reset_token=dummy_token/)
  })

  test('register first account page renders app container', async ({ page }) => {
    await page.goto('/register_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('register first account page has input fields', async ({ page }) => {
    await page.goto('/register_first_account', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // Registration page should have input fields for account creation
    const inputs = page.locator('input')
    await expect(inputs.first()).toBeVisible({ timeout: 15000 })
  })
})
