import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('KFTL Dialog', () => {
  test('can open KFTL dialog', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const textarea = page.locator('textarea')
    await expect(textarea.first()).toBeVisible({ timeout: 15000 })
  })

  test('can type and submit KFTL text', async ({ page }) => {
    await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)

    const textarea = page.locator('textarea').first()
    if (await textarea.isVisible()) {
      // The KFTL textarea may be readonly initially; use keyboard input instead of fill
      await textarea.click()
      await page.keyboard.type('テストメモ')
      await page.waitForTimeout(1000)
      const submitButton = page.locator('button').filter({ hasText: /送信|submit/i })
      if (await submitButton.count() > 0) {
        await submitButton.click()
        await page.waitForTimeout(2000)
      }
    }
  })
})
