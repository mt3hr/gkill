import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  navigateToMi, makeUniqueLabel, searchByKeyword,
  clickSidebarSearchButton, clickFabButton, clickContextMenuItem,
  clickDialogButton, createAndSelectMiBoard, dismissFloatingDialogs,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * mi の板列×検索。
 *
 * 「検索時の列(板)に検索時の結果が表示される」「列の板名表示が変わらない」を固定する。
 * 以前はサイドバーの板選択が全列共有のスティッキー変数で、板ツリーを
 * クリックした後に別の列で検索すると、最後にクリックした板名が検索条件に
 * 混入して列の板名表示ごと変わってしまうことがあった。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const DIALOG = '.gkill-floating-dialog, .v-dialog'

/**
 * mi画面の追加ダイアログから、新しい板のタスクを作る。
 *
 * KFTL等で作った直後の板は、サーバの板一覧(get_mi_board_list)に反映されるまで
 * 板ツリーに出ない。このテストは「新しい板です」確認の確定でクライアント側の
 * 板ツリーへ即時追加される経路(remember_confirmed_mi_boards →
 * append_mi_board_to_struct)を使う。ページをリロードすると即時追加分は
 * サーバ一覧からの再構築で消えるので、テストはリロードせず1セッションで完結させる。
 */
async function addMiTaskWithNewBoard(page: Page, title: string, boardName: string): Promise<void> {
  await clickFabButton(page)
  await clickContextMenuItem(page, MENU.addMi)
  const dialog = page.locator(DIALOG).first()
  await expect(dialog, 'タスク追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

  const titleField = dialog.locator('input[type="text"], .v-text-field input').first()
  await expect(titleField, 'タイトル入力欄が無い').toBeVisible({ timeout: 15000 })
  await titleField.fill(title)
  await expect(titleField).toHaveValue(title)

  await createAndSelectMiBoard(page, dialog, boardName)

  // 保存すると「新しい板です」確認が出る。確定で保存され、板ツリーにも即時追加される
  await dialog.locator('button').filter({ hasText: SAVE_BUTTON }).first().click()
  const confirmDialog = page.locator(DIALOG).filter({ hasText: '新しい板です' }).first()
  await expect(confirmDialog, '「新しい板です」確認ダイアログが開かない').toBeVisible({ timeout: 15000 })
  await clickDialogButton(page, SAVE_BUTTON)

  // 開いたままの追加ダイアログが後続のクリックを奪わないよう閉じておく
  await dismissFloatingDialogs(page)
}

/** サイドバーの板ツリーから板をクリックして列を開く/フォーカスする */
async function openBoardFromSidebar(page: Page, boardName: string): Promise<void> {
  const drawer = page.locator('.v-navigation-drawer').first()
  await expect(drawer, 'サイドバーが表示されていない').toBeVisible({ timeout: 15000 })
  const boardItem = drawer.getByText(boardName, { exact: true }).first()
  await expect(boardItem, `板ツリーに ${boardName} が見つからない`).toBeVisible({ timeout: 30000 })
  await boardItem.click()
}

/**
 * mi画面の列(td)を左からの位置で掴む。
 * タスク行にも板名テキスト(.v-card-title)が出るため、板名での絞り込みは
 * 全タスクを表示する「すべて」列に誤マッチする。列は開いた順に末尾へ
 * 追加される(すべて=0、以降は板を開いた順)ので位置で特定し、
 * 列タイトル(tdの先頭の.v-card-title)は別途検証する
 */
function columnAt(page: Page, index: number) {
  // 行の描画にも入れ子のtableがあるため、列のtd(直下)だけに限定する
  return page.locator('.mi_view_table > tbody > tr > td').nth(index)
}

function columnTitle(page: Page, index: number) {
  return columnAt(page, index).locator('.v-card-title').first()
}

test.describe('mi board columns', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(300000)
    await loginAsAdmin(page)
  })

  test('板の列には自分の板のタスクだけが表示され、検索しても板名が汚染されない', async ({ page }) => {
    const boardA = makeUniqueLabel('boardA')
    const boardB = makeUniqueLabel('boardB')
    const taskA = makeUniqueLabel('task_a')
    const taskB = makeUniqueLabel('task_b')

    await navigateToMi(page)
    await addMiTaskWithNewBoard(page, taskA, boardA)
    await addMiTaskWithNewBoard(page, taskB, boardB)

    // 板ツリーから両方の板を列として開き、それぞれ検索する
    // (rykv_hot_reload 既定OFFでは列を開いただけでは検索されない)
    await openBoardFromSidebar(page, boardA)
    await clickSidebarSearchButton(page)
    await openBoardFromSidebar(page, boardB)
    await clickSidebarSearchButton(page)

    // 列の並び: すべて(0) / 板A(1) / 板B(2)
    await expect(columnTitle(page, 1), '板Aの列が開かない').toHaveText(new RegExp(boardA), { timeout: 30000 })
    await expect(columnTitle(page, 2), '板Bの列が開かない').toHaveText(new RegExp(boardB), { timeout: 30000 })

    // 各列に自分の板のタスクだけが出る
    await expect(columnAt(page, 1), '板Aの列に板Aのタスクが出ない').toContainText(taskA, { timeout: 30000 })
    await expect(columnAt(page, 1), '板Aの列に別の板のタスクが混ざった').not.toContainText(taskB, { timeout: 30000 })
    await expect(columnAt(page, 2), '板Bの列に板Bのタスクが出ない').toContainText(taskB, { timeout: 30000 })
    await expect(columnAt(page, 2), '板Bの列に別の板のタスクが混ざった').not.toContainText(taskA, { timeout: 30000 })

    // 「すべて」列にフォーカスを戻してキーワード検索する。
    // 以前はツリーで最後にクリックした板(板B)がサイドバーに残っていて、
    // 「すべて」列が板Bの条件+板名表示に化けていた
    await columnAt(page, 0).locator('.kyou_list_view').first().click()
    await searchByKeyword(page, taskA)
    await clickSidebarSearchButton(page)

    // 「すべて」列の板名表示は「すべて」のまま、絞り込みだけが効く
    await expect(columnTitle(page, 0), '「すべて」列の板名表示が変わった').toHaveText(/すべて/)
    await expect(columnAt(page, 0), '「すべて」列に検索結果が出ない').toContainText(taskA, { timeout: 30000 })
    await expect(columnAt(page, 0), '「すべて」列に絞り込みが効いていない').not.toContainText(taskB, { timeout: 30000 })

    // 板A・板Bの列の板名と内容は元のまま
    await expect(columnTitle(page, 1), '板Aの列の板名表示が変わった').toHaveText(new RegExp(boardA))
    await expect(columnTitle(page, 2), '板Bの列の板名表示が変わった').toHaveText(new RegExp(boardB))
    await expect(columnAt(page, 1), '板Aの列の内容が変わった').toContainText(taskA, { timeout: 30000 })
    await expect(columnAt(page, 2), '板Bの列の内容が変わった').toContainText(taskB, { timeout: 30000 })
  })
})
