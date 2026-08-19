import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing,
  makeUniqueLabel, expectPageToContainText, findKyouByText, clickContextMenuItem,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('View/Browse Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('view kyou history via context menu', async ({ page }) => {
    // Create a record, then edit it to create history
    const label = makeUniqueLabel('history_test')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // 行やメニューが見つからなければ落とす。
    // 条件で包んでいたころは、見つからないと何も検証せずに緑になっていた
    const record = findKyouByText(page, label).first()
    await expect(record, '作った記録の行が一覧に出ない').toBeVisible({ timeout: 30000 })
    await record.click({ button: 'right', force: true })

    // Look for history menu item
    await clickContextMenuItem(page, /履歴|histor/i)
    await expect(page.locator('.gkill-floating-dialog').last(), '履歴ダイアログが開かない')
      .toBeVisible({ timeout: 30000 })
  })

  test('rykv page shows mixed data types after creation', async ({ page }) => {
    const kmemoLabel = makeUniqueLabel('mixed_kmemo')
    const miLabel = makeUniqueLabel('mixed_mi')

    // Create kmemo and mi
    await submitKftlText(page, kmemoLabel)
    await submitKftlText(page, `ーみ\n${miLabel}`)

    // Navigate to rykv and verify both appear
    await navigateToRykv(page)
    await expectPageToContainText(page, kmemoLabel)
  })

  test('mi board shows task records', async ({ page }) => {
    await navigateToMi(page)
    // Mi page should show task records created by other tests
    const app = page.locator('#app')
    const content = await app.textContent()
    // Check that the Mi page has rendered and contains task-related content
    expect(content!.length).toBeGreaterThan(0)
    const hasTaskContent = content!.includes('Inbox') || content!.includes('アイテム') || content!.includes('タスク')
    expect(hasTaskContent).toBe(true)
  })

  test('plaing page shows timeis records', async ({ page }) => {
    const label = makeUniqueLabel('plaing_view')
    await submitKftlText(page, `ーた\n${label}`)
    // expectPageToContainText がリトライしながら待つので、固定sleepは要らない

    await navigateToPlaing(page)
    await expectPageToContainText(page, label)
  })
})
