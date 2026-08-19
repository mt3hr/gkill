import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

/**
 * 共有ページ（認証なしで開ける公開ページ）。
 *
 * 共有IDに対して何が返るかは Go 側の handle_get_shared_kyous_test.go で
 * 網羅している（共有対象だけが返ること・取り消し後は取れないこと等）。
 * ここでは「不正な共有IDでもページが壊れず、エラーとして利用者に伝わる」という
 * クライアント側の振る舞いだけを見る。
 *
 * 以前は `#app` が存在することだけを確認しており、
 * 共有ページが真っ白でも読み込み中のままでも成功していた。
 */

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Share Page', () => {
  test('存在しない共有IDではエラーが表示され、読み込み中のままにならない', async ({ page }) => {
    await page.goto('/shared_page?share_id=e2e-nonexistent-share-id', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    // 読み込み中オーバーレイが出たままにならないこと
    const spinner = page.locator('.v-overlay .v-progress-circular').first()
    await expect(spinner, '読み込み中のまま止まっている').toBeHidden({ timeout: 30000 })

    // エラーが利用者に伝わること（shared-page.vue の v-alert は role="alert"）
    await expect(page.locator('.v-alert[role="alert"]').first(), 'エラーが表示されない').toBeVisible({ timeout: 30000 })
  })
})
