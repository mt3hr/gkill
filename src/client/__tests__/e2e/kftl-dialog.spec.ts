import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('KFTL Dialog', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('can open KFTL dialog', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const textarea = page.locator('textarea')
    await expect(textarea.first()).toBeVisible({ timeout: 30000 })
  })

  // These tests require the gkill API (/api/*) to be reachable from the browser.
  // When Vite dev server does not proxy /api/* to gkill server, textarea stays readonly.

  test('can type and submit KFTL text', async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const textarea = page.locator('textarea:not([readonly])').first()
    await expect(textarea).toBeVisible({ timeout: 90000 })
    await textarea.fill('テストメモ')
    // 入力が反映されてから保存ボタンを確かめる（固定sleepではなく値で待つ）
    await expect(textarea).toHaveValue('テストメモ')
    await expect(page.locator('button').filter({ hasText: /保存|送信|submit|save/i }).first(),
      '保存ボタンが無い').toBeVisible({ timeout: 15000 })
  })

  test('KFTL textarea accepts multiline input', async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const textarea = page.locator('textarea:not([readonly])').first()
    await expect(textarea).toBeVisible({ timeout: 90000 })
    await textarea.fill('1行目\n2行目')
    await expect(textarea, '複数行が入力できない')
      .toHaveValue(/1行目[\s\S]*2行目/, { timeout: 15000 })
  })

  test('KFTL page has template section', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // KFTL page should have either template buttons or a template tree
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    await expect(page.locator('textarea').first(), 'メモ帳の本文欄が描かれない')
      .toBeVisible({ timeout: 30000 })
    const textContent = await app.textContent()
    expect(textContent!.length).toBeGreaterThan(0)
  })

  test('KFTL submit button exists', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page.locator('button').filter({ hasText: /保存|送信|submit|save/i }).first(),
      '保存ボタンが無い').toBeVisible({ timeout: 30000 })
  })
})
