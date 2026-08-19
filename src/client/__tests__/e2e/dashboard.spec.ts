import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

/**
 * ダッシュボードを開いて、アプリバーが描けるまで待つ。
 *
 * 固定sleepの代わりに「日付ボタンが出た」を合図にする。
 * dashboard-view.vue のアプリバーは 前日(chevron-left) / 日付 / 翌日(chevron-right) の並び。
 */
async function openDashboard(page: Page) {
  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  const dateButton = page.locator('.v-app-bar button.text-none').first()
  await expect(dateButton, 'ダッシュボードの日付ボタンが出ない').toBeVisible({ timeout: 30000 })
  return dateButton
}

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ page }) => {
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('can navigate to dashboard page', async ({ page }) => {
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/dashboard/)
  })

  test('dashboard page renders content', async ({ page }) => {
    await openDashboard(page)
    const appContent = await page.locator('#app').innerHTML()
    expect(appContent.length).toBeGreaterThan(100)
  })

  test('dashboard page renders without JavaScript errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    await openDashboard(page)
    const criticalErrors = errors.filter(e =>
      !e.includes('ResizeObserver') &&
      !e.includes('devtools') &&
      !e.includes('[hmr]') &&
      !e.includes('Failed to fetch') &&
      !e.includes('Unexpected end of JSON input')
    )
    expect(criticalErrors.length).toBe(0)
  })

  test('dashboard page has date navigation buttons', async ({ page }) => {
    await openDashboard(page)
    // 前日・翌日ボタン（chevron left/right）が存在する
    await expect(page.locator('button .mdi-chevron-left').first(), '前日ボタンが無い').toBeVisible()
    await expect(page.locator('button .mdi-chevron-right').first(), '翌日ボタンが無い').toBeVisible()
  })

  test('dashboard page has floating action button', async ({ page }) => {
    await openDashboard(page)
    // FABボタン（+アイコン）が存在する
    await expect(page.locator('.v-avatar button, .position-fixed button').first(), 'FABが無い')
      .toBeVisible({ timeout: 30000 })
  })

  test('dashboard title is shown in toolbar', async ({ page }) => {
    await openDashboard(page)
    // ダッシュボードタイトルが表示されていること（日本語または英語）
    await expect(page.locator('.v-toolbar-title'), 'アプリバーにタイトルが出ない')
      .toHaveText(/ダッシュボード|Dashboard/i, { timeout: 30000 })
  })

  test('prev day button changes date', async ({ page }) => {
    const dateButton = await openDashboard(page)
    const before = await dateButton.textContent()

    await page.locator('button .mdi-chevron-left').first().click()

    // 「押しただけ」で終わらせず、表示が実際に前日へ動くことを見る
    await expect(dateButton, '前日ボタンを押しても日付が変わらない')
      .not.toHaveText(before ?? '', { timeout: 15000 })
  })

  test('next day button changes date', async ({ page }) => {
    const dateButton = await openDashboard(page)
    const before = await dateButton.textContent()

    await page.locator('button .mdi-chevron-right').first().click()

    await expect(dateButton, '翌日ボタンを押しても日付が変わらない')
      .not.toHaveText(before ?? '', { timeout: 15000 })
  })

  test('settings button opens application config dialog', async ({ page }) => {
    await openDashboard(page)

    // 設定ボタン（mdi-cog）をクリック。条件で包むと、ボタンが無いときに
    // 何も検証せずに緑になってしまう
    const settingsBtn = page.locator('button:has(.mdi-cog)').first()
    await expect(settingsBtn, '設定ボタンが無い').toBeVisible({ timeout: 15000 })
    await settingsBtn.click()

    await expect(page.locator('.gkill-floating-dialog').last(), '設定ダイアログが開かない')
      .toBeVisible({ timeout: 30000 })
  })
})
