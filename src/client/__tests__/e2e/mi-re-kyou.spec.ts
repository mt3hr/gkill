import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, clickContextMenuItem, clickDialogButton, createAndSelectMiBoard,
  confirmDelete, expectPageToContainText, waitForKyouByText,
  searchByKeyword, waitForKyouRowByRepName,
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

    // タスク化した記録がMi画面に出ること。
    // 一覧は仮想スクロールで数件しか描画しないので、絞り込まないと
    // 並列に走る他テストの記録に押し出されて見つからなくなる。
    // 確認はページ本文の有無で行う（同じ要約は「すべて」列と板の列の両方に出て、
    // 先に見つかるほうが描画対象外＝非表示のことがあるため、可視性は要求しない）
    await navigateToMi(page)
    await searchByKeyword(page, label)
    await expectPageToContainText(page, board)
    await expectPageToContainText(page, label)
  })

  /**
   * Mi画面の行は高さ固定 (mi-view.vue の kyou_height = 56 + 35) で
   * overflow:hidden なので、はみ出した分は切り落とされて見えなくなる。
   * MiReKyouは参照先Kyouを丸ごと埋め込んでいたため日時が枠外に出ていた。
   */
  test('Mi画面の行に参照先の要約と日時が収まる', async ({ page }) => {
    // 要約は長くしておく。v-checkbox は min-content 幅が 0 まで潰れるので、
    // 要約が長いほど flex の縮小量を持っていかれてチェックボックスが消えていた
    const label = makeUniqueLabel('mirekyou_row').concat('あ'.repeat(120))

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

    // チェックボックスが要約に押し潰されていないこと
    // 潰れるのは v-checkbox のルート。内側の .v-selection-control__wrapper は
    // width 固定で縮まず枠外へはみ出すので、そちらの幅を見ても検出できない
    const layout = await row.evaluate((el) => {
      const root = el.querySelector('.mirekyou_check') as HTMLElement
      const mark = el.querySelector('.mirekyou_check .v-selection-control__wrapper') as HTMLElement
      const summary = el.querySelector('.mirekyou_summary') as HTMLElement
      return {
        root_width: root.getBoundingClientRect().width,
        // 正の値ならチェックボックスの絵が要約の左端を追い越している＝重なっている
        overlap: mark.getBoundingClientRect().right - summary.getBoundingClientRect().left,
      }
    })
    expect(layout.root_width, `チェックボックスが潰れている (width=${layout.root_width})`)
      .toBeGreaterThanOrEqual(20)
    expect(layout.overlap, `チェックボックスが要約に重なっている (overlap=${layout.overlap})`)
      .toBeLessThanOrEqual(1)
  })

  /**
   * Kyouダイアログは行ではないので、参照先を丸ごと出す（詳細ペインと同じ）。
   * kyou-dialog.vue が高さに '80%' を渡していたせいで is_row_height が
   * 行と誤判定し、mi-re-kyou-view.vue の参照先ブロックが丸ごと消えていた。
   */
  test('Kyouダイアログを開くと参照先の内容が出る', async ({ page }) => {
    const label = makeUniqueLabel('mirekyou_dialog')

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addMiReKyou)

    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, 'タスク化ダイアログが開かない').toBeVisible({ timeout: 15000 })
    const board = makeUniqueLabel('mirekyou_dialog_board')
    await createAndSelectMiBoard(page, dialog, board)
    await clickDialogButton(page, SAVE_BUTTON)

    // タスク化した行は元の記録と同じ本文なので、リポジトリ名で特定する
    await navigateToRykv(page)
    await searchByKeyword(page, label)
    const taskRow = await waitForKyouRowByRepName(page, 'MiReKyou')

    // 参照先を詰める(compact)のは行高が KYOU_ROW_MAX_HEIGHT 未満のときだけなので、
    // rykv の一覧(180px)では行でも参照先が出る。詰まるのは Mi 画面の行(91px)で、
    // そちらは「Mi画面の行に参照先の要約と日時が収まる」が見ている。
    // kyou-view.vue のルート要素に @dblclick があるので、行のどこをダブルクリックしてもよい
    await taskRow.dblclick()

    const kyouDialog = page.locator('.kyou_dialog')
    await expect(kyouDialog, 'Kyouダイアログが開かない').toBeVisible({ timeout: 15000 })

    const target = kyouDialog.locator('.mirekyou_target')
    await expect(target, 'Kyouダイアログに参照先が出ていない').toBeVisible({ timeout: 15000 })
    // 参照先は元の記録（Kmemo）そのもの
    await expect(target.locator('.kyou_rep_name').filter({ hasText: /^\s*Kmemo\s*$/ }), '参照先が元の記録になっていない')
      .toHaveCount(1, { timeout: 15000 })
  })

  /**
   * 連鎖削除 (cascade-delete-kyou.ts) を入れたとき、update系APIが成功時に
   * errors: null を返すことを見落として spread で TypeError を投げており、
   * 確認ダイアログのクローズまで到達していなかった。
   * サーバ側の削除は成功しているので、再読み込みしてから消えたことを見るだけでは
   * 検出できない。閉じるところまでを confirmDelete が見る。
   */
  test('タスク化した記録を削除すると確認ダイアログが閉じて一覧から消える', async ({ page }) => {
    const label = makeUniqueLabel('mirekyou_delete')

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addMiReKyou)

    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, 'タスク化ダイアログが開かない').toBeVisible({ timeout: 15000 })
    const board = makeUniqueLabel('mirekyou_del_board')
    await createAndSelectMiBoard(page, dialog, board)
    await clickDialogButton(page, SAVE_BUTTON)

    // タスク化した行は元の記録と同じ本文で表示されるので、リポジトリ名で特定する。
    // Mi画面ではなく rykv で操作するのは、新しく作った板が選択中とは限らず、
    // 行がDOMにはあっても非表示のことがあるため（削除リポストのテストと同じ方針）。
    await navigateToRykv(page)
    await searchByKeyword(page, label)
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*MiReKyou\s*$/ }), 'タスク化した行が作られていない')
      .toHaveCount(1, { timeout: 30000 })

    const taskRow = await waitForKyouRowByRepName(page, 'MiReKyou')
    await taskRow.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.delete)
    // confirmDelete が確認ダイアログの自動クローズまで検証する
    await confirmDelete(page)

    await navigateToRykv(page)
    await searchByKeyword(page, label)
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*MiReKyou\s*$/ }), 'タスク化した行が消えていない')
      .toHaveCount(0, { timeout: 30000 })
    // タスク化した側だけが消え、元の記録は残る
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*Kmemo\s*$/ }), '元の記録まで消えている')
      .toHaveCount(1, { timeout: 30000 })
    await expectPageToContainText(page, label)
  })
})
