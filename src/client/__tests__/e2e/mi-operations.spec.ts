import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToMi, createAndSelectMiBoard,
  makeUniqueLabel, clickContextMenuItem, clickDialogButton,
  expectPageToContainText, waitForKyouByText, searchByKeyword,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * Mi（タスク）の操作。
 *
 * 以前はレコード・メニュー項目・ダイアログのそれぞれを
 * `if (await x.count() > 0)` で包んでいたため、UIが変わって掴めなくなっても
 * テストは緑のままだった。ここでは各入口が存在することを前提にする。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const DIALOG = '.gkill-floating-dialog, .v-dialog'

/**
 * 設定画面で「共有フッターを表示」を有効にする。
 *
 * `application_config.is_show_share_footer` は既定 false で、
 * off のままだと mi-query-editor-sidebar.vue の ShareKyouFooter が描画されない。
 */
async function enableShareFooter(page: Page): Promise<void> {
  // アプリ設定は画面上部の歯車ボタン（mdi-cog）が開くダイアログにある
  // （mi-page.vue → application-config-dialog.vue → application-config-view.vue）
  await navigateToMi(page)

  const configButton = page.locator('button:has(.mdi-cog)').first()
  await expect(configButton, '設定（歯車）ボタンが見つからない').toBeVisible({ timeout: 30000 })
  await configButton.click()

  const checkbox = page.locator('.v-checkbox').filter({ hasText: '共有フッターを表示' }).first()
  await expect(checkbox, '「共有フッターを表示」の設定が見つからない').toBeVisible({ timeout: 30000 })

  // Vuetify の input は視覚的に隠れているので、ラッパーをクリックする。
  // ラッパー全体をクリックするとラベル部分に当たって反応しないことがあるため、
  // .v-selection-control を狙う。
  const input = checkbox.locator('input[type="checkbox"]')
  if (!(await input.isChecked())) {
    await checkbox.locator('.v-selection-control__input').first().click()
    await expect(input, '共有フッターの設定がONにならない').toBeChecked({ timeout: 15000 })
  }

  // 「適用」で保存する（use-application-config-view.ts の update_application_config）
  const applyButton = page.locator('button').filter({ hasText: /^\s*適用\s*$/ }).first()
  await expect(applyButton, '設定の「適用」ボタンが見つからない').toBeVisible({ timeout: 15000 })
  const saved = page.waitForResponse((res) => res.url().includes('/api/update_application_config'), { timeout: 30000 })
  await applyButton.click()
  await saved
}

test.describe('Mi (Task) Operations', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番73: タスク板間移動
  //
  // 板名は `<v-select :items="mi_board_names">` で自由入力できない。
  // 新しい板は隣の ＋ ボタンが開く「板名追加」ダイアログで作る
  // （edit-mi-view.vue → new-board-name-dialog.vue）。
  test('タスクの板を編集ダイアログから変更するとMi画面に新しい板が出る', async ({ page }) => {
    const label = makeUniqueLabel('mi_board_move')
    const newBoard = makeUniqueLabel('board')

    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.edit)

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, '編集ダイアログが開かない').toBeVisible({ timeout: 15000 })

    await createAndSelectMiBoard(page, dialog, newBoard)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToMi(page)
    await expectPageToContainText(page, newBoard)
    await expectPageToContainText(page, label)
  })

  // 項番74: タスク完了状態編集
  test('タスクのチェックボックスで完了状態を切り替えられる', async ({ page }) => {
    const label = makeUniqueLabel('mi_complete')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    await waitForKyouByText(page, label)

    // 対象タスクのカードを掴む。mi-kyou-view.vue のルートが v-card で、
    // その中にチェックボックスとタイトルがある。
    // .first() だと外側のコンテナ v-card に当たって別のタスクの
    // チェックボックスを触ってしまうので、最も内側の .last() を使う。
    const card = page.locator('.v-card').filter({ hasText: label }).last()
    await expect(card, 'タスクのカードが見つからない').toBeVisible({ timeout: 30000 })

    const checkbox = card.locator('input[type="checkbox"]').first()
    await expect(checkbox, 'タスクのチェックボックスが見つからない').toBeAttached({ timeout: 15000 })
    await expect(checkbox).not.toBeChecked()

    // Vuetify のチェックボックスは input が視覚的に隠れているので、
    // ラッパー（.v-selection-control）をクリックする。
    // チェックはサーバへ update_mi を送るので、その応答を待ってから状態を見る。
    const response = page.waitForResponse((res) => res.url().includes('/api/update_mi'), { timeout: 30000 })
    await card.locator('.v-selection-control, .v-checkbox').first().click()
    await response

    // 完了にすると既定の検索条件（未チェック）から外れて一覧から消える
    // （use-mi-extract-check-state-query.ts の既定は MiCheckState.uncheck）
    await navigateToMi(page)
    await expect(page.locator('#app'), 'チェックしても未チェック一覧から消えない')
      .not.toContainText(label, { timeout: 30000 })
  })

  /**
   * v-checkbox のルート（.v-input--horizontal）は grid の中身が minmax(0,1fr) で、
   * min-content 幅が 0 まで潰れる。タイトルが長いほど flex の縮小量を持っていかれ、
   * チェックボックスが幅0になってタイトルの下に隠れていた。
   * 幅は toBeAttached では見えないので、実寸で確認する。
   */
  test('タイトルが長くてもチェックボックスが潰れない', async ({ page }) => {
    const label = makeUniqueLabel('mi_long_title').concat('あ'.repeat(120))
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    await waitForKyouByText(page, label)
    const card = page.locator('.v-card').filter({ hasText: label }).last()
    await expect(card, 'タスクのカードが見つからない').toBeVisible({ timeout: 30000 })
    await expect(card.locator('.mi_check'), 'チェックボックスが見つからない')
      .toBeVisible({ timeout: 15000 })

    // 潰れるのは v-checkbox のルート。内側の .v-selection-control__wrapper は
    // width 固定で縮まず枠外へはみ出すので、そちらの幅を見ても検出できない
    const layout = await card.evaluate((el) => {
      const root = el.querySelector('.mi_check') as HTMLElement
      const mark = el.querySelector('.mi_check .v-selection-control__wrapper') as HTMLElement
      const title = el.querySelector('.mi_title') as HTMLElement
      return {
        root_width: root.getBoundingClientRect().width,
        // 正の値ならチェックボックスの絵がタイトルの左端を追い越している＝重なっている
        overlap: mark.getBoundingClientRect().right - title.getBoundingClientRect().left,
      }
    })

    expect(layout.root_width, `チェックボックスが潰れている (width=${layout.root_width})`)
      .toBeGreaterThanOrEqual(24)
    expect(layout.overlap, `チェックボックスがタイトルに重なっている (overlap=${layout.overlap})`)
      .toBeLessThanOrEqual(1)
  })

  // 項番76 / 項番77: タスク共有状況の閲覧と共有停止
  //
  // 共有はコンテキストメニューではなく、Mi画面サイドバーのフッタにある
  // 「共有」「共有管理」ボタンから行う（mi-query-editor-sidebar.vue →
  // share-kyou-footer.vue → manage-share-button.vue）。
  //
  // 以前ここには「レコードを右クリック → 共有」を試すテストが2本あったが、
  // Miのコンテキストメニューに共有の項目は存在しない
  // （mi-context-menu.vue にあるのは タグ追加/テキスト追加/リポスト/
  // タスクにする/通知追加/編集/履歴/内容コピー/IDコピー/フォルダを開く/
  // ファイルを開く/削除 の12項目）。
  // `if (await shareItem.count() > 0)` で包まれていたため、
  // 入口が無いことに気づかないまま成功し続けていた。
  test('Mi画面のサイドバーから共有管理ダイアログを開ける', async ({ page }) => {
    const label = makeUniqueLabel('mi_share_view')
    await submitKftlText(page, `ーみ\n${label}`)

    // 共有フッタは既定で非表示（application-config.ts の is_show_share_footer = false）。
    // 設定画面で有効にしてからでないとサイドバーにボタンが出ない。
    await enableShareFooter(page)

    await navigateToMi(page)
    // 一覧は仮想スクロールで数件しか描画しないので、並列に走る他テストの記録に
    // 押し出される。キーワードで絞ってから存在を確認する
    await searchByKeyword(page, label)
    await waitForKyouByText(page, label)

    const manageShareButton = page.locator('button').filter({ hasText: /^\s*共有管理\s*$/ }).first()
    await expect(manageShareButton, 'サイドバーに共有管理ボタンが無い').toBeVisible({ timeout: 30000 })
    await manageShareButton.click()

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, '共有管理ダイアログが開かない').toBeVisible({ timeout: 15000 })

    // ダイアログがビューポートに対して極端に縦長でないこと
    // （項番76の「スクロールしすぎない」確認）
    const dialogBox = await dialog.boundingBox()
    const viewport = page.viewportSize()
    expect(dialogBox, 'ダイアログの大きさが取れない').not.toBeNull()
    expect(viewport, 'ビューポートサイズが取れない').not.toBeNull()
    expect(dialogBox!.height).toBeLessThan(viewport!.height * 1.5)
  })
})
