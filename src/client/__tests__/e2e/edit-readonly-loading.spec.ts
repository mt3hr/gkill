import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, findKyouByText,
} from './crud-helpers'

/**
 * Edit系ダイアログの「Loading中は入力フォームreadonly」検証。
 * API応答を遅延させてLoadingウィンドウを人工的に広げ、
 * その間の入力が readonly でブロックされることを確認する。
 */

let apiReachable = false
test.beforeAll(async () => {
  apiReachable = await checkGkillServer()
  test.skip(!apiReachable, 'gkill server is not running')
})

test.describe('Edit Dialog Readonly While Loading', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('edit kmemo inputs are readonly during load and editable after', async ({ page }) => {
    const label = makeUniqueLabel('dlg_ro')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // 以降の Kyou 再読込APIを遅延させ、Loading 中の状態を観測可能にする
    await page.route('**/api/get_kyous', async (route) => {
      await new Promise((r) => setTimeout(r, 3000))
      await route.continue()
    })

    // 右クリック → 編集 で編集ダイアログを開く
    const record = findKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await page.waitForTimeout(1000)
    const editItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /編集|edit/i }).first()
    expect(await editItem.count()).toBeGreaterThan(0)
    await editItem.click()

    // Loading 中: 本文 textarea が readonly であること
    const textarea = page.locator('.gkill-floating-dialog .v-textarea textarea').first()
    await expect(textarea).toBeVisible({ timeout: 5000 })
    await expect(textarea).toHaveJSProperty('readOnly', true)

    // Load 完了後: 編集可能に戻ること
    await expect(textarea).toHaveJSProperty('readOnly', false, { timeout: 20000 })
  })
})
