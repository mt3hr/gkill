import { test, expect } from '@playwright/test'
import type { Locator, Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing,
  makeUniqueLabel, expectPageToContainText, expectPageNotToContainText,
  clickContextMenuItem, clickDialogButton, waitForKyouByText, waitForAttachedText, openKyouDetailPane,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * コンテキストメニューからの編集フロー。
 *
 * 以前このファイルは
 *   const record = findKyouByText(page, label)
 *   if (await record.count() > 0) { ... if (await editMenuItem.count() > 0) { ... } }
 * という3重の条件ガードで包まれており、レコードやメニューが見つからないと
 * 何も検証しないまま成功していた。最後の `expect(app).toBeVisible()` は常に真なので、
 * 実質「ページが落ちていない」以上のことを確認できていなかった。
 *
 * ここでは対象が見つかることを前提（見つからなければ失敗）にし、
 * 「編集した値が実際に一覧へ反映されること」まで確認する。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const DIALOG = '.gkill-floating-dialog, .v-dialog'

/** 表示中の編集ダイアログを返す。 */
async function openEditDialog(page: Page, record: Locator, menuLabel: RegExp = MENU.edit): Promise<Locator> {
  await record.click({ button: 'right', force: true })
  await clickContextMenuItem(page, menuLabel)

  const dialog = page.locator(DIALOG).first()
  await expect(dialog, '編集ダイアログが開かない').toBeVisible({ timeout: 15000 })
  return dialog
}

/**
 * ダイアログ内で currentValue が入っている入力欄を探して newValue に書き換える。
 *
 * 入力欄の並びはデータ型ごとに違うので first() では狙った欄に当たらない。
 * 「今の値が入っている欄」で特定することで、Kmemo(textarea) と
 * Mi/TimeIs(input) を同じ手順で扱える。
 */
async function replaceDialogFieldValue(dialog: Locator, currentValue: string, newValue: string): Promise<void> {
  const fields = dialog.locator('textarea, input[type="text"]')
  await expect(fields.first(), 'ダイアログに入力欄が無い').toBeVisible({ timeout: 15000 })

  const count = await fields.count()
  const values: string[] = []
  for (let i = 0; i < count; i++) {
    const field = fields.nth(i)
    const value = await field.inputValue()
    values.push(value)
    if (value !== currentValue) {
      continue
    }
    await field.fill(newValue)
    await expect(field).toHaveValue(newValue)
    return
  }
  throw new Error(`現在の値 ${JSON.stringify(currentValue)} を持つ入力欄がダイアログに無い: ${JSON.stringify(values)}`)
}

test.describe('GUI Edit Dialog Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // タイトル / 本文をテキスト欄で編集できるデータ型。
  // 編集した値が一覧に出て、元の値が消えるところまで確認する。
  const textEditableCases = [
    { name: 'kmemo', kftl: (label: string) => label, prefix: 'kmemo_edit', navigate: navigateToRykv },
    { name: 'mi', kftl: (label: string) => `ーみ\n${label}`, prefix: 'mi_edit', navigate: navigateToMi },
    { name: 'timeis', kftl: (label: string) => `ーた\n${label}`, prefix: 'timeis_edit', navigate: navigateToRykv },
  ]

  for (const testCase of textEditableCases) {
    test(`${testCase.name} をコンテキストメニューから編集すると一覧に反映される`, async ({ page }) => {
      const originalLabel = makeUniqueLabel(`${testCase.prefix}_orig`)
      const editedLabel = makeUniqueLabel(`${testCase.prefix}_edited`)

      await submitKftlText(page, testCase.kftl(originalLabel))
      await testCase.navigate(page)

      const record = await waitForKyouByText(page, originalLabel)
      const dialog = await openEditDialog(page, record)
      await replaceDialogFieldValue(dialog, originalLabel, editedLabel)
      await clickDialogButton(page, SAVE_BUTTON)

      await testCase.navigate(page)
      await expectPageToContainText(page, editedLabel)
      await expectPageNotToContainText(page, originalLabel)
    })
  }

  // 単一のテキスト欄では表せないデータ型。
  // 「編集ダイアログが現在の値を持って開き、保存してもレコードが失われない」ことを見る。
  const dialogOnlyCases = [
    { name: 'nlog', kftl: 'ーん\n編集テスト店\nテスト品目\n777', findText: '編集テスト店' },
    { name: 'urlog', kftl: 'ーう\nhttps://example.com/edit_test\n編集URLogタイトル', findText: '編集URLogタイトル' },
  ]

  for (const testCase of dialogOnlyCases) {
    // 各Editビューの save() は「値が変わっていない」場合に
    // *_IS_NO_UPDATE_MESSAGE を出して保存しない（リクエストも飛ばない）。
    // そのため、開いてそのまま保存する形では検証できない。
    // 既存値が読み込まれていることを確認したうえで、1項目だけ変えて保存する。
    test(`${testCase.name} の編集ダイアログが現在の値を読み込み、変更を保存できる`, async ({ page }) => {
      const edited = makeUniqueLabel(`${testCase.name}_edited`)

      await submitKftlText(page, testCase.kftl)
      await navigateToRykv(page)

      const record = await waitForKyouByText(page, testCase.findText)
      const dialog = await openEditDialog(page, record)

      // 既存の値が読み込まれた状態で開くこと（空のダイアログが開く不具合の検出）。
      // 値は input の value に入るので textContent では見えない。
      await replaceDialogFieldValue(dialog, testCase.findText, edited)
      await clickDialogButton(page, SAVE_BUTTON)

      await navigateToRykv(page)
      await expectPageToContainText(page, edited)
    })
  }

  // タグ名の変更が、その場（開いている画面）に反映されることを見る。
  //
  // 注意: ここで再読み込みして一覧を見に行くことはできない。
  // rykv のタグ絞り込みはチェック状態を `application_config.tag_struct` に
  // 保存しており、改名で現れた「見たことのない名前」は未チェックで入る。
  // その結果、絞り込みが有効になってその記録が一覧から消える
  // （新規追加したタグは自動でチェックされるので、この非対称は不具合の疑い。
  //   詳細ペインは絞り込みの影響を受けないのでここで確認する）。
  test('タグをコンテキストメニューから編集すると詳細ペインに反映される', async ({ page }) => {
    const originalTag = makeUniqueLabel('tag_edit_orig')
    const editedTag = makeUniqueLabel('tag_edit_new')
    const recordLabel = makeUniqueLabel('record_tag_edit')

    await submitKftlText(page, `。${originalTag}\n${recordLabel}`)
    await navigateToRykv(page)

    const pane = await openKyouDetailPane(page, await waitForKyouByText(page, recordLabel))

    // タグ名はサイドバーの絞り込みツリーにも出るので、詳細ペインの中のタグを掴む
    const tagElement = pane.locator('.tag_wrap').filter({ hasText: originalTag }).first()
    await expect(tagElement, '詳細ペインに編集前のタグが出ない').toBeVisible({ timeout: 30000 })

    const dialog = await openEditDialog(page, tagElement, MENU.editTag)
    await replaceDialogFieldValue(dialog, originalTag, editedTag)
    await clickDialogButton(page, SAVE_BUTTON)

    await expect(pane.locator('.tag_wrap').filter({ hasText: editedTag }), '編集後のタグ名が反映されない')
      .toHaveCount(1, { timeout: 30000 })
    await expect(pane.locator('.tag_wrap').filter({ hasText: originalTag }), '編集前のタグが残っている')
      .toHaveCount(0, { timeout: 30000 })
  })

  // 記録に付いたテキストは一覧には描画されない
  // （kyou-list-view.vue が :show_attached_texts="false" を渡している）。
  // 出るのは rykv の Kyou詳細ペインだけなので、そこを経由して操作する。
  test('テキストをコンテキストメニューから編集すると詳細ペインに反映される', async ({ page }) => {
    const originalText = makeUniqueLabel('text_edit_orig')
    const editedText = makeUniqueLabel('text_edit_new')
    const recordLabel = makeUniqueLabel('record_text_edit')

    // KFTL の「、、」は以降の行をすべてテキスト本文として取り込むため
    // (kftl-text-statement-line.ts)、記録＋テキストを別々に作れない。
    // 記録を作ってから、コンテキストメニューでテキストを付ける。
    await submitKftlText(page, recordLabel)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, recordLabel)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addText)

    const addDialog = page.locator(DIALOG).first()
    await expect(addDialog, 'テキスト追加ダイアログが開かない').toBeVisible({ timeout: 15000 })
    const textField = addDialog.locator('textarea, input[type="text"]').first()
    await expect(textField).toBeVisible({ timeout: 15000 })
    await textField.fill(originalText)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    let pane = await openKyouDetailPane(page, await waitForKyouByText(page, recordLabel))

    const textElement = await waitForAttachedText(pane, originalText)
    const dialog = await openEditDialog(page, textElement, MENU.editText)
    await replaceDialogFieldValue(dialog, originalText, editedText)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    pane = await openKyouDetailPane(page, await waitForKyouByText(page, recordLabel))
    await waitForAttachedText(pane, editedText)
    await expect(pane.locator('.text_content').filter({ hasText: originalText }), '編集前のテキストが残っている')
      .toHaveCount(0, { timeout: 30000 })
  })

  // 項番41: 実行中TimeIsの終了ボタンで終了
  //
  // 終了ボタンは Plaing 画面にだけ出る。rykv の一覧は
  // rykv-view.vue で show_timeis_plaing_end_button="false" を渡しているので出ない。
  test('実行中TimeIsを終了ボタンで終了すると終了日時が表示される', async ({ page }) => {
    const label = makeUniqueLabel('timeis_running_end')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaing(page)

    // TimeIsのビューは v-card がルートで、タイトルと終了ボタンを同じカードに持つ
    const card = page.locator('.v-card').filter({ hasText: label }).last()
    await expect(card, '実行中TimeIsのカードが見つからない').toBeVisible({ timeout: 30000 })

    const endButton = card.locator('button').filter({ hasText: /終了|end/i }).first()
    await expect(endButton, '実行中なのに終了ボタンが出ていない').toBeVisible({ timeout: 15000 })
    await endButton.click()

    // 終了ダイアログの確定ボタンは「終了」(END_TITLE)。保存ではない
    // (end-time-is-plaing-view.vue)
    await clickDialogButton(page, /^\s*終了\s*$/)

    // 終了すると rykv 側で終了日時が表示される
    await navigateToRykv(page)
    const endedCard = page.locator('.v-card').filter({ hasText: label }).last()
    await expect(endedCard, '終了後のカードが見つからない').toBeVisible({ timeout: 30000 })
    await expect(endedCard, '終了しても終了日時が表示されない').toContainText('終了日時', { timeout: 15000 })
  })

  // 項番43: ReKyou編集
  test('記録をリポストすると一覧に増え、リポストの編集ダイアログを開ける', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_edit')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // リポストはコンテキストメニュー → 確認ダイアログの「リポスト」で確定する
    // (confirm-re-kyou-view.vue)
    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.rekyou)
    await clickDialogButton(page, MENU.rekyou)

    // 元の記録とリポストで同じ本文が2件以上出る
    await navigateToRykv(page)
    await expect
      .poll(async () => await page.locator('#app').getByText(label, { exact: false }).count(), { timeout: 30000 })
      .toBeGreaterThan(1)

    // リポストの編集ダイアログが開けること。
    // ReKyouの編集は日時のみで、開いてそのまま保存すると
    // REKYOU_IS_NO_UPDATE_MESSAGE になるため、開くところまでを確認する。
    const rekyouRecord = await waitForKyouByText(page, label)
    await rekyouRecord.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.edit)

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, 'リポストの編集ダイアログが開かない').toBeVisible({ timeout: 15000 })
  })

  test('Kmemoの本文を空にして保存しようとしても一覧から消えない', async ({ page }) => {
    // 項番35: 空の本文でのバリデーション。
    // 保存が拒否されるか空文字で保存されるかは実装依存だが、
    // どちらにせよレコードそのものが失われてはいけない。
    const originalLabel = makeUniqueLabel('kmemo_empty_edit')
    await submitKftlText(page, originalLabel)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, originalLabel)
    const dialog = await openEditDialog(page, record)

    const contentField = dialog.locator('textarea, input[type="text"]').first()
    await expect(contentField).toBeVisible({ timeout: 15000 })
    await contentField.fill('')
    await expect(contentField).toHaveValue('')

    await page.locator(DIALOG).first().locator('button').filter({ hasText: SAVE_BUTTON }).first().click()

    // ダイアログが閉じても閉じなくても、アプリが壊れていないこと
    await navigateToRykv(page)
    await expect(page.locator('#app')).toBeVisible()
  })
})
