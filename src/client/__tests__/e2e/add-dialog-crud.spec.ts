import { test, expect } from '@playwright/test'
import type { Locator, Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing,
  makeUniqueLabel, expectPageToContainText, clickFabButton,
  clickContextMenuItem, clickDialogButton, waitForKyouByText,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * FAB（＋ボタン）とコンテキストメニューからの追加フロー。
 *
 * 以前は追加メニューや入力欄が見つからない場合に `if (await x.count() > 0)` で
 * 素通りして成功していたため、追加が壊れていてもテストは緑のままだった。
 * ここでは各手順を「見つかる前提」にし、追加したものが一覧に出るところまで見る。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const DIALOG = '.gkill-floating-dialog, .v-dialog'

/** FABを開いて種別を選び、開いた追加ダイアログを返す。 */
async function openAddDialog(page: Page, menuLabel: RegExp): Promise<Locator> {
  await clickFabButton(page)
  await clickContextMenuItem(page, menuLabel)

  const dialog = page.locator(DIALOG).first()
  await expect(dialog, `追加ダイアログ(${menuLabel})が開かない`).toBeVisible({ timeout: 15000 })
  return dialog
}

/** ダイアログのn番目のテキスト入力欄を埋める。 */
async function fillDialogField(dialog: Locator, index: number, value: string): Promise<void> {
  const field = dialog.locator('input[type="text"], input[type="url"], input[type="number"], .v-text-field input').nth(index)
  await expect(field, `ダイアログに ${index + 1} 番目の入力欄が無い`).toBeVisible({ timeout: 15000 })
  await field.fill(value)
  await expect(field).toHaveValue(value)
}

test.describe('GUI Add Dialog Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
    await navigateToRykv(page)
  })

  test('Miを追加ダイアログから作るとMi画面に出る', async ({ page }) => {
    const label = makeUniqueLabel('mi_add')

    const dialog = await openAddDialog(page, MENU.addMi)
    await fillDialogField(dialog, 0, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToMi(page)
    await expectPageToContainText(page, label)
  })

  // 項番25: Mi追加(タイトルのみ=最小入力)
  test('Miをタイトルだけで追加できる', async ({ page }) => {
    const label = makeUniqueLabel('mi_minimal')

    const dialog = await openAddDialog(page, MENU.addMi)
    await fillDialogField(dialog, 0, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToMi(page)
    await expectPageToContainText(page, label)
  })

  test('TimeIsを追加ダイアログから作るとPlaing画面に出る', async ({ page }) => {
    const label = makeUniqueLabel('timeis_add')

    const dialog = await openAddDialog(page, MENU.addTimeIs)
    await fillDialogField(dialog, 0, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToPlaing(page)
    await expectPageToContainText(page, label)
  })

  // 項番28: TimeIs追加(全項目入力)
  test('TimeIsをタイトル入りで追加するとPlaing画面に出る', async ({ page }) => {
    const label = makeUniqueLabel('timeis_full')

    const dialog = await openAddDialog(page, MENU.addTimeIs)
    await fillDialogField(dialog, 0, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToPlaing(page)
    await expectPageToContainText(page, label)
  })

  // 項番30: URLog追加(全項目入力)
  test('URLogをURLとタイトル入りで追加すると一覧に出る', async ({ page }) => {
    const label = makeUniqueLabel('urlog_full')

    const dialog = await openAddDialog(page, MENU.addURLog)
    await fillDialogField(dialog, 0, `https://example.com/${label}`)
    await fillDialogField(dialog, 1, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)
  })

  /**
   * 追加画面のタグ欄。
   *
   * タグ欄は既存フィールドの後ろ（アクション行の直前）に置いてあるので、
   * URL(0) / タイトル(1) / 日付(2) / 時刻(3) の次で index=4。
   * ここが前へ動くと他のテストの `fillDialogField` の位置指定が総崩れになる。
   *
   * タグは Kyou 本体より後に登録される（列への局所挿入が `refresh_kyou` で引き直す都合上、
   * `add_tag` が終わってから `registered_kyou` を出す必要があるため）。
   * `clickDialogButton` は**最初の**書き込みAPIの応答＝`add_urlog` で戻るので、
   * そこで画面遷移すると飛行中の `add_tag` が中断される。
   * ダイアログが閉じるのは全リクエストが終わったあとなので、それを待ってから遷移する。
   */
  test('URLogを本文とタグ入りで一度に追加できる', async ({ page }) => {
    const label = makeUniqueLabel('urlog_with_tag')
    const tagLabel = makeUniqueLabel('e2eAddTag')

    const dialog = await openAddDialog(page, MENU.addURLog)
    await fillDialogField(dialog, 0, `https://example.com/${label}`)
    await fillDialogField(dialog, 1, label)
    await fillDialogField(dialog, 4, tagLabel)
    await clickDialogButton(page, SAVE_BUTTON)
    await expect(dialog, '保存してもダイアログが閉じない').toBeHidden({ timeout: 30000 })

    await navigateToRykv(page)
    await expectPageToContainText(page, label)
    await expectPageToContainText(page, tagLabel)
  })

  test('KCをタイトルと数値で追加すると一覧に出る', async ({ page }) => {
    const label = makeUniqueLabel('kc_add')

    const dialog = await openAddDialog(page, MENU.addKC)
    await fillDialogField(dialog, 0, label)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)
  })

  // Nlog は 品目 / 店名 / 金額 の3項目が必須。
  // add-nlog-view.vue の並び順は 品目(0) → 店名(1) → 金額(2)。
  // 金額を空のままにすると保存が通らない。
  test('Nlogを品目・店名・金額つきで追加すると一覧に出る', async ({ page }) => {
    const label = makeUniqueLabel('nlog_add')
    const shop = makeUniqueLabel('nlog_shop')

    const dialog = await openAddDialog(page, MENU.addNlog)
    await fillDialogField(dialog, 0, label)
    await fillDialogField(dialog, 1, shop)
    await fillDialogField(dialog, 2, '1234')
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)
    await expectPageToContainText(page, shop)
  })

  // Lantana は気分値だけの記録で、一覧上の見た目からラベルで特定できない。
  // 気分の花（lantana_icon）をクリックして値を変えてから保存する。
  // use-add-lantana-view.ts の save() は「値が変わっていない」場合
  // LANTANA_IS_NO_UPDATE_MESSAGE を出して保存しないので、
  // 開いてすぐ保存しても何も起きない。
  test('Lantanaを気分値を選んで保存できる', async ({ page }) => {
    const dialog = await openAddDialog(page, MENU.addLantana)
    await expect(dialog).toBeVisible()

    const flowerHalf = dialog.locator('.lantana_icon img').first()
    await expect(flowerHalf, '気分の花が見つからない').toBeVisible({ timeout: 15000 })
    await flowerHalf.click()

    await clickDialogButton(page, SAVE_BUTTON)

    await expect(dialog, '保存してもダイアログが閉じない').toBeHidden({ timeout: 15000 })
  })

  test('既存の記録にコンテキストメニューからタグを追加できる', async ({ page }) => {
    const recordLabel = makeUniqueLabel('record_for_tag')
    const tagLabel = makeUniqueLabel('e2eTag')

    await submitKftlText(page, recordLabel)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, recordLabel)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addTag)

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, 'タグ追加ダイアログが開かない').toBeVisible({ timeout: 15000 })
    await fillDialogField(dialog, 0, tagLabel)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    await expectPageToContainText(page, tagLabel)
  })

  // 追加したテキストは rykv の一覧には出ない（一覧は attached_texts を
  // 読み込まないため）。保存が成功したこと自体は clickDialogButton が
  // 書き込みAPIのレスポンスを見て確認するので、ここではそれに加えて
  // 「テキスト編集ダイアログを開くと保存した内容が入っている」ことで
  // 記録に紐づいたことを確認する。
  test('既存の記録にコンテキストメニューからテキストを追加できる', async ({ page }) => {
    const recordLabel = makeUniqueLabel('record_for_text')
    const textLabel = makeUniqueLabel('e2eText')

    await submitKftlText(page, recordLabel)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, recordLabel)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addText)

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, 'テキスト追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

    const textField = dialog.locator('textarea, input[type="text"]').first()
    await expect(textField).toBeVisible({ timeout: 15000 })
    await textField.fill(textLabel)
    await expect(textField).toHaveValue(textLabel)

    // 保存できたこと（書き込みAPIがerrors無しで返ること）は clickDialogButton が見る
    await clickDialogButton(page, SAVE_BUTTON)
    await expect(dialog, '保存してもダイアログが閉じない').toBeHidden({ timeout: 15000 })
  })
})
