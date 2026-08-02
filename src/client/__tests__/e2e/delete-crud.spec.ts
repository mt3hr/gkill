import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, confirmDelete, clickDialogButton,
  clickContextMenuItem, waitForKyouByText, waitForAttachedTag, waitForAttachedText, openKyouDetailPane, waitForKyouRowByRepName, searchByKeyword,
  expectPageToContainText, expectPageNotToContainText,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * コンテキストメニューからの削除フロー。
 *
 * 以前は削除ヘルパが「レコードが見つからなければ false を返す」設計で、
 * 呼び出し側も `if (deleted) { ... }` としていたため、削除が動かなくても
 * テストは成功していた。ここでは削除できることを前提にし、
 * 削除後に一覧から消えるところまで確認する。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/** 指定テキストの要素を右クリックして削除する。見つからなければ失敗する。 */
async function deleteViaContextMenu(page: Page, textToFind: string): Promise<void> {
  const record = await waitForKyouByText(page, textToFind)
  await record.click({ button: 'right', force: true })
  await clickContextMenuItem(page, MENU.delete)
  await confirmDelete(page)
}

test.describe('GUI Delete Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // rykv 上で本文/タイトルから特定できるデータ型は、
  // 作成 → 一覧に出る → 削除 → 一覧から消える、を通しで確認する。
  const rykvCases = [
    { name: 'kmemo', prefix: 'kmemo_delete', kftl: (l: string) => l },
    { name: 'nlog', prefix: 'nlog_del_shop', kftl: (l: string) => `ーん\n${l}\nテスト品目\n100` },
    { name: 'urlog', prefix: 'urlog_del', kftl: (l: string) => `ーう\nhttps://example.com/${l}\n${l}` },
    { name: 'timeis', prefix: 'timeis_del', kftl: (l: string) => `ーた\n${l}` },
  ]

  for (const testCase of rykvCases) {
    test(`${testCase.name} をコンテキストメニューから削除すると一覧から消える`, async ({ page }) => {
      const label = makeUniqueLabel(testCase.prefix)
      await submitKftlText(page, testCase.kftl(label))
      await navigateToRykv(page)

      await expectPageToContainText(page, label)
      await deleteViaContextMenu(page, label)

      await navigateToRykv(page)
      await expectPageNotToContainText(page, label)
    })
  }

  test('Mi をコンテキストメニューから削除するとMi画面から消える', async ({ page }) => {
    const label = makeUniqueLabel('mi_delete')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    await expectPageToContainText(page, label)
    await deleteViaContextMenu(page, label)

    await navigateToMi(page)
    await expectPageNotToContainText(page, label)
  })

  test('記録に付けたタグを削除すると表示から消える', async ({ page }) => {
    const label = makeUniqueLabel('tag_del_record')
    const tagName = makeUniqueLabel('deltag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)

    // タグ名は rykv サイドバーの絞り込みツリーにも出るので、
    // 一覧に描画されたタグ（.tag_wrap）を掴む
    const tag = await waitForAttachedTag(page, tagName)
    await tag.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.deleteTag)
    await confirmDelete(page)

    await navigateToRykv(page)
    await expect(page.locator('.tag_wrap').filter({ hasText: tagName }), 'タグが消えていない')
      .toHaveCount(0, { timeout: 30000 })
    // タグを消しても記録自体は残る
    await expectPageToContainText(page, label)
  })

  // 項番54: Text削除 (元NG→修正済み回帰テスト)
  //
  // 記録に付いたテキストは一覧には描画されない
  // （kyou-list-view.vue が :show_attached_texts="false" を渡している）。
  // 出るのは rykv の Kyou詳細ペインだけなので、そこを経由して操作する。
  test('記録に付けたテキストを削除すると詳細ペインから消える', async ({ page }) => {
    const label = makeUniqueLabel('text_del_record')
    const textBody = makeUniqueLabel('deltext')

    // KFTL の「、、」は以降の行をすべてテキスト本文として取り込むため
    // (kftl-text-statement-line.ts)、記録＋テキストを別々に作れない。
    // 記録を作ってから、コンテキストメニューでテキストを付ける。
    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addText)

    const addDialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(addDialog, 'テキスト追加ダイアログが開かない').toBeVisible({ timeout: 15000 })
    const textField = addDialog.locator('textarea, input[type="text"]').first()
    await expect(textField).toBeVisible({ timeout: 15000 })
    await textField.fill(textBody)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    let pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))

    const text = await waitForAttachedText(pane, textBody)
    await text.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.deleteText)
    await confirmDelete(page)

    await navigateToRykv(page)
    pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))
    await expect(pane.locator('.text_content').filter({ hasText: textBody }), 'テキストが消えていない')
      .toHaveCount(0, { timeout: 30000 })
    // テキストを消しても記録自体は残る
    await expectPageToContainText(page, label)
  })

  // リポストは元の記録と同じ本文で表示されるため、本文だけでは元記録と区別できない。
  // 本文で掴んで削除すると元記録のほうを消してしまい、ターゲットを失った
  // リポストも一覧から落ちて件数が 0 になる。
  // リポジトリ名（ReKyou）と本文の両方で行を特定する。
  test('リポストを作って削除しても元の記録は残る', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_test')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // リポストはコンテキストメニュー → 確認ダイアログの「リポスト」で確定する
    // (confirm-re-kyou-view.vue)
    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.rekyou)
    await clickDialogButton(page, MENU.rekyou)

    // 一覧は仮想スクロールで数件しか描画せず、並列に走る他テストの記録に
    // 押し出されるので、この本文で絞ってから件数を見る。
    //
    // リポストは元の記録を入れ子で描画する（re-kyou-view.vue が
    // <KyouView :show_rep_name="true"> で元Kyouを中に出す）ため、
    // `.kyou_rep_name` の総数は「元の記録 / リポスト / リポスト内の元の記録」で3になる。
    // 総数で数えると読み違えるので、リポジトリ名で絞って数える。
    await navigateToRykv(page)
    await searchByKeyword(page, label)
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*ReKyou\s*$/ }), 'リポストが作られていない')
      .toHaveCount(1, { timeout: 30000 })

    // リポストだけを削除する
    // 検索でこの本文だけに絞ってあるので、ReKyou の行はこの1件だけ
    const rekyouRow = await waitForKyouRowByRepName(page, 'ReKyou')
    await rekyouRow.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.delete)
    await confirmDelete(page)

    // 元の記録（Kmemo）だけが残る
    await navigateToRykv(page)
    await searchByKeyword(page, label)
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*ReKyou\s*$/ }), 'リポストが消えていない')
      .toHaveCount(0, { timeout: 30000 })
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*Kmemo\s*$/ }), '元の記録まで消えている')
      .toHaveCount(1, { timeout: 30000 })
  })
})
