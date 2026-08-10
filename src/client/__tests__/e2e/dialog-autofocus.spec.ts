import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  navigateToMi, navigateToRykv, submitKftlText, makeUniqueLabel,
  waitForKyouByText, clickContextMenuItem, clickFabButton,
  MENU,
} from './crud-helpers'

/**
 * ダイアログを開いたら最初のテキスト入力欄にカーソルが載っていること。
 *
 * 判定は classes/dialog-autofocus.ts の純関数で、選び方そのものは
 * __tests__/unit/classes/dialog-autofocus.test.ts が固定している。
 * ここで見るのは「実際にダイアログを開いたときに当たるか」＝
 * use-floating-dialog.ts の配線と、入力欄が v-if で遅れて生えるケース。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/** フォーカスが当たっている要素の素性。三項演算子を使わずに済むよう Boolean/String で潰す */
async function readFocusedElement(page: import('@playwright/test').Page) {
  return page.evaluate(() => {
    const element = document.activeElement as HTMLElement
    return {
      tag: String(element.tagName),
      in_dialog_body: Boolean(element.closest('.gkill-floating-dialog__body')),
      in_dialog_header: Boolean(element.closest('.gkill-floating-dialog__header')),
      is_readonly: Boolean((element as HTMLInputElement).readOnly),
      label: String(element.closest('.v-input')?.textContent).trim(),
    }
  })
}

test.describe('ダイアログの自動フォーカス', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  /**
   * 板名追加ダイアログ (new-board-name-dialog.vue) は入力欄に autofocus を
   * 書いていない。共通実装が当てられているかを見るのに一番素直な題材。
   * ヘッダの透過チェックボックスを掴んでいないことも同時に確かめる。
   */
  test('入力欄に autofocus を書いていないダイアログでもカーソルが載る', async ({ page }) => {
    await navigateToMi(page)
    await clickFabButton(page)
    await clickContextMenuItem(page, MENU.addMi)

    const addMiDialog = page.locator('.gkill-floating-dialog').last()
    await expect(addMiDialog, 'タスク追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

    // 板名追加ダイアログ (mdi-plus) を重ねて開く
    const addBoardButton = addMiDialog.locator('button:has(.mdi-plus)').first()
    await expect(addBoardButton, '板名追加ボタン(＋)が見つからない').toBeVisible({ timeout: 15000 })
    await addBoardButton.click()

    const boardDialog = page.locator('.gkill-floating-dialog').last()
    await expect(boardDialog.locator('input[type="text"]'), '板名追加ダイアログが開かない')
      .toBeVisible({ timeout: 15000 })

    await expect(boardDialog.locator('input:focus'), '板名の入力欄にカーソルが載っていない')
      .toHaveCount(1, { timeout: 15000 })

    const focused = await readFocusedElement(page)
    expect(focused.tag).toBe('INPUT')
    expect(focused.in_dialog_body, 'ダイアログ本文の外にフォーカスしている').toBe(true)
    expect(focused.in_dialog_header, 'ヘッダの透過チェックボックスを掴んでいる').toBe(false)
    expect(focused.is_readonly, '読み取り専用の欄にフォーカスしている').toBe(false)

    // そのまま打ち込めること
    await page.keyboard.type('autofocus_board')
    await expect(boardDialog.locator('input:focus')).toHaveValue('autofocus_board')
  })

  /**
   * 既に autofocus を書いてある画面 (add-tag-view.vue) では Vuetify に任せる。
   * 共通実装が割り込んで別の欄を掴んでいないことを見る。
   */
  test('autofocus を書いてある画面ではその欄にカーソルが載る', async ({ page }) => {
    const label = makeUniqueLabel('autofocus_tag')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addTag)

    const dialog = page.locator('.gkill-floating-dialog').last()
    await expect(dialog, 'タグ追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

    await expect(dialog.locator('input:focus'), 'タグ名の入力欄にカーソルが載っていない')
      .toHaveCount(1, { timeout: 15000 })

    const focused = await readFocusedElement(page)
    expect(focused.tag).toBe('INPUT')
    expect(focused.in_dialog_body).toBe(true)
    expect(focused.label, 'タグ名以外の欄にフォーカスしている').toContain('タグ')
  })

  /**
   * 入力欄が無いダイアログ (確認系) では何も掴まない。
   * ヘッダの透過チェックボックスや×ボタンを掴むと Enter で誤操作になる。
   */
  test('入力欄が無いダイアログではフォーカスを動かさない', async ({ page }) => {
    const label = makeUniqueLabel('autofocus_confirm')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.delete)

    const dialog = page.locator('.gkill-floating-dialog').last()
    await expect(dialog, '削除確認ダイアログが開かない').toBeVisible({ timeout: 15000 })

    // 入力欄そのものが無いので、当たりようがない
    await expect(dialog.locator('.gkill-floating-dialog__body input:not([type="checkbox"])'))
      .toHaveCount(0, { timeout: 15000 })

    const focused = await readFocusedElement(page)
    expect(focused.in_dialog_header, 'ヘッダの操作要素を掴んでいる').toBe(false)
  })
})
