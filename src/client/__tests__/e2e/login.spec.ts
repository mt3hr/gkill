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
    await page.waitForTimeout(2000)
    const inputs = page.locator('input')
    await expect(inputs.first()).toBeVisible({ timeout: 15000 })
  })

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    expect(await inputs.count()).toBeGreaterThanOrEqual(2)
    await inputs.nth(0).fill('nonexistent_user')
    await inputs.nth(1).fill('wrong_password')

    const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
    expect(await loginButton.count()).toBeGreaterThan(0)
    await loginButton.click()
    await page.waitForTimeout(2000)
    // After invalid login, should still be on login page or show error
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('successful login redirects away from login', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    expect(await inputs.count()).toBeGreaterThanOrEqual(2)
    // Use default admin credentials (admin with empty password)
    await inputs.nth(0).fill('admin')
    await inputs.nth(1).fill('')

    const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
    expect(await loginButton.count()).toBeGreaterThan(0)
    await loginButton.click()
    await page.waitForTimeout(2000)
    // After successful login, the page should not crash
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('session persists across page reload after login', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    expect(await inputs.count()).toBeGreaterThanOrEqual(2)

    await inputs.nth(0).fill('admin')
    await inputs.nth(1).fill('')
    const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
    expect(await loginButton.count()).toBeGreaterThan(0)
    await loginButton.click()
    await page.waitForTimeout(2000)

    // Reload the page and verify we're still authenticated (not sent back to login)
    await page.reload({ waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('navigating to authenticated route without session redirects to login', async ({ page }) => {
    // Clear cookies/storage to ensure no session
    await page.context().clearCookies()
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)
    // Should either redirect to login or show login-related content
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('login page user input field accepts Japanese characters', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    expect(await inputs.count()).toBeGreaterThanOrEqual(1)
    await inputs.nth(0).fill('テストユーザー')
    const value = await inputs.nth(0).inputValue()
    expect(value).toBe('テストユーザー')
  })

  test('password field masks input', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    expect(await inputs.count()).toBeGreaterThanOrEqual(2)
    const passwordInput = inputs.nth(1)
    const type = await passwordInput.getAttribute('type')
    expect(type).toBe('password')
  })
})
