import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { E2E_USER, E2E_PASSWORD } from './e2e-credentials'
import { openLoginPage, submitLogin } from './helpers'

// This spec tests unauthenticated flows — clear storageState from setup project
test.use({ storageState: { cookies: [], origins: [] } })

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Login page', () => {
  // In the Vue router, '/' is the login page (not '/login')

  test('can load login page', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/\/($|\?|register_first_account)/, { timeout: 15000 })
  })

  test('login page has input fields', async ({ page }) => {
    const inputs = await openLoginPage(page)
    await expect(inputs.first()).toBeVisible()
  })

  test('login with invalid credentials shows error', async ({ page }) => {
    await submitLogin(page, 'nonexistent_user', 'wrong_password')

    // login-page.vue はエラーを role="alert" の v-alert で出す。
    // 「#app が見えている」だけの確認だと、何が起きても緑になってしまう
    await expect(page.locator('.v-alert[role="alert"]').first(), 'ログイン失敗のエラーが出ない')
      .toBeVisible({ timeout: 30000 })
    // 失敗したのだからログイン画面から動いていないこと
    await expect(page).toHaveURL(/\/(\?.*)?$/)
  })

  test('successful login redirects away from login', async ({ page }) => {
    await submitLogin(page, E2E_USER, E2E_PASSWORD)

    // 成功すると既定ページ（/kftl など）へ置き換わる
    await expect(page, 'ログインしても画面が変わらない')
      .not.toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })
  })

  test('session persists across page reload after login', async ({ page }) => {
    await submitLogin(page, E2E_USER, E2E_PASSWORD)
    await expect(page).not.toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })

    const after_login_url = new URL(page.url()).pathname
    await page.reload({ waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // 再読込でログイン画面へ戻されないこと（= セッションが残っている）
    await expect(page, '再読込でログイン画面へ戻された')
      .toHaveURL(new RegExp(`${after_login_url.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`), { timeout: 30000 })
  })

  test('navigating to authenticated route without session redirects to login', async ({ page }) => {
    // Clear cookies/storage to ensure no session
    await page.context().clearCookies()
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // セッションが無ければログイン画面へ送られる
    await expect(page, 'セッション無しでも認証必要ページに留まっている')
      .toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })
  })

  test('login page user input field accepts Japanese characters', async ({ page }) => {
    const inputs = await openLoginPage(page)
    await inputs.nth(0).fill('テストユーザー')
    await expect(inputs.nth(0)).toHaveValue('テストユーザー')
  })

  test('password field masks input', async ({ page }) => {
    const inputs = await openLoginPage(page)
    await expect(inputs.nth(1)).toHaveAttribute('type', 'password')
  })
})
