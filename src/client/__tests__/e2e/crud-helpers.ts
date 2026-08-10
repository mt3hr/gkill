import type { Locator, Page } from '@playwright/test'
import { expect } from '@playwright/test'

/** ダイアログのルート要素に共通するセレクタ。 */
const DIALOG_ROOT = '.gkill-floating-dialog, .v-dialog'

/** ダイアログのボタンに共通するセレクタ。 */
const DIALOG_BUTTON = '.gkill-floating-dialog button, .v-dialog button, .v-overlay__content .v-card button'

/**
 * コンテキストメニュー / FABメニューの項目に共通するセレクタ。
 *
 * `.v-list-item` だけで探すと rykv サイドバーの記録分類ツリー
 * （「気分」「支出」「タスク」など）に当たってしまう。FABメニューの項目名と
 * 同じ文字列が並んでいるため、`.first()` が常にサイドバー側を掴んでいた。
 * v-menu は `.v-menu > .v-overlay__content` へ teleport されるので、
 * その配下に限定することでメニューだけを対象にする。
 */
const CONTEXT_MENU_ITEM = '.v-menu .v-list-item, .v-menu .v-btn, [role="menuitem"]'

/**
 * 書き込み系APIのURL判定。
 *
 * ダイアログの保存/削除が「実際にサーバへ届いて応答が返った」ことを待つのに使う。
 * `/api/` を丸ごと対象にすると、画面表示のために飛んでいる get_kyous 等の
 * 応答を拾ってしまい、保存が完了する前に次の操作へ進んでリクエストが
 * 中断される（画面遷移でabortされる）。
 */
function isWriteApiResponse(url: string): boolean {
  return /\/api\/(add|update|delete|submit)_/.test(url)
}

/** ダイアログの保存ボタン。 */
export const SAVE_BUTTON = /^\s*保存\s*$/

/**
 * Generate a unique label for test data using timestamp.
 */
export function makeUniqueLabel(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

/**
 * TagStructに無いタグを付けようとすると、保存の前に確認ダイアログが出る
 * （CONFIRM_UNKNOWN_TAG_MESSAGE =「新しいタグです。追加しますか？」）。
 * KFTLの送信（use-kftl-view.ts）とタグ追加ダイアログ（use-add-tag-view.ts）の
 * 両方にあり、確定しないと保存リクエストが飛ばない。
 *
 * テストは毎回ユニークなタグ名を使うので必ずここを通る。出たら確定する。
 */
async function confirmUnknownTagIfShown(page: Page): Promise<void> {
  const dialog = page.locator(DIALOG_ROOT).filter({ hasText: '新しいタグです' }).first()
  await dialog.waitFor({ state: 'visible', timeout: 3000 }).catch(() => {
    // 既知のタグだけなら確認ダイアログは出ない
  })
  if (await dialog.count() > 0) {
    await dialog.locator('button').filter({ hasText: SAVE_BUTTON }).first().click()
  }
}

/**
 * 板ツリーに無い板名を指定して保存しようとすると、保存の前に確認ダイアログが出る
 * （CONFIRM_UNKNOWN_MI_BOARD_MESSAGE =「新しい板です。追加しますか？」）。
 * タグ版（confirmUnknownTagIfShown）と同じで、確定しないと保存リクエストが飛ばない。
 *
 * createAndSelectMiBoard は毎回新しい板を作るので、Mi / MiReKyou の保存は必ずここを通る。
 */
async function confirmUnknownMiBoardIfShown(page: Page): Promise<void> {
  const dialog = page.locator(DIALOG_ROOT).filter({ hasText: '新しい板です' }).first()
  await dialog.waitFor({ state: 'visible', timeout: 3000 }).catch(() => {
    // 既知の板だけなら確認ダイアログは出ない
  })
  if (await dialog.count() > 0) {
    await dialog.locator('button').filter({ hasText: SAVE_BUTTON }).first().click()
  }
}

/**
 * Submit KFTL text via the KFTL page.
 * Navigates to /kftl, fills textarea, and clicks save.
 *
 * 保存完了は固定sleepではなく実シグナルで待つ。
 * 成功時のみ clear() が走って textarea が空になり、エラー時は内容が残る
 * (use-kftl-view.ts の submit()/clear() を参照)。
 * エラーを期待する呼び出しでは expectSuccess: false を渡すこと。
 */
export async function submitKftlText(
  page: Page,
  text: string,
  options: { expectSuccess?: boolean } = {},
): Promise<void> {
  const { expectSuccess = true } = options
  await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  // Use id selector for the KFTL textarea
  const textarea = page.locator('#kftl_text_area')
  await expect(textarea).toBeVisible({ timeout: 90000 })
  // Wait for the save button to become enabled (application_config loaded)
  const saveButton = page.locator('button').filter({ hasText: /保存|送信|submit|save/i }).first()
  await expect(saveButton).toBeEnabled({ timeout: 30000 })
  await textarea.fill(text)
  // fill が反映されてから保存する（入力途中の値で送るとテスト対象がずれる）
  await expect(textarea).toHaveValue(text, { timeout: 15000 })
  await saveButton.click()

  await confirmUnknownTagIfShown(page)

  if (expectSuccess) {
    // 送信成功でテキストエリアがクリアされる
    await expect(textarea).toHaveValue('', { timeout: 30000 })
  } else {
    // 失敗を期待する場合は入力が残ったままであること。
    // 「クリアされないこと」は待っても確定しないので、
    // 保存ボタンが再度押せる状態に戻ったのを合図にする。
    await expect(saveButton).toBeEnabled({ timeout: 30000 })
    await expect(textarea).not.toHaveValue('')
  }
}

/**
 * 読み込み中オーバーレイ（v-overlay 内の v-progress-circular）が消えるまで待つ。
 *
 * 以前は goto のあとに固定で3秒待っていたが、
 *   - 読み込みが3秒で終わらないマシンではフレークする
 *   - 終わっていても必ず3秒待つので全体が遅くなる
 * ため、実際の読み込み状態を見るようにしている。
 * オーバーレイが一度も出ないこともあるので、出現は短く待つだけにして
 * 「消えていること」を本体の条件にしている。
 */
async function waitForLoadingOverlayToFinish(page: Page): Promise<void> {
  const overlay = page.locator('.v-overlay .v-progress-circular').first()
  await overlay.waitFor({ state: 'visible', timeout: 3000 }).catch(() => {
    // オーバーレイが出ないまま読み込みが終わった場合。そのまま次の待機へ進む
  })
  await expect(overlay).toBeHidden({ timeout: 60000 })
}

/** ページ遷移してアプリの読み込み完了を待つ共通処理。 */
async function navigateTo(page: Page, path: string): Promise<void> {
  await page.goto(path, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await waitForLoadingOverlayToFinish(page)
  await dismissFloatingDialogs(page)
}

/**
 * Navigate to RYKV page and wait for it to load.
 */
export async function navigateToRykv(page: Page): Promise<void> {
  await navigateTo(page, '/rykv')
}

/**
 * Navigate to Mi board page and wait for it to load.
 */
export async function navigateToMi(page: Page): Promise<void> {
  await navigateTo(page, '/mi')
}

/**
 * Navigate to Plaing (TimeIs) page and wait for it to load.
 */
export async function navigateToPlaing(page: Page): Promise<void> {
  await navigateTo(page, '/plaing')
}

/**
 * Navigate to Settings page and wait for it to load.
 */
export async function navigateToSettings(page: Page): Promise<void> {
  await navigateTo(page, '/saihate')
}

/**
 * Check if the page contains the given text anywhere in #app.
 * 一度読むだけでリトライしない。アサーションには使わず、
 * expectPageToContainText / waitForPageText を使うこと。
 */
export async function pageContainsText(page: Page, text: string): Promise<boolean> {
  const app = page.locator('#app')
  const content = await app.textContent()
  return content != null && content.includes(text)
}

/**
 * #app に text が現れるまで待って検証する (自動リトライ)。
 * 描画待ちの固定sleepに依存しないための土台。
 */
export async function expectPageToContainText(page: Page, text: string, timeout = 30000): Promise<void> {
  await expect(page.locator('#app')).toContainText(text, { timeout })
}

/**
 * #app から text が消えるまで待って検証する (削除確認用)。
 */
export async function expectPageNotToContainText(page: Page, text: string, timeout = 30000): Promise<void> {
  await expect(page.locator('#app')).not.toContainText(text, { timeout })
}

/**
 * text が現れれば true、timeoutまで待って現れなければ false。条件分岐用。
 */
export async function waitForPageText(page: Page, text: string, timeout = 15000): Promise<boolean> {
  try {
    await expect(page.locator('#app')).toContainText(text, { timeout })
    return true
  } catch {
    return false
  }
}

/**
 * コンテキストメニュー / FABメニューの項目ラベル。
 *
 * 部分一致にすると別の項目を掴んでしまう組み合わせがあるため、
 * 前後の空白だけ許した完全一致にしてある。
 * 例: 「タグ追加」を `/タグ.*追加/` で探すと、先に並んでいる
 * 「タグ履歴から追加」に当たってしまう。
 *
 * 値は `src/locales/ja.json` の対応するキーと揃えること。
 */
export const MENU = {
  // コンテキストメニュー
  addTag: /^\s*タグ追加\s*$/, // ADD_TAG_TITLE
  addText: /^\s*テキスト追加\s*$/, // ADD_TEXT_TITLE
  addNotification: /^\s*通知追加\s*$/, // ADD_NOTIFICATION_TITLE
  addMiReKyou: /^\s*タスクにする\s*$/, // ADD_MI_REKYOU_TITLE
  rekyou: /^\s*リポスト\s*$/, // REKYOU_TITLE（「リポスト編集」と区別する）
  edit: /^\s*編集\s*$/, // EDIT_TITLE
  delete: /^\s*削除\s*$/, // DELETE_TITLE
  histories: /^\s*履歴\s*$/, // KYOU_HISTORIES_TITLE

  // 記録に付いたタグ / テキスト / 通知のコンテキストメニューは
  // Kyou本体とは別のメニューで、項目名も「タグ編集」「テキスト削除」のように
  // 対象名が前に付く（attached-*-context-menu.vue）。
  editTag: /^\s*タグ編集\s*$/, // TAG_CONTEXTMENU_EDIT_TAG
  deleteTag: /^\s*タグ削除\s*$/, // TAG_CONTEXTMENU_DELETE
  editText: /^\s*テキスト編集\s*$/, // TEXT_CONTEXTMENU_EDIT_TEXT
  deleteText: /^\s*テキスト削除\s*$/, // TEXT_CONTEXTMENU_DELETE
  editNotification: /^\s*通知編集\s*$/, // NOTIFICATION_CONTEXTMENU_EDIT_NOTIFICATION
  deleteNotification: /^\s*通知削除\s*$/, // DELETE_NOTIFICATION_TITLE

  // FAB（＋ボタン）の追加メニュー。*_APP_NAME
  addKC: /^\s*数値記録\s*$/,
  addURLog: /^\s*ブックマーク\s*$/,
  addTimeIs: /^\s*打刻帳\s*$/, // 「打刻メモ帳」と区別する
  addMi: /^\s*タスク\s*$/,
  addNlog: /^\s*支出\s*$/,
  addLantana: /^\s*気分\s*$/,
} as const

/**
 * Right-click on an element matching the selector to open context menu.
 *
 * 固定時間のsleepではなく、メニュー項目が実際に表示されるまで待つ。
 * sleepだと遅いマシンで足りずにフレークし、速いマシンでは無駄に待つことになる。
 */
export async function openContextMenu(page: Page, selector: string): Promise<void> {
  await dismissFloatingDialogs(page)
  const element = page.locator(selector).first()
  await element.click({ button: 'right', force: true })
  await expect(page.locator(CONTEXT_MENU_ITEM).first()).toBeVisible({ timeout: 15000 })
}

/**
 * Click a context menu item by its text label.
 * 対象の項目が表示されるまで待ってからクリックする。
 */
export async function clickContextMenuItem(page: Page, label: RegExp | string): Promise<void> {
  const menuItem = page.locator(CONTEXT_MENU_ITEM).filter({ hasText: label }).first()
  await expect(menuItem).toBeVisible({ timeout: 15000 })
  await menuItem.click()
}

/**
 * Click a button in a dialog (e.g., save or delete confirm).
 *
 * クリックしただけで次へ進むと、保存リクエストが飛ぶ前に画面遷移して
 * 中断されることがある。かといってダイアログが閉じるのを待つのも正しくない
 * （タグ追加のように、保存しても開いたままのダイアログがある）。
 * そこで「書き込みAPIの応答が返ってきたこと」を完了の合図にする。
 */
export async function clickDialogButton(page: Page, label: RegExp | string): Promise<void> {
  // ダイアログは重なって開くことがある。閉じたはずの前のダイアログを掴まないよう、
  // 表示中のもののうち最前面（DOM上で最後）を対象にする。
  const button = page.locator(DIALOG_BUTTON).filter({ hasText: label }).filter({ visible: true }).last()
  await expect(button).toBeVisible({ timeout: 15000 })

  const responsePromise = page.waitForResponse((res) => isWriteApiResponse(res.url()), { timeout: 30000 })
  await button.click()
  // 新しいタグ・新しい板を伴う保存では、確定しないとリクエストが飛ばない
  await confirmUnknownTagIfShown(page)
  await confirmUnknownMiBoardIfShown(page)
  const response = await responsePromise

  // gkillは失敗も HTTP 200 + errors配列 で返すので、中身まで見る。
  // ここを見ないと「保存できていないのに次のアサーションまで進む」ことになる。
  const body = await response.json().catch(() => null)
  const errors = (body as { errors?: unknown[] } | null)?.errors ?? []
  expect(errors, `${response.url()} がエラーを返した`).toHaveLength(0)
}

/**
 * Confirm a delete dialog by clicking the delete/confirm button.
 *
 * 削除を確定したら確認ダイアログは自動で閉じる。押しただけで先へ進むと、
 * 「サーバ側は消えているのに画面が閉じない」不具合を素通ししてしまう
 * （このあと再読み込みしてから消えたことを見るテストは、閉じなくても通ってしまう）。
 * 重なって開いている他のダイアログを巻き込まないよう、枚数が1枚減ることで見る。
 */
export async function confirmDelete(page: Page): Promise<void> {
  const dialogs = page.locator('.gkill-floating-dialog')
  const countBeforeDelete = await dialogs.count()
  expect(countBeforeDelete, '削除確認ダイアログが開いていない').toBeGreaterThan(0)

  await clickDialogButton(page, /削除|delete/i)

  await expect(dialogs, '削除後に確認ダイアログが自動で閉じない')
    .toHaveCount(countBeforeDelete - 1, { timeout: 15000 })
}

/**
 * rykv のサイドバーでキーワード検索を掛ける。
 *
 * 一覧は仮想スクロールで数件しか描画しない。E2Eは1つのサーバを共有して
 * 並列に走るため、他のテストが作った記録に押し出されて、
 * 少し前に作った記録が描画範囲から外れることがある。
 * 「作った直後に見えること」以外を確認するテストは、この検索で絞ってから見る。
 *
 * キーワード欄は「キーワード」チェックボックスがONのときだけ表示され
 * (keyword-query.vue の v-show)、@change で検索が走る。
 */
export async function searchByKeyword(page: Page, keyword: string): Promise<void> {
  const drawer = page.locator('.v-navigation-drawer').first()
  if (!(await drawer.isVisible())) {
    // モバイル幅などで畳まれている場合はハンバーガーで開く
    await page.locator('.v-app-bar button, .v-toolbar button').first().click()
    await expect(drawer, 'サイドバーが開かない').toBeVisible({ timeout: 15000 })
  }

  // 入力欄は「キーワード」チェックボックスがONのときだけ表示される
  // （keyword-query.vue の v-show="cloned_find_query.use_words"）
  const useKeyword = drawer.locator('.v-checkbox').filter({ hasText: /^\s*キーワード\s*$/ }).first()
  await expect(useKeyword, 'キーワード検索のチェックボックスが見つからない').toBeVisible({ timeout: 30000 })
  const useKeywordInput = useKeyword.locator('input[type="checkbox"]')
  if (!(await useKeywordInput.isChecked())) {
    await useKeyword.locator('.v-selection-control__input').first().click()
    await expect(useKeywordInput, 'キーワード検索が有効にならない').toBeChecked({ timeout: 15000 })
  }

  const keywordField = drawer.locator('input[type="text"]').first()
  await expect(keywordField, 'キーワードの入力欄が見つからない').toBeVisible({ timeout: 15000 })

  await keywordField.fill(keyword)
  // @change は blur / Enter で発火する
  await keywordField.press('Enter')
  await expect(keywordField).toHaveValue(keyword)

  await waitForLoadingOverlayToFinish(page)
}

/**
 * サイドバーの「検索」ボタンを押して、get_kyous の応答が返るまで待つ。
 *
 * `rykv_hot_reload` は既定 false のため、サイドバーで条件を編集しただけでは
 * 検索は走らない（searchByKeyword の Enter は hot reload ON のときだけ効く）。
 * フォーカス中の列に条件を確実に適用するテストは、編集後にこれを呼ぶこと。
 */
export async function clickSidebarSearchButton(page: Page): Promise<void> {
  const drawer = page.locator('.v-navigation-drawer').first()
  const searchButton = drawer.locator('button').filter({ hasText: /^\s*検索\s*$/ }).first()
  await expect(searchButton, 'サイドバーの検索ボタンが見つからない').toBeVisible({ timeout: 15000 })
  const responsePromise = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
  await searchButton.click()
  await responsePromise
}

/**
 * リポジトリ名（Kmemo / ReKyou / Mi など）で一覧の行を特定し、
 * コンテキストメニューを開くのに使える要素を返す。
 *
 * リポストは元の記録と同じ本文で表示されるので、本文だけでは区別できない。
 * さらにリポストは元の記録を**入れ子で**描画する（re-kyou-view.vue が
 * `<KyouView :show_rep_name="true">` で元Kyouを中に出す）ため、
 * 行そのものを掴んで `click()` すると中心が入れ子側に当たり、
 * 元の記録のコンテキストメニューが開いてしまう。
 * （この形でリポストを消したつもりが元記録を消していた。）
 *
 * `@contextmenu` は kyou-view.vue の v-row にあり `.kyou_rep_name` はその子なので、
 * リポジトリ名の要素を直接右クリックすれば、必ずその行自身のメニューが開く。
 *
 * repName は完全一致で渡すこと（"ReKyou" は "MiReKyou" の部分文字列）。
 */
export async function waitForKyouRowByRepName(page: Page, repName: string, timeout = 30000) {
  const repNameCell = page.locator('.kyou_rep_name')
    .filter({ hasText: new RegExp(`^\\s*${repName}\\s*$`) })
    .first()
  await expect(repNameCell, `${repName} の行が見つからない`).toBeVisible({ timeout })
  return repNameCell
}

/**
 * Mi の追加/編集ダイアログで、新しい板名を作って選択する。
 *
 * 板名は `<v-select :items="mi_board_names">` で自由入力できない
 * （add-mi-view.vue / edit-mi-view.vue）。新しい板は隣の mdi-plus ボタンが開く
 * 「板名追加」ダイアログ（new-board-name-dialog.vue）で作る。
 */
export async function createAndSelectMiBoard(page: Page, dialog: Locator, boardName: string): Promise<void> {
  const addBoardButton = dialog.locator('button:has(.mdi-plus)').first()
  await expect(addBoardButton, '板名追加ボタン(＋)が見つからない').toBeVisible({ timeout: 15000 })
  await addBoardButton.click()

  const boardDialog = page.locator(DIALOG_ROOT).filter({ hasText: '板名追加' }).first()
  await expect(boardDialog, '板名追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

  const boardField = boardDialog.locator('input[type="text"]').first()
  await expect(boardField, '板名の入力欄が無い').toBeVisible({ timeout: 15000 })
  await boardField.fill(boardName)
  await expect(boardField).toHaveValue(boardName)

  await boardDialog.locator('button').filter({ hasText: /^\s*板名追加\s*$/ }).first().click()
  await expect(boardDialog, '板名追加ダイアログが閉じない').toBeHidden({ timeout: 15000 })

  // 作った板が v-select に選ばれた状態になること
  const boardSelect = dialog.locator('.v-select input').first()
  await expect(boardSelect, '作った板名が選択されていない').toHaveValue(boardName, { timeout: 15000 })
}

/**
 * 対象の記録にフォーカスして rykv の Kyou詳細ペインを開き、そのペインを返す。
 *
 * 記録に付いたテキストと通知は **一覧には出ない**。
 * kyou-list-view.vue が `:show_attached_texts="false"`
 * `:show_attached_notifications="false"` を渡しているため。
 * 出るのは rykv-view.vue の詳細ペイン（`:show_attached_texts="true"`）だけなので、
 * テキスト・通知を確認したいテストはここを経由する。
 * （タグは一覧にも出る。`application_config.show_tags_in_list` 次第）
 *
 * 詳細ペインは focused_kyou（＝記録のクリック）と、
 * ツールバーの mdi-file-document ボタン（トグル）の両方が要る。
 */
export async function openKyouDetailPane(page: Page, record: Locator): Promise<Locator> {
  await record.click()

  const pane = page.locator('.kyou_detail_view').first()
  if (!(await pane.isVisible())) {
    const toggle = page.locator('button:has(.mdi-file-document)').first()
    await expect(toggle, '詳細表示切替ボタンが見つからない').toBeVisible({ timeout: 15000 })
    await toggle.click()
  }
  await expect(pane, 'Kyou詳細ペインが開かない').toBeVisible({ timeout: 15000 })
  return pane
}

/**
 * Click the FAB (+) button on rykv page to open add menu.
 * The FAB is a v-btn with mdi-plus icon inside a position-fixed v-avatar.
 */
export async function clickFabButton(page: Page): Promise<void> {
  // Close any floating dialogs (tutorial dialog) that may intercept clicks
  await dismissFloatingDialogs(page)

  // The FAB is: v-avatar.position-fixed > v-menu > v-btn[icon="mdi-plus"]
  const fab = page.locator('.position-fixed button, .position-fixed .v-btn').first()
  await expect(fab, 'FAB(＋ボタン)が見つからない').toBeVisible({ timeout: 15000 })
  await fab.click({ force: true })

  // FABのメニューが開くまで待つ
  await expect(page.locator(CONTEXT_MENU_ITEM).first(), 'FABメニューが開かない').toBeVisible({ timeout: 15000 })
}

/**
 * Dismiss any floating dialogs (e.g., tutorial dialog) that may intercept pointer events.
 */
export async function dismissFloatingDialogs(page: Page): Promise<void> {
  const floatingDialogs = page.locator('.gkill-floating-dialog')
  // チュートリアルダイアログの閉じるボタンはアイコンのみ (mdi-close) でテキストを持たないため、
  // hasText だけでは掴めない。掴み損ねると iframe やチェックボックスが
  // 後続クリックのポインタイベントを奪ってテストが落ちる。
  // 閉じたことを確認するまでリトライする。
  for (let attempt = 0; attempt < 5; attempt++) {
    if (await floatingDialogs.count() === 0) {
      return
    }
    const closeBtn = floatingDialogs.first()
      .locator('button:has(.mdi-close), button:has-text("×"), button:has-text("閉じる"), button:has-text("close")')
      .first()
    if (await closeBtn.count() > 0) {
      await closeBtn.click({ force: true }).catch(() => { /* 閉じかけている最中は無視 */ })
    } else {
      await page.keyboard.press('Escape')
    }
    // 閉じるアニメーションが終わってDOMから消えるのを待つ。
    // 消えなければ次のループでもう一度閉じにいく。
    await floatingDialogs.first().waitFor({ state: 'detached', timeout: 3000 }).catch(() => {
      // まだ残っている場合はループを続ける
    })
  }
}

/**
 * Find and click a kyou item on rykv page that contains the given text.
 * Returns the locator for the found item.
 */
export function findKyouByText(page: Page, text: string) {
  return page.locator('#app').locator(`text=${text}`).first()
}

/**
 * findKyouByText の待機つき版。
 * リストの描画が終わるまで待ってから locator を返す。
 * 並列実行時は描画が間に合わずに count 0 になることがあるため、
 * 「対象が見つかっていること」を前提にするテストはこちらを使う。
 */
export async function waitForKyouByText(page: Page, text: string, timeout = 30000) {
  const record = findKyouByText(page, text)
  await record.waitFor({ state: 'visible', timeout })
  return record
}

/**
 * 記録に付いたタグ / テキスト / 通知を掴む。
 *
 * これらは rykv のサイドバー（タグ絞り込みツリー）にも同じ文字列で現れるため、
 * `findKyouByText` だと先にサイドバー側に当たってしまい、
 * 右クリックしてもコンテキストメニューが開かない。
 * 描画されている要素のクラス（attached-tag.vue / attached-text.vue /
 * attached-notification.vue のルート）で限定する。
 *
 * テキストと通知は一覧に出ないので、`openKyouDetailPane` が返す
 * 詳細ペインを scope として渡すこと。
 */
export async function waitForAttachedTag(page: Page, tagName: string, timeout = 30000) {
  const tag = page.locator('.tag_wrap').filter({ hasText: tagName }).first()
  await expect(tag, `記録に付いたタグ ${tagName} が見つからない`).toBeVisible({ timeout })
  return tag
}

export async function waitForAttachedText(scope: Locator, text: string, timeout = 30000) {
  const element = scope.locator('.text_content').filter({ hasText: text }).first()
  await expect(element, `記録に付いたテキスト ${text} が見つからない`).toBeVisible({ timeout })
  return element
}

export async function waitForAttachedNotification(scope: Locator, content: string, timeout = 30000) {
  const element = scope.locator('.notification_content').filter({ hasText: content }).first()
  await expect(element, `記録に付いた通知 ${content} が見つからない`).toBeVisible({ timeout })
  return element
}

/**
 * 通知ダイアログの通知時刻を設定する。
 *
 * 時刻の入力欄は readonly で、クリックすると v-time-picker が開く。
 * 未設定のままだと use-add-notification-view.ts の
 * notification_time_is_blank でガードされ、保存リクエストが飛ばない。
 */
export async function pickNotificationTime(page: Page): Promise<void> {
  const dialog = page.locator(DIALOG_ROOT).first()
  const timeField = dialog.locator('input[readonly]').last()
  await expect(timeField, '通知時刻の入力欄が見つからない').toBeVisible({ timeout: 15000 })
  await timeField.click()

  const picker = page.locator('.v-time-picker').first()
  await expect(picker, '時刻ピッカーが開かない').toBeVisible({ timeout: 15000 })
  // 時 → 分 の順に1つずつ選ぶとピッカーが閉じる
  await picker.locator('.v-time-picker-clock__item, button').first().click()
  await picker.locator('.v-time-picker-clock__item, button').first().click()

  await expect(timeField, '時刻が設定されていない').not.toHaveValue('')
}
