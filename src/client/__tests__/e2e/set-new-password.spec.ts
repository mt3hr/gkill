import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
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
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('set new password page has password input fields', async ({ page }) => {
    await page.goto('/set_new_password', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // Password change page should have input fields for new password
    const inputs = page.locator('input')
    const inputCount = await inputs.count()
    expect(inputCount).toBeGreaterThan(0)
  })
})
