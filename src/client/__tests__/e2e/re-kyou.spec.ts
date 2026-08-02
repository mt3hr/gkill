import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel, clickContextMenuItem,
  clickDialogButton, waitForKyouByText, waitForKyouRowByRepName, searchByKeyword,
  MENU,
} from './crud-helpers'

/**
 * ReKyou（リポスト）。
 *
 * ReKyouは参照先Kyouを入れ子で描画するため、以前は行の本文を右クリックすると
 * **参照先（Kmemo等）のコンテキストメニュー**が開き、ReKyou自身の編集・削除には
 * どの経路からも到達できなかった（re-kyou-view.vue のルートハンドラが
 * コメントアウトされ、内側KyouViewのメニューが生きていたため）。
 * MiReKyouと同じく「行のメニュー＝その行自身のメニュー」に揃えた。
 *
 * これはUIの挙動変更なので単体テストでは捕まらない。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('ReKyou (リポスト)', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('リポストの行を右クリックするとリポスト自身のメニューが出る', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_menu')

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.rekyou)
    await clickDialogButton(page, MENU.rekyou)

    // 仮想スクロールで他テストの記録に押し出されるので、この本文で絞る
    await navigateToRykv(page)
    await searchByKeyword(page, label)
    const rekyouRow = await waitForKyouRowByRepName(page, 'ReKyou')

    // 行の「本文側」を右クリックする。
    // ここが参照先の入れ子部分で、以前はKmemoのメニューが開いていた。
    const rekyouCard = rekyouRow.locator('xpath=ancestor::*[contains(@class,"kyou_in_list")][1]')
    await rekyouCard.click({ button: 'right', force: true })

    // メニュー項目の顔ぶれはKmemoとReKyouで完全に同じなので、
    // 「編集」を押してどちらのダイアログが開くかで判定する
    await clickContextMenuItem(page, MENU.edit)

    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, '編集ダイアログが開かない').toBeVisible({ timeout: 15000 })
    await expect(dialog, 'リポストではなく参照先の編集ダイアログが開いている')
      .toContainText('リポスト編集', { timeout: 15000 })
  })
})
