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

  /**
   * Mi画面の行は高さ固定 (mi-view.vue の kyou_height = 56 + 35) で
   * overflow:hidden なので、はみ出した分は切り落とされて見えなくなる。
   * MiReKyouは参照先Kyouを丸ごと埋め込んでいたため日時が枠外に出ていた。
   */
  test('Mi画面の行に参照先の要約と日時が収まる', async ({ page }) => {
    const label = makeUniqueLabel('mirekyou_row')

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addMiReKyou)

    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, 'タスク化ダイアログが開かない').toBeVisible({ timeout: 15000 })

    const board = makeUniqueLabel('mirekyou_row_board')
    await createAndSelectMiBoard(page, dialog, board)

    // 日時が行に出るかを見たいので開始日時を入れる。
    // 日付ピッカーを操作せずに済むよう「現在日時」ボタンを使う（開始・終了・制限の順に並ぶ）。
    await dialog.locator('button').filter({ hasText: /^\s*現在日時\s*$/ }).first().click()

    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToMi(page)
    await expectPageToContainText(page, label)

    const row = page.locator('.kyou_in_list').filter({ hasText: label }).first()
    await expect(row, 'Mi画面にタスク化した行が出ない').toBeVisible({ timeout: 30000 })

    // はみ出していないこと。これが破綻の直接的な指標
    const size = await row.evaluate((el) => ({
      scroll: el.scrollHeight,
      client: el.clientHeight,
      // clientHeight は border-top:1px を含まないので、行高の確認には offsetHeight を使う
      offset: (el as HTMLElement).offsetHeight,
    }))
    expect(size.offset, '行の高さが mi-view.vue の指定 (56 + 35) と違う').toBe(91)
    expect(size.scroll, `行からはみ出している (scrollHeight=${size.scroll}, clientHeight=${size.client})`)
      .toBeLessThanOrEqual(size.client + 1)

    // 参照先の要約・板名・日時が行の中にあること
    await expect(row).toContainText(label)
    await expect(row).toContainText(board)
    await expect(row, '日時が行に出ていない').toContainText('開始日時')

    // 外側と内側のKyouViewで時刻が二重に出ていないこと
    expect(await row.locator('.kyou_related_time').count(), '行に時刻が2つ以上出ている')
      .toBeLessThanOrEqual(1)
  })
})
