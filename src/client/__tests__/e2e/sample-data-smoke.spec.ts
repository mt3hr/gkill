import { test, expect } from '@playwright/test'
import http from 'node:http'

/**
 * 配布サンプルデータ (resources/gkill_sample_data) の起動スモーク。
 *
 * run-e2e.mjs がサンプルデータのコピーを home にした gkill_server を別ポートで起動し、
 * URL を GKILL_E2E_SAMPLE_URL で渡してくる。このspecは Vite を経由せず、
 * サンプルサーバの embed 配信フロントエンドを直接踏む
 * （= 利用者が LAUNCH_GKILL_SAMPLE_DATA.bat で体験するのと同じ経路）。
 *
 * README 記載の資格情報でログインし、rykv に記録が出ることまでを1本で確認する。
 * ログインはレート制限 (IP毎15分10回) を消費するので、この1回に留めること。
 */

const sampleUrl = process.env.GKILL_E2E_SAMPLE_URL ?? ''

// 自前のログインを検証するので setup の storageState は使わない
test.use({ storageState: { cookies: [], origins: [] } })

/** サンプルサーバに到達できるか (check-server.ts の checkGkillServer と同じ手順) */
function checkSampleServer(): Promise<boolean> {
  return new Promise((resolve) => {
    const url = new URL(sampleUrl)
    const req = http.request(
      { hostname: url.hostname, port: Number(url.port), path: '/', method: 'GET', timeout: 10000 },
      () => resolve(true),
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.end()
  })
}

test.beforeAll(async () => {
  const alive = sampleUrl !== '' && await checkSampleServer()
  test.skip(!alive, 'sample-data gkill server is not running (npm run test_client_e2e 経由で実行すること)')
})

test('サンプルデータのサーバへログインでき、rykv に記録が表示される', async ({ page }) => {
  test.setTimeout(120000)

  // ログイン (資格情報は resources/gkill_sample_data/README.txt 記載の配布値)
  await page.goto(`${sampleUrl}/`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  const inputs = page.locator('input')
  await expect(inputs.nth(1), 'ログイン画面の入力欄が描かれない').toBeVisible({ timeout: 30000 })
  await inputs.nth(0).fill('gkill_sample_data')
  await inputs.nth(1).fill('sample')
  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i }).first()
  await expect(loginButton, 'ログインボタンが見つからない').toBeVisible({ timeout: 15000 })
  await loginButton.click()
  // ログイン成功で router.replace("/" + default_page) される
  await page.waitForURL((url) => url.pathname !== '/', { timeout: 60000 })

  // rykv でサンプルデータの記録が実際に一覧へ出ること
  await page.goto(`${sampleUrl}/rykv`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await expect(page.locator('[data-gkill-view-ready="true"]').first(), '初期検索が完了しない')
    .toBeVisible({ timeout: 90000 })
  await expect(page.locator('.kyou_in_list').first(), 'サンプルデータの記録が一覧に出ない')
    .toBeVisible({ timeout: 30000 })
})
