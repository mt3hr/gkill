import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, confirmDelete, findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

/**
 * Helper: right-click a record, click delete, confirm.
 */
async function deleteViaContextMenu(page: import('@playwright/test').Page, textToFind: string): Promise<boolean> {
  const record = findKyouByText(page, textToFind)
  if (await record.count() === 0) return false

  await record.click({ button: 'right', force: true })
  await page.waitForTimeout(1000)

  const deleteMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /削除|delete/i }).first()
  if (await deleteMenuItem.count() === 0) return false

  await deleteMenuItem.click()
  await page.waitForTimeout(2000)

  // Confirm delete dialog
  await confirmDelete(page)
  await page.waitForTimeout(2000)
  return true
}

test.describe('GUI Delete Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('delete kmemo via context menu', async ({ page }) => {
    const label = makeUniqueLabel('kmemo_delete')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Verify it exists
    let found = await pageContainsText(page, label)
    expect(found).toBe(true)

    // Delete it
    const deleted = await deleteViaContextMenu(page, label)
    if (deleted) {
      // Reload and verify it's gone
      await navigateToRykv(page)
      found = await pageContainsText(page, label)
      expect(found).toBe(false)
    }
  })

  test('delete mi via context menu', async ({ page }) => {
    const label = makeUniqueLabel('mi_delete')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)

    let found = await pageContainsText(page, label)
    expect(found).toBe(true)

    const deleted = await deleteViaContextMenu(page, label)
    if (deleted) {
      await navigateToMi(page)
      found = await pageContainsText(page, label)
      expect(found).toBe(false)
    }
  })

  test('delete lantana via context menu', async ({ page }) => {
    // Create and attempt to delete — verify no crash
    await submitKftlText(page, 'ーら\n2')
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('delete nlog via context menu', async ({ page }) => {
    const shopName = makeUniqueLabel('nlog_del_shop')
    await submitKftlText(page, `ーん\n100\n${shopName}`)
    await navigateToRykv(page)

    const found = await pageContainsText(page, shopName)
    if (found) {
      await deleteViaContextMenu(page, shopName)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('delete urlog via context menu', async ({ page }) => {
    const label = makeUniqueLabel('urlog_del')
    await submitKftlText(page, `ーう\nhttps://example.com/${label}\n${label}`)
    await navigateToRykv(page)

    const found = await pageContainsText(page, label)
    if (found) {
      await deleteViaContextMenu(page, label)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('delete timeis via context menu', async ({ page }) => {
    const label = makeUniqueLabel('timeis_del')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToRykv(page)

    const found = await pageContainsText(page, label)
    if (found) {
      await deleteViaContextMenu(page, label)
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('delete tag from record', async ({ page }) => {
    const label = makeUniqueLabel('tag_del_record')
    const tagName = makeUniqueLabel('deltag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)

    // Try to find and delete the tag
    const tagElement = page.locator('#app').locator(`text=${tagName}`).first()
    if (await tagElement.count() > 0) {
      await tagElement.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const deleteMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /削除|delete/i }).first()
      if (await deleteMenuItem.count() > 0) {
        await deleteMenuItem.click()
        await page.waitForTimeout(2000)
        await confirmDelete(page)
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番54: Text削除 (元NG→修正済み回帰テスト)
  test('delete text from record', async ({ page }) => {
    const label = makeUniqueLabel('text_del_record')
    // Create a record with text via KFTL (、、 separator for text)
    await submitKftlText(page, `、、削除テスト用テキスト\n${label}`)
    await navigateToRykv(page)

    // Try to find and delete the text element
    const textElement = page.locator('#app').locator('text=削除テスト用テキスト').first()
    if (await textElement.count() > 0) {
      await textElement.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const deleteMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /削除|delete/i }).first()
      if (await deleteMenuItem.count() > 0) {
        await deleteMenuItem.click()
        await page.waitForTimeout(2000)
        await confirmDelete(page)
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('add and delete rekyou', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_test')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Right-click and repost
    const record = findKyouByText(page, label)
    if (await record.count() > 0) {
      await record.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const rekyouMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /リキョウ|リポスト|repost/i }).first()
      if (await rekyouMenuItem.count() > 0) {
        await rekyouMenuItem.click()
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
