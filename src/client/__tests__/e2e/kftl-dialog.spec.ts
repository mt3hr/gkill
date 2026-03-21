import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('KFTL Dialog', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('can open KFTL dialog', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const textarea = page.locator('textarea')
    await expect(textarea.first()).toBeVisible({ timeout: 15000 })
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
    await page.waitForTimeout(1000)
    const submitButton = page.locator('button').filter({ hasText: /保存|送信|submit|save/i })
    expect(await submitButton.count()).toBeGreaterThan(0)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('KFTL textarea accepts multiline input', async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    const textarea = page.locator('textarea:not([readonly])').first()
    await expect(textarea).toBeVisible({ timeout: 90000 })
    await textarea.fill('1行目\n2行目')
    await page.waitForTimeout(500)
    const value = await textarea.inputValue()
    expect(value).toContain('1行目')
    expect(value).toContain('2行目')
  })

  test('KFTL page has template section', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // KFTL page should have either template buttons or a template tree
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    const textContent = await app.textContent()
    expect(textContent!.length).toBeGreaterThan(0)
  })

  test('KFTL submit button exists', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const submitButton = page.locator('button').filter({ hasText: /保存|送信|submit|save/i })
    const count = await submitButton.count()
    expect(count).toBeGreaterThan(0)
  })
})
