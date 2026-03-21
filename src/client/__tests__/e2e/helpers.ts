import type { Page } from '@playwright/test'

/**
 * Login as admin user (default: admin with empty password).
 * Navigates to login page, fills credentials, clicks login, waits for redirect.
 * Returns true if login succeeded (URL changed), false otherwise.
 */
export async function loginAsAdmin(page: Page): Promise<boolean> {
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(3000)

  const inputs = page.locator('input')
  if (await inputs.count() < 2) return false

  await inputs.nth(0).fill('admin')
  await inputs.nth(1).fill('')

  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i })
  if (await loginButton.count() === 0) return false

  await loginButton.click()
  await page.waitForTimeout(5000)

  const url = page.url()
  // Login should redirect away from '/' to another page
  return !url.endsWith('/') || url.includes('/kftl') || url.includes('/rykv') || url.includes('/mi')
}
