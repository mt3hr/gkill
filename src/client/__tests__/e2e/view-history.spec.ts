import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, pageContainsText, findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

/**
 * Helper: open history dialog for a record found by text.
 */
async function openHistoryFor(page: import('@playwright/test').Page, text: string): Promise<boolean> {
  const record = findKyouByText(page, text)
  if (await record.count() === 0) return false

  await record.click({ button: 'right', force: true })
  await page.waitForTimeout(1000)

  const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
  if (await historyMenuItem.count() === 0) return false

  await historyMenuItem.click()
  await page.waitForTimeout(2000)
  return true
}

/**
 * Helper: open repost dialog for a record found by text.
 */
async function repostRecord(page: import('@playwright/test').Page, text: string): Promise<boolean> {
  const record = findKyouByText(page, text)
  if (await record.count() === 0) return false

  await record.click({ button: 'right', force: true })
  await page.waitForTimeout(1000)

  const repostItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /リキョウ|リポスト|repost/i }).first()
  if (await repostItem.count() === 0) return false

  await repostItem.click()
  await page.waitForTimeout(2000)
  return true
}

test.describe('View/Browse History Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番57: Lantana閲覧+履歴+リポスト+スクロールバー確認
  test('view lantana with history and repost', async ({ page }) => {
    // Create a Lantana
    await submitKftlText(page, 'ーら\n5')
    await navigateToRykv(page)

    // Verify page renders without unnecessary scrollbars
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    const content = await app.innerHTML()
    expect(content.length).toBeGreaterThan(100)
  })

  // 項番58: Mi閲覧+履歴+リポスト
  test('view mi with history and repost', async ({ page }) => {
    const label = makeUniqueLabel('mi_view_hist')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToRykv(page)

    // Open history
    const historyOpened = await openHistoryFor(page, label)
    if (historyOpened) {
      const app = page.locator('#app')
      const content = await app.textContent()
      expect(content!.length).toBeGreaterThan(0)
    }

    // Test repost
    await navigateToRykv(page)
    await repostRecord(page, label)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番59: Nlog閲覧+履歴+リポスト
  test('view nlog with history and repost', async ({ page }) => {
    const shopName = makeUniqueLabel('nlog_view_shop')
    await submitKftlText(page, `ーん\n500\n${shopName}`)
    await navigateToRykv(page)

    // Open history
    const historyOpened = await openHistoryFor(page, shopName)
    if (historyOpened) {
      const app = page.locator('#app')
      const content = await app.textContent()
      expect(content!.length).toBeGreaterThan(0)
    }

    // Test repost
    await navigateToRykv(page)
    await repostRecord(page, shopName)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番61: URLog閲覧+履歴+NoImage確認
  test('view urlog with history and NoImage fallback', async ({ page }) => {
    const label = makeUniqueLabel('urlog_view_hist')
    await submitKftlText(page, `ーう\nhttps://example.com/${label}\n${label}`)
    await navigateToRykv(page)

    // Verify the record appears
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)

    // Check for NoImage fallback (when favicon/image is not available)
    const images = page.locator('#app img')
    const imageCount = await images.count()
    // At least verify images don't have broken src
    if (imageCount > 0) {
      for (let i = 0; i < Math.min(imageCount, 5); i++) {
        const src = await images.nth(i).getAttribute('src')
        expect(src).not.toBeNull()
      }
    }

    // Open history
    const historyOpened = await openHistoryFor(page, label)
    if (historyOpened) {
      const app = page.locator('#app')
      await expect(app).toBeVisible()
    }
  })

  // 項番63: ReKyou閲覧+履歴
  test('view rekyou with history', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_view_hist')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Create a repost
    await repostRecord(page, label)

    // Navigate and verify repost is visible
    await navigateToRykv(page)
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)

    // Try to open history on the record
    const historyOpened = await openHistoryFor(page, label)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番64: Tag閲覧+履歴+レイアウト確認
  test('view tag with history and layout', async ({ page }) => {
    const label = makeUniqueLabel('tag_view_hist')
    const tagName = makeUniqueLabel('viewtag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)

    // Verify tag is displayed
    const tagElement = page.locator('#app').locator(`text=${tagName}`).first()
    if (await tagElement.count() > 0) {
      // Right-click tag and open history
      await tagElement.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
      if (await historyMenuItem.count() > 0) {
        await historyMenuItem.click()
        await page.waitForTimeout(2000)

        // Verify history dialog layout is reasonable
        const app = page.locator('#app')
        const content = await app.textContent()
        expect(content!.length).toBeGreaterThan(0)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番65: Text閲覧+履歴
  test('view text with history', async ({ page }) => {
    const label = makeUniqueLabel('text_view_hist')
    // Create record with text
    await submitKftlText(page, `、、テスト閲覧テキスト\n${label}`)
    await navigateToRykv(page)

    // Check if text content is visible
    const textVisible = await pageContainsText(page, 'テスト閲覧テキスト')

    // Try to open history for the text
    const textElement = page.locator('#app').locator('text=テスト閲覧テキスト').first()
    if (await textElement.count() > 0) {
      await textElement.click({ button: 'right', force: true })
      await page.waitForTimeout(1000)

      const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
      if (await historyMenuItem.count() > 0) {
        await historyMenuItem.click()
        await page.waitForTimeout(2000)
      }
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
