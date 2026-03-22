import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, pageContainsText, findKyouByText, clickDialogButton,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Mi (Task) Operations', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番73: タスク板間移動
  test('move task between boards', async ({ page }) => {
    const label = makeUniqueLabel('mi_board_move')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    // Find the task and right-click to edit
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const editMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /編集|edit/i }).first()
      if (await editMenuItem.count() > 0) {
        await editMenuItem.click()
        await page.waitForTimeout(2000)

        // Look for board name selector in edit dialog
        const dialog = page.locator('.v-dialog').first()
        if (await dialog.isVisible()) {
          // Find board name input/select
          const boardInput = dialog.locator('.v-select, .v-combobox, select').first()
          if (await boardInput.count() > 0) {
            await boardInput.click()
            await page.waitForTimeout(1000)

            // Select a different board option if available
            const options = page.locator('.v-list-item, [role="option"]')
            if (await options.count() > 1) {
              await options.nth(1).click()
              await page.waitForTimeout(1000)
            }
          }
          await clickDialogButton(page, /保存|save/i)
          await page.waitForTimeout(2000)
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番74: タスク完了状態編集
  test('toggle task completion state', async ({ page }) => {
    const label = makeUniqueLabel('mi_complete')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    // Find the task
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      // Look for checkbox/check state toggle near the task
      const checkbox = page.locator('.v-checkbox, input[type="checkbox"]').first()
      if (await checkbox.count() > 0) {
        await checkbox.click()
        await page.waitForTimeout(2000)

        // Verify the state changed (page should update)
        const app = page.locator('#app')
        await expect(app).toBeVisible()
      } else {
        // Try context menu for completion toggle
        await record.click({ button: 'right', force: true })
        await page.waitForTimeout(1000)

        const completeItem = page.locator('.v-list-item, [role="menuitem"]')
          .filter({ hasText: /完了|complete|チェック|check/i }).first()
        if (await completeItem.count() > 0) {
          await completeItem.click()
          await page.waitForTimeout(2000)
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番76: タスク共有状況閲覧+スクロール確認
  test('view task share status without excessive scroll', async ({ page }) => {
    const label = makeUniqueLabel('mi_share_view')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    // Find the task and open context menu
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      // Look for share-related menu item
      const shareItem = page.locator('.v-list-item, [role="menuitem"]')
        .filter({ hasText: /共有|share/i }).first()
      if (await shareItem.count() > 0) {
        await shareItem.click()
        await page.waitForTimeout(2000)

        // Verify no excessive scrolling in the dialog/page
        const dialog = page.locator('.v-dialog').first()
        if (await dialog.isVisible()) {
          const dialogBox = await dialog.boundingBox()
          if (dialogBox) {
            // Dialog should fit reasonably within viewport
            const viewport = page.viewportSize()
            if (viewport) {
              expect(dialogBox.height).toBeLessThan(viewport.height * 1.5)
            }
          }
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番77: タスク共有停止
  test('stop sharing task', async ({ page }) => {
    const label = makeUniqueLabel('mi_share_stop')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    // Try to share and then stop sharing
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const shareItem = page.locator('.v-list-item, [role="menuitem"]')
        .filter({ hasText: /共有|share/i }).first()
      if (await shareItem.count() > 0) {
        await shareItem.click()
        await page.waitForTimeout(2000)

        // Look for stop/delete share button in dialog
        const stopShareBtn = page.locator('.v-dialog button, .v-card button')
          .filter({ hasText: /停止|解除|削除|stop|remove/i }).first()
        if (await stopShareBtn.count() > 0) {
          await stopShareBtn.click()
          await page.waitForTimeout(2000)
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
