import { test as setup, expect, type Page } from '@playwright/test'
import http from 'node:http'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
// login.spec.ts など「実際にログインする」テストとも共有する（テストファイル同士は import できない）
import { E2E_USER, E2E_PASSWORD } from './e2e-credentials'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
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

/**
 * 初回起動なら最初のアカウントを登録する。
 *
 * 「登録済みかどうか」は環境の状態で決まる分岐なので、テスト本体には置かない。
 * （テスト本体の条件分岐は「対象が無いと何も検証せずに緑になる」形を招くので禁止してある。
 *   ここはセットアップの前提を整える処理で、失敗すれば下のログインが落ちる）
 */
async function registerFirstAccountIfNeeded(page: Page, token: string): Promise<void> {
  if (!token) {
    return
  }
  await page.goto(`/register_first_account?reset_token=${token}`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })

  // 固定sleepではなく入力欄が描画されるまで待つ (5つ目まで揃ってから使う)
  const inputs = page.locator('input')
  await expect(inputs.nth(4), '初回登録フォームが描かれない').toBeVisible({ timeout: 30000 })

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

setup('register and login', async ({ page }) => {
  setup.setTimeout(120000)

  // Ensure .auth/ directory exists
  fs.mkdirSync(path.dirname(STORAGE_STATE), { recursive: true })

  // 1. Perform initial registration if needed
  await registerFirstAccountIfNeeded(page, await getResetToken())

  // 2. Login
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })

  // 固定sleepではなくログインフォームが描画されるまで待つ
  const inputs = page.locator('input')
  await expect(inputs.nth(1), 'ログインフォームが描かれない').toBeVisible({ timeout: 30000 })

  await inputs.nth(0).fill(E2E_USER)
  await inputs.nth(1).fill(E2E_PASSWORD)

  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
  await expect(loginButton.first()).toBeVisible()
  await loginButton.first().click()
  // ログイン成功で router.replace("/" + default_page) される (use-login-view.ts)。
  // 固定sleepではなく "/" から離れることを待つ
  await page.waitForURL((url) => url.pathname !== '/', { timeout: 60000 })

  // Verify login succeeded (redirected away from login page)
  await expect(page, 'ログインしてもログイン画面から動いていない').not.toHaveURL(/\/(\?.*)?$/)

  // 3. Save storage state (cookies + localStorage)
  await page.context().storageState({ path: STORAGE_STATE })
})
