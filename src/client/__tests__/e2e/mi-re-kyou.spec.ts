import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, clickContextMenuItem, clickDialogButton, createAndSelectMiBoard,
  expectPageToContainText, waitForKyouByText,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * MiReKyou（既存Kyouのタスク化）。
 *
 * 以前このファイルは Playwright から gkill の API を直接叩く
 * 統合テスト5本だった。API の振る舞い（追加・取得・更新・ターゲット解決・
 * ボード一覧）は gkill_server_api_test.go の TestHandleAddMiReKyou_* 系へ移し、
 * 他のデータ型と同じ層に揃えてある。そちらは npm run test_server で
 * e2e スタック無しに高速に回る。
 *
 * ここに残すのは e2e にしか確認できないもの、
 * すなわち「rykv のコンテキストメニューから実際にタスク化できる」導線。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('MiReKyou (既存Kyouのタスク化)', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('rykvのコンテキストメニューからタスク化するとMi画面に出る', async ({ page }) => {
    const label = makeUniqueLabel('mirekyou_target')

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    // ADD_MI_REKYOU_TITLE = 「タスクにする」
    await clickContextMenuItem(page, MENU.addMiReKyou)

    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, 'タスク化ダイアログが開かない').toBeVisible({ timeout: 15000 })

    // 板名を明示する。
    // use-add-mi-re-kyou-view.ts は既定の板名を reset() でしか設定しないため、
    // 開いた直後は空のことがあり、そのまま保存するとどの板にも並ばない。
    // 板名は v-select で自由入力できないので、＋ボタンから新しい板を作る。
    const board = makeUniqueLabel('mirekyou_board')
    await createAndSelectMiBoard(page, dialog, board)

    await clickDialogButton(page, SAVE_BUTTON)

    // タスク化した記録がMi画面に出ること
    await navigateToMi(page)
    await expectPageToContainText(page, board)
    await expectPageToContainText(page, label)
  })
})
