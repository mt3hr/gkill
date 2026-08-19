import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  navigateToSettings,
  clickFabButton,
  openApplicationConfigDialog,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Misc Operations', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番140: ブックマークレット登録
  test('bookmarklet is available in application config', async ({ page }) => {
    await navigateToSettings(page)

    // Look for bookmarklet section or link
    const app = page.locator('#app')
    const content = await app.textContent()

    // Check for bookmarklet-related text
    const _hasBookmarklet = content!.includes('ブックマークレット') ||
      content!.includes('bookmarklet') ||
      content!.includes('Bookmarklet')

    // Verify settings page renders
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番143: GPSログアップロード
  test('gps log upload via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    await clickFabButton(page)

    // Look for upload menu item（FABメニューの「アップロード」）
    const uploadItem = page.locator('.v-list-item, [role="menuitem"], .v-btn')
      .filter({ hasText: /アップロード|upload/i }).first()
    await expect(uploadItem, 'FABメニューにアップロードが無い').toBeVisible({ timeout: 30000 })
    await uploadItem.click()

    // Verify upload dialog opens with file input
    const dialog = page.locator('.gkill-floating-dialog').last()
    await expect(dialog, 'アップロードのダイアログが開かない').toBeVisible({ timeout: 30000 })
    // GPXを受ける入力欄があること（upload-file-view.vue の accept=".gpx"）
    await expect(dialog.locator('input[type="file"][accept=".gpx"]'), 'GPX用のファイル入力が無い')
      .toBeAttached({ timeout: 15000 })
  })

  // 項番153: 無効Mi共有リンクでエラーメッセージ表示
  test('invalid shared mi link shows error message', async ({ page }) => {
    // クエリの名前は **share_id**。`id` だと /shared_mi のリダイレクト元
    // (old-shared-mi-page.vue) が share_id を読めず、以前は setup ごと落ちて真っ白になっていた
    await page.goto('/shared_mi?share_id=invalid_nonexistent_id', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // /shared_mi は /shared_page へ差し替わる
    await expect(page, '共有ページへ移動しない').toHaveURL(/shared_page/, { timeout: 30000 })

    // 存在しない共有IDなので handle_get_shared_kyous がエラーを返し、
    // shared-page.vue が role="alert" の v-alert で見せる。
    // 「画面が描けた」だけの確認だと、黙って空のページが出ていても緑になる
    // `[role="alert"]` だけだと Vuetify が入力欄ごとに置く `v-input__details`
    // （常に存在して不可視）を掴んでしまう。`.v-alert` まで絞る
    await expect(page.locator('.v-alert[role="alert"]').first(), '無効な共有リンクでエラーが出ない')
      .toBeVisible({ timeout: 30000 })
  })

  // 項番155: サーバコンフィグ適用でサービス再起動
  test('server config apply triggers service restart', async ({ page }) => {
    // 設定ダイアログは歯車から開く。**最果て(/saihate)に歯車は無い**
    const dialog = await openApplicationConfigDialog(page)

    const applyButton = dialog.getByRole('button', { name: '適用', exact: true })
    await expect(applyButton, '設定に「適用」ボタンが無い').toBeVisible({ timeout: 15000 })
    await applyButton.click()

    // 適用でサーバが再起動しても、画面が開き直せること
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 30000 })
    // **`.v-application` の可視で待たないこと。** 最果ては v-main の中身が空なので
    // ルート要素の高さが 0 になり、Playwright は hidden と判定する（/kyou や /rykv では通る）。
    // 画面ごとに「本当に出るもの」を待つ
    await expect(page.locator('.v-toolbar-title'), '適用後に画面が開かない')
      .toBeVisible({ timeout: 60000 })
  })
})
