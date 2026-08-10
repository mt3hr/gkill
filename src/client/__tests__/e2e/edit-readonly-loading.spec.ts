import { test, expect } from '@playwright/test'
import type { Locator, Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, waitForKyouByText,
  clickFabButton, clickContextMenuItem,
  MENU,
} from './crud-helpers'

/**
 * オーバーレイ（v-menu の中身）を、その外側を押して閉じる。
 *
 * オーバーレイは画面に収まるよう上下に反転するので、
 * 「この要素を押せば外側」と決め打ちにすると、反転したときに覆われてクリックが届かない。
 * オーバーレイ自身の外接矩形から外側の点を求めることで、開く向きに依存しなくなる。
 */
async function closeOverlayByClickingOutside(page: Page, overlay: Locator): Promise<void> {
  const box = await overlay.first().boundingBox()
  expect(box, 'オーバーレイの外接矩形が取れない').not.toBeNull()
  const viewport = page.viewportSize()
  expect(viewport, 'ビューポートサイズが取れない').not.toBeNull()

  const centerY = box!.y + box!.height / 2
  // 左に十分な余白があれば左外側、無ければ下外側を押す
  const target = box!.x > 20
    ? { x: Math.max(2, box!.x - 10), y: centerY }
    : { x: box!.x + box!.width / 2, y: Math.min(viewport!.height - 2, box!.y + box!.height + 10) }
  await page.mouse.click(target.x, target.y)
}

/**
 * Add/Edit系ダイアログの「処理中は入力フォームreadonly」検証。
 *
 * readonly は is_busy = is_loading || is_requested_submit にバインドされている
 * (use-edit-kmemo-view.ts)。このうち is_loading 側は、リストから開いた Kyou は
 * clone() が is_typed_data_loaded を引き継ぐため load_typed_datas() が早期 return し、
 * 通信が発生せず観測可能なウィンドウにならない。
 * そのため確実に到達する is_requested_submit 側 (保存中) を検証する。
 * 保存APIの応答を遅延させて Submitting ウィンドウを人工的に広げる。
 */

let apiReachable = false
test.beforeAll(async () => {
  apiReachable = await checkGkillServer()
  test.skip(!apiReachable, 'gkill server is not running')
})

test.describe('Edit Dialog Readonly While Submitting', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('edit kmemo inputs are readonly during save and editable after', async ({ page }) => {
    const label = makeUniqueLabel('dlg_ro')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // 右クリック → 編集 で編集ダイアログを開く
    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    const editItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /編集|edit/i }).first()
    await expect(editItem).toBeVisible({ timeout: 10000 })
    await editItem.click()

    const textarea = page.locator('.gkill-floating-dialog .v-textarea textarea').first()
    await expect(textarea).toBeVisible({ timeout: 10000 })

    // 保存前: 編集可能であること
    await expect(textarea).toHaveJSProperty('readOnly', false, { timeout: 20000 })

    // 保存APIを遅延させ、Submitting 中の状態を観測可能にする
    await page.route('**/api/update_kmemo', async (route) => {
      await new Promise((r) => setTimeout(r, 5000))
      await route.continue()
    })

    const saveButton = page.locator('.gkill-floating-dialog button')
      .filter({ hasText: /保存|save/i }).first()
    const editedValue = `${label}_edited`

    // ダイアログの load() が非同期で kmemo_value を入れ直すため、fill した内容が
    // 巻き戻ることがある。巻き戻ったまま保存すると save() が「更新なし」で
    // 早期 return してしまい Submitting ウィンドウが発生しない。
    // 「入力 → 保存 → readonly になる」までを一括でリトライして確実に観測する。
    await expect(async () => {
      await textarea.fill(editedValue)
      await expect(textarea).toHaveValue(editedValue, { timeout: 2000 })
      await expect(saveButton).toBeEnabled({ timeout: 5000 })
      await saveButton.click()
      // 保存中: 本文 textarea が readonly であること
      await expect(textarea).toHaveJSProperty('readOnly', true, { timeout: 3000 })
    }).toPass({ timeout: 60000 })

    // 保存完了でダイアログが閉じること (= 実際に更新リクエストが通ったこと)
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 30000 })
  })

  /**
   * Add系は入力欄の readonly は入っていたが、日時ピッカーを開く v-menu に
   * :disabled が無く、保存中でも日付・時刻を変更できてしまっていた。
   * 日時の v-text-field は元から静的 readonly（ピッカー専用）なので、
   * 塞ぐべきは v-menu 側であり、readOnly プロパティでは検出できない。
   */
  test('add urlog inputs and date picker are locked during save', async ({ page }) => {
    const label = makeUniqueLabel('add_ro')
    await navigateToRykv(page)

    await clickFabButton(page)
    await clickContextMenuItem(page, MENU.addURLog)

    const dialog = page.locator('.gkill-floating-dialog').first()
    await expect(dialog, 'URLog追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

    const urlField = dialog.locator('.v-text-field input').first()
    await expect(urlField).toBeVisible({ timeout: 15000 })
    await expect(urlField).toHaveJSProperty('readOnly', false, { timeout: 20000 })
    await urlField.fill(`https://example.com/${label}`)

    // 日付ピッカーは v-menu の中身なので、開いているときだけ active な overlay に載る。
    // rykv のサイドバー (calendar-query.vue) が v-show で隠した .v-date-picker を
    // 常時DOMに残しているため、overlay に限定しないと常に1件見つかってしまう。
    const openDatePicker = page.locator('.v-overlay--active .v-date-picker')
    const dateField = dialog.locator('.v-text-field input').nth(2)

    // 保存前は開くこと。これが無いと後段の「開かない」検証が
    // セレクタ間違いでも通ってしまう
    await dateField.click()
    await expect(openDatePicker, '保存前なのに日付ピッカーが開かない').toHaveCount(1, { timeout: 15000 })
    // ピッカーの外側をクリックして閉じる。
    // 画面下端に近い位置で開くとピッカーは上向きに反転し、上にあるURL欄を覆う。
    // そのため「URL欄をクリックする」ような固定の相手では、覆われてクリックが届かない。
    // ピッカー自身の外接矩形から外側の点を計算して押す。
    // Escapeはダイアログごと閉じうるので使わない。
    await closeOverlayByClickingOutside(page, openDatePicker)
    await expect(openDatePicker, '日付ピッカーが閉じない').toHaveCount(0, { timeout: 15000 })

    // 保存APIを遅延させ、Submitting 中の状態を観測可能にする。
    // 遅延中に readonly と日付ピッカーの両方を見るので、edit側より長めに取る
    await page.route('**/api/add_urlog', async (route) => {
      await new Promise((r) => setTimeout(r, 15000))
      await route.continue()
    })

    const saveButton = dialog.locator('button').filter({ hasText: /保存|save/i }).first()
    await expect(saveButton).toBeEnabled({ timeout: 15000 })
    await saveButton.click()

    // 保存中: URL欄が readonly になっていること
    await expect(urlField, '保存中にURL欄が編集できる').toHaveJSProperty('readOnly', true, { timeout: 10000 })

    // 保存中: 日付欄を押しても日付ピッカーが開かないこと
    await dateField.click({ force: true })
    await expect(openDatePicker, '保存中に日付ピッカーが開いてしまう')
      .toHaveCount(0, { timeout: 5000 })

    // 保存完了でダイアログが閉じること
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 30000 })
  })
})
