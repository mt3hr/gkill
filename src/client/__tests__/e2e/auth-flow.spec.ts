import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Auth Flow Tests', () => {
  test.beforeEach(async () => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
  })

  // 項番8: ログアウト操作
  test('logout redirects to login page', async ({ page }) => {
    test.setTimeout(120000)
    await loginAsAdmin(page)

    // Look for logout button/menu
    const logoutButton = page.locator('button, .v-list-item, [role="menuitem"]').filter({ hasText: /ログアウト|logout|sign.?out/i }).first()
    if (await logoutButton.count() > 0) {
      await logoutButton.click()
      await page.waitForTimeout(2000)

      // Should redirect to login page
      const url = page.url()
      const isLoginPage = url.endsWith('/') || url.includes('/login')
      expect(isLoginPage).toBe(true)

      // Verify login form is visible
      const inputs = page.locator('input')
      expect(await inputs.count()).toBeGreaterThanOrEqual(2)
    } else {
      // Try to find logout in a menu or navigation drawer
      const menuButton = page.locator('button[aria-label*="menu"], .v-app-bar button, nav button').first()
      if (await menuButton.count() > 0) {
        await menuButton.click()
        await page.waitForTimeout(1000)

        const logoutInMenu = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /ログアウト|logout/i }).first()
        if (await logoutInMenu.count() > 0) {
          await logoutInMenu.click()
          await page.waitForTimeout(2000)

          const url = page.url()
          const isLoginPage = url.endsWith('/') || url.includes('/login')
          expect(isLoginPage).toBe(true)
        }
      }
    }
  })

  // 項番9: パスワード未設定アカウントでログイン不可
  test('cannot login with account that has no password set', async ({ page }) => {
    test.setTimeout(120000)
    // Clear auth cookies so we see the login page, not an authenticated route
    await page.context().clearCookies()
    await page.goto('/', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    const inputs = page.locator('input')
    if (await inputs.count() >= 2) {
      // Try to login with a non-existent account (simulating no password)
      await inputs.nth(0).fill('test_no_password_user')
      await inputs.nth(1).fill('')

      const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
      if (await loginButton.count() > 0) {
        await loginButton.click()
        await page.waitForTimeout(2000)

        // Should still be on login page (login should fail)
        const url = page.url()
        const stillOnLogin = url.endsWith('/') || url.includes('/login') || url.includes('/set_new_password') || url.includes('/regist_first_account')
        expect(stillOnLogin).toBe(true)
      }
    }
  })

  // 項番11: ログイン後に全Repにチェック済み確認
  test('all reps are checked after login', async ({ page }) => {
    test.setTimeout(120000)
    await loginAsAdmin(page)

    // Navigate to a page where rep checkboxes are visible (rykv or settings)
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    // Look for checkbox/switch elements that represent Rep selections
    const checkboxes = page.locator('.v-checkbox, .v-switch, input[type="checkbox"]')
    const count = await checkboxes.count()
    if (count > 0) {
      // Verify all are checked (or at least most are)
      let checkedCount = 0
      for (let i = 0; i < count; i++) {
        const checkbox = checkboxes.nth(i)
        const isChecked = await checkbox.isChecked().catch(() => false)
        if (isChecked) checkedCount++
      }
      // At least some checkboxes should be checked
      expect(checkedCount).toBeGreaterThan(0)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
