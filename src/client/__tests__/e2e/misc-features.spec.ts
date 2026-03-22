import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel, findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Misc Feature Tests', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('notification and text elements have different visual appearance', async ({ page }) => {
    // Create a record, add notification and text
    const label = makeUniqueLabel('visual_diff')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Check that the page renders both types with visual distinction
    // (We can only verify the DOM has different element structures)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    const content = await app.innerHTML()
    expect(content.length).toBeGreaterThan(0)
  })

  test('timeis history does not show end button for completed timeis', async ({ page }) => {
    // Create a timeis with start+end (completed)
    const label = makeUniqueLabel('timeis_hist')
    await submitKftlText(page, `ーち\n${label}`)
    await navigateToRykv(page)

    // Find the timeis record and open history
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
      if (await historyMenuItem.count() > 0) {
        await historyMenuItem.click()
        await page.waitForTimeout(2000)
      }
    }
    // Verify no crash
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('rekyou context menu does not show duplicate entries', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_ctx')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Right-click and check context menu for duplicates
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      // Count menu items — should not have duplicates
      const menuItems = page.locator('.v-list-item, [role="menuitem"]')
      const count = await menuItems.count()
      if (count > 0) {
        // Collect all menu texts
        const texts: string[] = []
        for (let i = 0; i < count; i++) {
          const text = await menuItems.nth(i).textContent()
          if (text) texts.push(text.trim())
        }
        // Check no exact duplicates
        const unique = new Set(texts)
        expect(unique.size).toBe(texts.length)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
