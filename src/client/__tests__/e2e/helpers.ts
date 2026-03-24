import type { Page } from '@playwright/test'

/**
 * Login as the E2E test user.
 * With Playwright's setup project, the storageState (cookies) is already loaded.
 * This function navigates to an authenticated page to confirm the session is active.
 * Returns true if login succeeded (URL is an authenticated page), false otherwise.
 */
export async function loginAsAdmin(page: Page): Promise<boolean> {
  await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  const url = page.url()
  return url.includes('/kftl') || url.includes('/rykv') || url.includes('/mi')
}
