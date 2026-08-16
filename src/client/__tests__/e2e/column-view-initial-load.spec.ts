import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import { waitForColumnViewReady } from './crud-helpers'

/**
 * 初期検索の完了を待たずに画面を見せることの検証。
 *
 * 以前は「保存済み列の初期検索が全部終わるまで画面全体を隠す」実装で、
 * サイドバーもテーブルも v-show で消え、全画面オーバーレイが被さっていた。
 * 検索が1本でも解決しないだけで画面全体が固まるうえ、初期化中は
 * サイドバーの編集を丸ごと捨てていた。
 *
 * get_kyous の応答を遅らせて「初期検索の飛行中」を決定論的に作る。
 * 固定の待ち時間は使わない（no-wait-for-timeout に触れない）。
 */

const SEARCH_DELAY_MS = 4000

let apiReachable = false
test.beforeAll(async () => {
  apiReachable = await checkGkillServer()
  test.skip(!apiReachable, 'gkill server is not running')
})

/** goto の前に仕込む。初期検索を人工的に伸ばして「飛行中」を観測できるようにする */
async function delayGetKyous(page: Page): Promise<void> {
  await page.route('**/api/get_kyous', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, SEARCH_DELAY_MS))
    await route.continue()
  })
}

test.describe('列ビューの初期表示', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('rykv: 初期検索の飛行中でもサイドバーと列が見えて操作できる', async ({ page }) => {
    await delayGetKyous(page)
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // 全画面オーバーレイのスピナーは初期検索を待たずに消える
    const root = page.locator('.rykv_view_wrap')
    await expect(root, 'ビューのルートが出ない').toBeVisible({ timeout: 30000 })

    // まだ初期検索は終わっていない
    await expect(root, '初期検索が終わる前に準備完了になっている')
      .toHaveAttribute('data-gkill-view-ready', 'false', { timeout: 5000 })

    // その状態でサイドバーと列が見えていて、ハンバーガーも押せる
    await expect(page.locator('.rykv_query_editor_sidebar'), '初期検索中にサイドバーが隠れている')
      .toBeVisible({ timeout: 15000 })
    await expect(page.locator('.rykv_view_table'), '初期検索中に列のテーブルが隠れている')
      .toBeVisible({ timeout: 15000 })
    await expect(page.locator('.v-app-bar button').first(), '初期検索中にハンバーガーが押せない')
      .toBeEnabled({ timeout: 15000 })

    // 遅延が明けたら準備完了になる
    await waitForColumnViewReady(page)
    await expect(root).toHaveAttribute('data-gkill-view-ready', 'true')
  })

  test('mi: 初期検索の飛行中でもサイドバーと列が見えて操作できる', async ({ page }) => {
    await delayGetKyous(page)
    await page.goto('/mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    const root = page.locator('.mi_view_wrap')
    await expect(root, 'ビューのルートが出ない').toBeVisible({ timeout: 30000 })

    await expect(root, '初期検索が終わる前に準備完了になっている')
      .toHaveAttribute('data-gkill-view-ready', 'false', { timeout: 5000 })

    await expect(page.locator('.mi_query_editor_sidebar'), '初期検索中にサイドバーが隠れている')
      .toBeVisible({ timeout: 15000 })
    await expect(page.locator('.mi_view_table'), '初期検索中に列のテーブルが隠れている')
      .toBeVisible({ timeout: 15000 })

    await waitForColumnViewReady(page)
    await expect(root).toHaveAttribute('data-gkill-view-ready', 'true')
  })
})
