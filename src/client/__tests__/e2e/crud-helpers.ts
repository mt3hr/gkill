import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'

/**
 * Generate a unique label for test data using timestamp.
 */
export function makeUniqueLabel(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

/**
 * Submit KFTL text via the KFTL page.
 * Navigates to /kftl, fills textarea, and clicks save.
 */
export async function submitKftlText(page: Page, text: string): Promise<void> {
  await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  // Use id selector for the KFTL textarea
  const textarea = page.locator('#kftl_text_area')
  await expect(textarea).toBeVisible({ timeout: 90000 })
  // Wait for the save button to become enabled (application_config loaded)
  const saveButton = page.locator('button').filter({ hasText: /保存|送信|submit|save/i }).first()
  await expect(saveButton).toBeEnabled({ timeout: 30000 })
  await textarea.fill(text)
  await page.waitForTimeout(500)
  await saveButton.click()
  await page.waitForTimeout(2000)
}

/**
 * Navigate to RYKV page and wait for it to load.
 */
export async function navigateToRykv(page: Page): Promise<void> {
  await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(3000)
  await dismissFloatingDialogs(page)
}

/**
 * Navigate to Mi board page and wait for it to load.
 */
export async function navigateToMi(page: Page): Promise<void> {
  await page.goto('/mi', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(3000)
  await dismissFloatingDialogs(page)
}

/**
 * Navigate to Plaing (TimeIs) page and wait for it to load.
 */
export async function navigateToPlaing(page: Page): Promise<void> {
  await page.goto('/plaing', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(3000)
  await dismissFloatingDialogs(page)
}

/**
 * Navigate to Settings page and wait for it to load.
 */
export async function navigateToSettings(page: Page): Promise<void> {
  await page.goto('/saihate', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await page.waitForTimeout(2000)
  await dismissFloatingDialogs(page)
}

/**
 * Check if the page contains the given text anywhere in #app.
 */
export async function pageContainsText(page: Page, text: string): Promise<boolean> {
  const app = page.locator('#app')
  const content = await app.textContent()
  return content != null && content.includes(text)
}

/**
 * Right-click on an element matching the selector to open context menu.
 */
export async function openContextMenu(page: Page, selector: string): Promise<void> {
  await dismissFloatingDialogs(page)
  const element = page.locator(selector).first()
  await element.click({ button: 'right', force: true })
  await page.waitForTimeout(1000)
}

/**
 * Click a context menu item by its text label.
 */
export async function clickContextMenuItem(page: Page, label: RegExp | string): Promise<void> {
  const menuItem = typeof label === 'string'
    ? page.locator('.v-list-item, .v-menu .v-btn, [role="menuitem"]').filter({ hasText: label })
    : page.locator('.v-list-item, .v-menu .v-btn, [role="menuitem"]').filter({ hasText: label })
  await menuItem.first().click()
  await page.waitForTimeout(1000)
}

/**
 * Click a button in a dialog (e.g., save or delete confirm).
 */
export async function clickDialogButton(page: Page, label: RegExp | string): Promise<void> {
  const button = page.locator('.gkill-floating-dialog button, .v-dialog button, .v-card button').filter({ hasText: label })
  await button.first().click()
  await page.waitForTimeout(2000)
}

/**
 * Confirm a delete dialog by clicking the delete/confirm button.
 */
export async function confirmDelete(page: Page): Promise<void> {
  await clickDialogButton(page, /削除|delete/i)
}

/**
 * Click the FAB (+) button on rykv page to open add menu.
 * The FAB is a v-btn with mdi-plus icon inside a position-fixed v-avatar.
 */
export async function clickFabButton(page: Page): Promise<void> {
  // Close any floating dialogs (tutorial dialog) that may intercept clicks
  await dismissFloatingDialogs(page)

  // The FAB is: v-avatar.position-fixed > v-menu > v-btn[icon="mdi-plus"]
  const fab = page.locator('.position-fixed button, .position-fixed .v-btn').first()
  if (await fab.count() > 0) {
    await fab.click({ force: true })
  } else {
    // Fallback: look for a button with mdi-plus icon
    const plusBtn = page.locator('.mdi-plus').first()
    if (await plusBtn.count() > 0) {
      await plusBtn.click({ force: true })
    } else {
      const addBtn = page.locator('button').filter({ hasText: /\+|追加|add/i }).first()
      await addBtn.click()
    }
  }
  await page.waitForTimeout(1000)
}

/**
 * Dismiss any floating dialogs (e.g., tutorial dialog) that may intercept pointer events.
 */
export async function dismissFloatingDialogs(page: Page): Promise<void> {
  const floatingDialogs = page.locator('.gkill-floating-dialog')
  const count = await floatingDialogs.count()
  for (let i = 0; i < count; i++) {
    // Try to close the dialog by clicking a close button or pressing Escape
    const closeBtn = floatingDialogs.nth(i).locator('button').filter({ hasText: /×|閉じる|close/i }).first()
    if (await closeBtn.count() > 0) {
      await closeBtn.click()
      await page.waitForTimeout(500)
    }
  }
  // Also try pressing Escape to close any modal
  if (count > 0) {
    await page.keyboard.press('Escape')
    await page.waitForTimeout(500)
  }
}

/**
 * Find and click a kyou item on rykv page that contains the given text.
 * Returns the locator for the found item.
 */
export function findKyouByText(page: Page, text: string) {
  return page.locator('#app').locator(`text=${text}`).first()
}
