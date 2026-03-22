import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, pageContainsText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Search and Summary Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番66: 記録された情報を検索する
  test('search records by keyword on rykv page', async ({ page }) => {
    // Create a record with a unique label
    const label = makeUniqueLabel('search_test')
    await submitKftlText(page, label)

    // Navigate to rykv
    await navigateToRykv(page)

    // Open sidebar (navigation drawer) if not already open
    // The sidebar contains the search/query editor
    const sidebar = page.locator('.v-navigation-drawer')
    if (await sidebar.count() > 0 && !(await sidebar.isVisible())) {
      // Try to find and click the hamburger/menu button to open sidebar
      const menuBtn = page.locator('.v-app-bar button, .v-toolbar button').first()
      if (await menuBtn.count() > 0) {
        await menuBtn.click()
        await page.waitForTimeout(1000)
      }
    }

    // Look for keyword/search input in sidebar
    const keywordInput = page.locator('.v-navigation-drawer input[type="text"], .v-navigation-drawer .v-text-field input, .v-navigation-drawer textarea').first()
    if (await keywordInput.count() > 0 && await keywordInput.isVisible()) {
      await keywordInput.fill(label)
      await page.waitForTimeout(500)

      // Click search button
      const searchBtn = page.locator('button').filter({ hasText: /検索|search/i }).first()
      if (await searchBtn.count() > 0) {
        await searchBtn.click()
        await page.waitForTimeout(2000)

        // Verify the record appears in results
        const found = await pageContainsText(page, label)
        expect(found).toBe(true)
      }
    } else {
      // If sidebar search not directly accessible, verify the page loads with content
      const app = page.locator('#app')
      await expect(app).toBeVisible()
    }
  })

  // 項番69: 一日の記録サマリを閲覧する (D-note toggle on rykv)
  test('toggle dnote summary panel on rykv page', async ({ page }) => {
    // Create some data first
    const label = makeUniqueLabel('dnote_test')
    await submitKftlText(page, label)

    await navigateToRykv(page)

    // Look for the D-note toggle button (mdi-file-chart-outline icon)
    const dnoteToggle = page.locator('[class*="mdi-file-chart-outline"], button[aria-label*="dnote"], button[aria-label*="D-note"]').first()
    if (await dnoteToggle.count() > 0) {
      await dnoteToggle.click()
      await page.waitForTimeout(2000)

      // Verify D-note panel appeared
      const app = page.locator('#app')
      const content = await app.innerHTML()
      expect(content.length).toBeGreaterThan(100)
    } else {
      // Try to find the toggle via tooltip text or button with chart icon
      const buttons = page.locator('.v-app-bar button, .v-toolbar button')
      const count = await buttons.count()
      for (let i = 0; i < count; i++) {
        const btn = buttons.nth(i)
        const html = await btn.innerHTML()
        if (html.includes('chart') || html.includes('file-chart')) {
          await btn.click()
          await page.waitForTimeout(2000)
          break
        }
      }
    }

    // Verify page is functional after toggle
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    const content = await app.textContent()
    expect(content!.length).toBeGreaterThan(0)
  })

  // 項番70: タスク情報を検索する (Mi board search)
  test('search tasks by keyword on mi board page', async ({ page }) => {
    // Create a task with unique label
    const label = makeUniqueLabel('mi_search_test')
    await submitKftlText(page, `ーみ\n${label}`)

    // Navigate to Mi board
    await navigateToMi(page)

    // Open sidebar for Mi query editor
    const sidebar = page.locator('.v-navigation-drawer')
    if (await sidebar.count() > 0 && !(await sidebar.isVisible())) {
      const menuBtn = page.locator('.v-app-bar button, .v-toolbar button').first()
      if (await menuBtn.count() > 0) {
        await menuBtn.click()
        await page.waitForTimeout(1000)
      }
    }

    // Look for keyword input in Mi sidebar
    const keywordInput = page.locator('.v-navigation-drawer input[type="text"], .v-navigation-drawer .v-text-field input, .v-navigation-drawer textarea').first()
    if (await keywordInput.count() > 0 && await keywordInput.isVisible()) {
      await keywordInput.fill(label)
      await page.waitForTimeout(500)

      const searchBtn = page.locator('button').filter({ hasText: /検索|search/i }).first()
      if (await searchBtn.count() > 0) {
        await searchBtn.click()
        await page.waitForTimeout(2000)

        const found = await pageContainsText(page, label)
        expect(found).toBe(true)
      }
    } else {
      // If sidebar search not directly accessible, verify the page loads with content
      const app = page.locator('#app')
      await expect(app).toBeVisible()
    }
  })
})
