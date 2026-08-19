import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('can navigate to settings page', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/saihate/)
  })

  test('settings page renders content', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // **`.v-application` の可視で待たないこと。** 最果ては v-main の中身が空なので
    // ルート要素の高さが 0 になり、Playwright は hidden と判定する（/kyou や /rykv では通る）。
    // 画面ごとに「本当に出るもの」を待つ
    await expect(page.locator('.v-toolbar-title'), 'アプリバーのタイトルが出ない')
      .toBeVisible({ timeout: 30000 })
    const appContent = await page.locator('#app').innerHTML()
    expect(appContent.length).toBeGreaterThan(100)
  })

  test('settings page renders without JavaScript errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // 画面が立ち上がりきってから拾う（固定sleepだと遅い環境で取りこぼす）
    await expect(page.locator('.v-toolbar-title'), 'アプリバーのタイトルが出ない')
      .toBeVisible({ timeout: 30000 })
    // Filter out known benign errors
    const criticalErrors = errors.filter(e =>
      !e.includes('ResizeObserver') &&
      !e.includes('devtools') &&
      !e.includes('[hmr]') &&
      !e.includes('Failed to fetch') &&
      !e.includes('Unexpected end of JSON input')
    )
    expect(criticalErrors.length).toBe(0)
  })

  test('settings page has buttons or interactive controls', async ({ page }) => {
    await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // 固定sleepではなく、操作できる要素が1つでも描かれるのを待つ
    await expect(page.locator('button, input, .v-switch, [role="switch"]').first(),
      '設定画面に操作できる要素が1つも無い').toBeVisible({ timeout: 30000 })
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
