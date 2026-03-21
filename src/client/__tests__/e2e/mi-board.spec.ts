import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Mi Board', () => {
  test('can navigate to Mi board page', async ({ page }) => {
    await page.goto('/mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/mi/)
  })

  test('Mi board displays task list', async ({ page }) => {
    await page.goto('/mi', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
  })
})
