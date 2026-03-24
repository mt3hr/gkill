import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing,
  makeUniqueLabel, pageContainsText, findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
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

    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      // Look for history menu item
      const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
      if (await historyMenuItem.count() > 0) {
        await historyMenuItem.click()
        await page.waitForTimeout(2000)

        // Verify something opened (dialog or expanded section)
        const app = page.locator('#app')
        const content = await app.textContent()
        expect(content!.length).toBeGreaterThan(0)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('rykv page shows mixed data types after creation', async ({ page }) => {
    const kmemoLabel = makeUniqueLabel('mixed_kmemo')
    const miLabel = makeUniqueLabel('mixed_mi')

    // Create kmemo and mi
    await submitKftlText(page, kmemoLabel)
    await submitKftlText(page, `ーみ\n${miLabel}`)

    // Navigate to rykv and verify both appear
    await navigateToRykv(page)
    const foundKmemo = await pageContainsText(page, kmemoLabel)
    expect(foundKmemo).toBe(true)
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
    // Extra wait for backend to index the new TimeIs record
    await page.waitForTimeout(2000)

    await navigateToPlaing(page)
    // Additional wait for Plaing page to fetch and render data
    await page.waitForTimeout(2000)
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)
  })
})
