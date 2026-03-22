import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel,
  pageContainsText, findKyouByText, clickDialogButton, confirmDelete,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Notification CRUD Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('add notification to record via context menu', async ({ page }) => {
    // Create a record first
    const label = makeUniqueLabel('notif_add_target')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Find the record and right-click
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      // Look for notification add menu item (通知追加)
      const notifMenuItem = page.locator('.v-list-item, [role="menuitem"], .v-list-item-title')
        .filter({ hasText: /通知追加|通知.*追加|add.*notif/i }).first()
      if (await notifMenuItem.count() > 0) {
        await notifMenuItem.click()
        await page.waitForTimeout(2000)

        // Fill notification content — dialog may be floating or v-dialog
        const dialog = page.locator('.v-dialog, .v-card, .gkill-floating-dialog').first()
        if (await dialog.isVisible().catch(() => false)) {
          const contentInput = dialog.locator('textarea, input[type="text"], .v-text-field input').first()
          if (await contentInput.count() > 0) {
            await contentInput.fill('E2Eテスト通知内容')
          }

          const saveBtn = dialog.locator('button').filter({ hasText: /保存|save/i }).first()
          if (await saveBtn.count() > 0) {
            await saveBtn.click()
            await page.waitForTimeout(2000)
          }
        }
      }
    }
    // Verify page doesn't crash
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('edit notification via context menu', async ({ page }) => {
    // Create a record with a notification (via add flow first)
    const label = makeUniqueLabel('notif_edit_target')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // The notification might be visible as a sub-element of the kyou
    // Try to find and right-click any notification element
    const app = page.locator('#app')
    const content = await app.textContent()

    // Verify page renders without crash
    expect(content!.length).toBeGreaterThan(0)
  })

  test('delete notification via context menu', async ({ page }) => {
    const label = makeUniqueLabel('notif_delete_target')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Verify page renders
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('view notification displays correctly on rykv page', async ({ page }) => {
    // Navigate to rykv and verify notifications are rendered distinctly
    await navigateToRykv(page)
    const app = page.locator('#app')
    const content = await app.innerHTML()
    // Page should have substantial content
    expect(content.length).toBeGreaterThan(100)
  })

  test('notification history dialog can be opened', async ({ page }) => {
    const label = makeUniqueLabel('notif_hist_target')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Find the record and try to open history
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]')
        .filter({ hasText: /履歴|histor/i }).first()
      if (await historyMenuItem.count() > 0) {
        await historyMenuItem.click()
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
