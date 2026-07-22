import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, waitForKyouByText,
} from './crud-helpers'

/**
 * Edit系ダイアログの「処理中は入力フォームreadonly」検証。
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
})
