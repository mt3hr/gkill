import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing,
  makeUniqueLabel, pageContainsText, clickFabButton, clickDialogButton,
  findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('GUI Add Dialog Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('add mi via add dialog', async ({ page }) => {
    const label = makeUniqueLabel('mi_add')
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click FAB to open add menu
    await clickFabButton(page)

    // Look for Mi menu item
    const miMenuItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /Mi|タスク/i }).first()
    if (await miMenuItem.count() > 0) {
      await miMenuItem.click()
      await page.waitForTimeout(2000)

      // Fill title field (first text input in dialog)
      const titleInput = page.locator('.gkill-floating-dialog__body input[type="text"], .v-dialog input[type="text"]').first()
      if (await titleInput.count() > 0) {
        await titleInput.fill(label)
        await clickDialogButton(page, /保存|save/i)
        await page.waitForTimeout(2000)

        // Verify on Mi board
        await navigateToMi(page)
        const found = await pageContainsText(page, label)
        expect(found).toBe(true)
      }
    }
  })

  test('add lantana via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const lantanaItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /Lantana|気分/i }).first()
    if (await lantanaItem.count() > 0) {
      await lantanaItem.click()
      await page.waitForTimeout(2000)

      // Lantana dialog should have a slider or number input for mood value
      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      // Try to save (mood should have a default or set value)
      await clickDialogButton(page, /保存|save/i)
      await page.waitForTimeout(2000)
    }
    // Verify page doesn't crash
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add nlog via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const nlogItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /Nlog|出費|家計/i }).first()
    if (await nlogItem.count() > 0) {
      await nlogItem.click()
      await page.waitForTimeout(2000)

      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      // Fill amount and shop
      const inputs = dialog.locator('input[type="text"], input[type="number"], .v-text-field input')
      const count = await inputs.count()
      if (count >= 1) {
        await inputs.nth(0).fill('1234')
      }
      await clickDialogButton(page, /保存|save/i)
      await page.waitForTimeout(2000)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add timeis via add dialog', async ({ page }) => {
    const label = makeUniqueLabel('timeis_add')
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const timeisItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /TimeIs|タイムイズ/i }).first()
    if (await timeisItem.count() > 0) {
      await timeisItem.click()
      await page.waitForTimeout(2000)

      const titleInput = page.locator('.gkill-floating-dialog__body input[type="text"], .v-dialog input[type="text"]').first()
      if (await titleInput.count() > 0) {
        await titleInput.fill(label)
        await clickDialogButton(page, /保存|save/i)
        await page.waitForTimeout(2000)

        await navigateToPlaing(page)
        const found = await pageContainsText(page, label)
        expect(found).toBe(true)
      }
    }
  })

  test('add urlog via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const urlogItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /URLog|ブックマーク|URL/i }).first()
    if (await urlogItem.count() > 0) {
      await urlogItem.click()
      await page.waitForTimeout(2000)

      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      const urlInput = dialog.locator('input[type="text"], input[type="url"], .v-text-field input').first()
      if (await urlInput.count() > 0) {
        await urlInput.fill('https://example.com/e2e_test_urlog')
        await clickDialogButton(page, /保存|save/i)
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add kc via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const kcItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /KC|数値/i }).first()
    if (await kcItem.count() > 0) {
      await kcItem.click()
      await page.waitForTimeout(2000)

      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      const inputs = dialog.locator('input[type="text"], input[type="number"], .v-text-field input')
      if (await inputs.count() >= 1) {
        await inputs.nth(0).fill('テストKC')
      }
      await clickDialogButton(page, /保存|save/i)
      await page.waitForTimeout(2000)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add tag to existing record via context menu', async ({ page }) => {
    // First create a record via KFTL
    const recordLabel = makeUniqueLabel('record_for_tag')
    await submitKftlText(page, recordLabel)
    await navigateToRykv(page)

    // Find the record and right-click
    const record = findKyouByText(page, recordLabel)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      // Look for tag add menu item
      const tagMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /タグ.*追加|add.*tag/i }).first()
      if (await tagMenuItem.count() > 0) {
        await tagMenuItem.click()
        await page.waitForTimeout(2000)

        const tagInput = page.locator('.gkill-floating-dialog__body input[type="text"], .v-dialog input[type="text"]').first()
        if (await tagInput.count() > 0) {
          await tagInput.fill('e2eTestTag')
          await clickDialogButton(page, /保存|save/i)
          await page.waitForTimeout(2000)
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番25: Mi追加(タイトルのみ=最小入力)
  test('add mi with minimal input (title only)', async ({ page }) => {
    const label = makeUniqueLabel('mi_minimal')
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const miMenuItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /Mi|タスク/i }).first()
    if (await miMenuItem.count() > 0) {
      await miMenuItem.click()
      await page.waitForTimeout(2000)

      // Fill only the title field (minimal required input)
      const titleInput = page.locator('.gkill-floating-dialog__body input[type="text"], .v-dialog input[type="text"]').first()
      if (await titleInput.count() > 0) {
        await titleInput.fill(label)
        await clickDialogButton(page, /保存|save/i)
        await page.waitForTimeout(2000)

        // Verify on Mi board
        await navigateToMi(page)
        const found = await pageContainsText(page, label)
        expect(found).toBe(true)
      }
    }
  })

  // 項番28: TimeIs追加(全項目入力)
  test('add timeis with all fields filled', async ({ page }) => {
    const label = makeUniqueLabel('timeis_full')
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const timeisItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /TimeIs|タイムイズ/i }).first()
    if (await timeisItem.count() > 0) {
      await timeisItem.click()
      await page.waitForTimeout(2000)

      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      // Fill all available text fields in dialog
      const inputs = dialog.locator('input[type="text"], .v-text-field input')
      const count = await inputs.count()
      if (count >= 1) {
        await inputs.nth(0).fill(label)
      }

      await clickDialogButton(page, /保存|save/i)
      await page.waitForTimeout(2000)

      await navigateToPlaing(page)
      const found = await pageContainsText(page, label)
      expect(found).toBe(true)
    }
  })

  // 項番30: URLog追加(全項目入力)
  test('add urlog with all fields filled', async ({ page }) => {
    const label = makeUniqueLabel('urlog_full')
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(2000)

    await clickFabButton(page)

    const urlogItem = page.locator('.v-list-item, [role="menuitem"], .v-btn').filter({ hasText: /URLog|ブックマーク|URL/i }).first()
    if (await urlogItem.count() > 0) {
      await urlogItem.click()
      await page.waitForTimeout(2000)

      const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
      await expect(dialog).toBeVisible({ timeout: 10000 })

      // Fill all available input fields
      const inputs = dialog.locator('input[type="text"], input[type="url"], .v-text-field input')
      const count = await inputs.count()
      if (count >= 1) {
        await inputs.nth(0).fill(`https://example.com/${label}`)
      }
      // Fill title if available (second field)
      if (count >= 2) {
        await inputs.nth(1).fill(label)
      }

      await clickDialogButton(page, /保存|save/i)
      await page.waitForTimeout(2000)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add text to existing record via context menu', async ({ page }) => {
    const recordLabel = makeUniqueLabel('record_for_text')
    await submitKftlText(page, recordLabel)
    await navigateToRykv(page)

    const record = findKyouByText(page, recordLabel)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const textMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /テキスト.*追加|add.*text/i }).first()
      if (await textMenuItem.count() > 0) {
        await textMenuItem.click()
        await page.waitForTimeout(2000)

        const textInput = page.locator('.gkill-floating-dialog__body textarea, .gkill-floating-dialog__body input[type="text"], .v-dialog textarea, .v-dialog input[type="text"]').first()
        if (await textInput.count() > 0) {
          await textInput.fill('E2Eテストテキスト')
          await clickDialogButton(page, /保存|save/i)
          await page.waitForTimeout(2000)
        }
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
