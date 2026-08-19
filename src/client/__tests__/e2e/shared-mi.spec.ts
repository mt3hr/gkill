import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Shared Mi Page', () => {
  test('shared mi page loads without crashing', async ({ page }) => {
    // Navigate without a share ID parameter; page should still load without crashing
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
  })

  test('shared mi page renders app container', async ({ page }) => {
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('shared mi page does not show fatal error', async ({ page }) => {
    await page.goto('/shared_mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // share_id を付けずに開いたので「共有情報が見つからない」が出る。
    // 以前は old-shared-mi-page.vue が `query.share_id!.toString()` で setup ごと落ち、
    // **エラーも出ない真っ白な画面**になっていた（今回サーバ/クライアント側を修正済み）。
    // `[role="alert"]` だけだと Vuetify が入力欄ごとに置く `v-input__details`（常に存在・不可視）を
    // 掴むので、必ず `.v-alert` まで絞ること
    await expect(page, '共有ページへ移動しない').toHaveURL(/shared_page/, { timeout: 30000 })
    await expect(page.locator('.v-alert[role="alert"]').first(), '共有情報が無いのにエラーが出ない')
      .toBeVisible({ timeout: 30000 })
  })
})
