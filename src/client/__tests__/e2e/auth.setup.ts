import { test as setup, expect } from '@playwright/test'
import http from 'node:http'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const E2E_PASSWORD = 'e2etest'
const E2E_USER = 'e2e_user'
export const STORAGE_STATE = path.join(__dirname, '.auth/user.json')

// GKILL_E2E_BASE_URL でテスト対象サーバを上書きできる (既定: http://localhost:9999)
const gkillUrl = new URL(process.env.GKILL_E2E_BASE_URL ?? 'http://localhost:9999')
const gkillPort = Number(gkillUrl.port || 9999)

/**
 * Get the password reset token from gkill_server's redirect response.
 * On first run, gkill_server redirects / to /register_first_account?reset_token=<token>.
 * Returns empty string if no redirect (password already set).
 */
function getResetToken(): Promise<string> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: gkillUrl.hostname, port: gkillPort, path: '/', method: 'GET', timeout: 5000 },
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

setup('register and login', async ({ page }) => {
  setup.setTimeout(120000)

  // Ensure .auth/ directory exists
  fs.mkdirSync(path.dirname(STORAGE_STATE), { recursive: true })

  // 1. Perform initial registration if needed
  const token = await getResetToken()
  if (token) {
    await page.goto(`/register_first_account?reset_token=${token}`, { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // 固定sleepではなく入力欄が描画されるまで待つ (5つ目まで揃ってから count を読む)
    const inputs = page.locator('input')
    await expect(inputs.nth(4)).toBeVisible({ timeout: 30000 })
    const inputCount = await inputs.count()
    expect(inputCount).toBeGreaterThanOrEqual(5)

    // Fill registration form:
    // Input 0: ユーザID, Input 1: パスワード, Input 2: パスワード（再）,
    // Input 3: 管理者パスワード, Input 4: 管理者パスワード（再）
    await inputs.nth(0).fill(E2E_USER)
    await inputs.nth(1).fill(E2E_PASSWORD)
    await inputs.nth(2).fill(E2E_PASSWORD)
    await inputs.nth(3).fill(E2E_PASSWORD)
    await inputs.nth(4).fill(E2E_PASSWORD)

    const registerBtn = page.locator('button').filter({ hasText: /登録|regist/i }).first()
    await expect(registerBtn).toBeVisible()
    await registerBtn.click()
    // 登録完了で router.replace("/") される (use-register-first-account-view.ts)
    await page.waitForURL((url) => url.pathname === '/', { timeout: 60000 })
  }

  // 2. Login
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })

  // 固定sleepではなくログインフォームが描画されるまで待つ
  const inputs = page.locator('input')
  await expect(inputs.nth(1)).toBeVisible({ timeout: 30000 })
  expect(await inputs.count()).toBeGreaterThanOrEqual(2)

  await inputs.nth(0).fill(E2E_USER)
  await inputs.nth(1).fill(E2E_PASSWORD)

  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
  await expect(loginButton.first()).toBeVisible()
  await loginButton.first().click()
  // ログイン成功で router.replace("/" + default_page) される (use-login-view.ts)。
  // 固定sleepではなく "/" から離れることを待つ
  await page.waitForURL((url) => url.pathname !== '/', { timeout: 60000 })

  // Verify login succeeded (redirected away from login page)
  const url = page.url()
  expect(url.includes('/kftl') || url.includes('/rykv') || url.includes('/mi') || !url.endsWith('/')).toBeTruthy()

  // 3. Save storage state (cookies + localStorage)
  await page.context().storageState({ path: STORAGE_STATE })
})
