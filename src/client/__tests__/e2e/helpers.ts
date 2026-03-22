import type { Page } from '@playwright/test'
import http from 'node:http'

const E2E_PASSWORD = 'e2etest'
const E2E_USER = 'e2e_user'

/** Track whether initial registration has been completed in this test run */
let registrationComplete = false

/**
 * Get the password reset token from gkill_server's redirect response.
 * On first run, gkill_server redirects / to /regist_first_account?reset_token=<token>.
 * Returns empty string if no redirect (password already set).
 */
function getResetToken(): Promise<string> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: '127.0.0.1', port: 9999, path: '/', method: 'GET', timeout: 5000 },
      (res) => {
        const location = res.headers['location'] || ''
        const match = location.match(/reset_token=([^&]+)/)
        resolve(match ? match[1] : '')
      },
    )
    req.on('error', () => resolve(''))
    req.on('timeout', () => { req.destroy(); resolve('') })
    req.end()
  })
}

/**
 * Perform initial account registration via the browser UI.
 * This sets the admin password and creates a test user account.
 * Must be done once per clean gkill_server startup.
 */
async function performInitialRegistration(page: Page): Promise<boolean> {
  if (registrationComplete) return true

  const token = await getResetToken()
  if (!token) {
    // No redirect — registration already done
    registrationComplete = true
    return true
  }

  // Navigate to registration page with reset token
  await page.goto(`/regist_first_account?reset_token=${token}`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(2000)

  const inputs = page.locator('input')
  const inputCount = await inputs.count()
  if (inputCount < 5) return false

  // Fill registration form:
  // Input 0: ユーザID (new account user id)
  // Input 1: パスワード (new account password)
  // Input 2: パスワード（再） (retype)
  // Input 3: 管理者パスワード (admin password)
  // Input 4: 管理者パスワード（再） (admin retype)
  await inputs.nth(0).fill(E2E_USER)
  await inputs.nth(1).fill(E2E_PASSWORD)
  await inputs.nth(2).fill(E2E_PASSWORD)
  await inputs.nth(3).fill(E2E_PASSWORD)
  await inputs.nth(4).fill(E2E_PASSWORD)

  // Click registration button
  const registerBtn = page.locator('button').filter({ hasText: /登録|regist/i }).first()
  if (await registerBtn.count() === 0) return false

  await registerBtn.click()
  // Wait for registration to complete (multi-step API flow — needs more time)
  await page.waitForTimeout(8000)

  registrationComplete = true
  return true
}

/**
 * Login as the E2E test user.
 * On first call, performs initial registration if needed.
 * Then navigates to login page, fills credentials, clicks login, waits for redirect.
 * Returns true if login succeeded (URL changed), false otherwise.
 */
export async function loginAsAdmin(page: Page): Promise<boolean> {
  await performInitialRegistration(page)

  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(2000)

  const inputs = page.locator('input')
  if (await inputs.count() < 2) return false

  await inputs.nth(0).fill(E2E_USER)
  await inputs.nth(1).fill(E2E_PASSWORD)

  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
  if (await loginButton.count() === 0) return false

  await loginButton.click()
  await page.waitForTimeout(2000)

  const url = page.url()
  // Login should redirect away from '/' to another page
  return !url.endsWith('/') || url.includes('/kftl') || url.includes('/rykv') || url.includes('/mi')
}
