import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
})

test.describe('Plaing TimeIs Page', () => {
  test('can navigate to Plaing page', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/plaing/)
  })

  test('Plaing page renders app container', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('Plaing page has interactive elements', async ({ page }) => {
    await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await page.waitForTimeout(3000)
    // Plaing is the TimeIs timestamp page; check for buttons or timer elements
    const buttons = page.locator('button')
    const buttonsCount = await buttons.count()
    expect(buttonsCount).toBeGreaterThan(0)
  })
})
