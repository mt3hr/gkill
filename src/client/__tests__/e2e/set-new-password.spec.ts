import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Set New Password Page', () => {
  test('can navigate to set new password page', async ({ page }) => {
    await page.goto('/set_new_password', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/set_new_password/)
  })

  test('set new password page renders app container', async ({ page }) => {
    await page.goto('/set_new_password', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('set new password page has password input fields', async ({ page }) => {
    await page.goto('/set_new_password', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // Password change page should have input fields for new password
    await expect(page.locator('input').first(), 'パスワードの入力欄が描かれない')
      .toBeVisible({ timeout: 30000 })
  })
})
