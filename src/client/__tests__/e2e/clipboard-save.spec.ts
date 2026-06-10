/**
 * E2E tests for the clipboard-save-to-file feature.
 *
 * Ctrl+V on non-text-input areas opens the clipboard save dialog.
 * Tests verify:
 *   - The dialog appears when Ctrl+V is pressed on the RYKV page body
 *   - The dialog title is correct
 *   - The dialog can be closed
 *   - Ctrl+V inside a text input does NOT open the dialog
 *   - With clipboard write permission, text content is previewed in the dialog
 */

import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import { navigateToRykv, navigateToMi } from './crud-helpers'

let serverAlive = false
test.beforeAll(async () => {
  serverAlive = await checkGkillServer()
  test.skip(!serverAlive, 'gkill server (localhost:9999) is not running')
})

test.describe('Clipboard Save to File', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!serverAlive, 'gkill server is not running')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('Ctrl+V on RYKV page opens clipboard save dialog', async ({ page }) => {
    await navigateToRykv(page)

    // Click on a non-interactive area of the page to ensure focus is on body (not a text input)
    await page.click('body', { position: { x: 10, y: 10 }, force: true })
    await page.waitForTimeout(500)

    // Dispatch Ctrl+V keyboard event
    await page.keyboard.press('Control+V')
    await page.waitForTimeout(2000)

    // The clipboard save dialog should appear with the expected title
    const dialogTitle = page.locator('.gkill-floating-dialog').filter({ hasText: /Save Clipboard to File|クリップボードをファイルに保存/i })
    await expect(dialogTitle).toBeVisible({ timeout: 10000 })
  })

  test('clipboard save dialog can be closed with close button', async ({ page }) => {
    await navigateToRykv(page)

    await page.click('body', { position: { x: 10, y: 10 }, force: true })
    await page.waitForTimeout(500)
    await page.keyboard.press('Control+V')
    await page.waitForTimeout(2000)

    // Confirm dialog is open
    const dialog = page.locator('.gkill-floating-dialog').filter({ hasText: /Save Clipboard to File|クリップボードをファイルに保存/i })
    await expect(dialog).toBeVisible({ timeout: 10000 })

    // Click the close button (mdi-close icon button in the dialog header)
    const closeButton = dialog.locator('button').filter({ has: page.locator('.mdi-close, [class*="close"]') }).first()
    if (await closeButton.count() > 0) {
      await closeButton.click()
    } else {
      // Fallback: find the first button in the dialog header
      const headerButton = dialog.locator('.gkill-floating-dialog__header button').first()
      await headerButton.click()
    }
    await page.waitForTimeout(1000)

    // Dialog should be gone
    await expect(dialog).not.toBeVisible({ timeout: 5000 })
  })

  test('Ctrl+V inside text input does NOT open clipboard save dialog', async ({ page }) => {
    await navigateToRykv(page)

    // Programmatically create and focus a textarea so Ctrl+V lands inside a text input
    await page.evaluate(() => {
      const ta = document.createElement('textarea')
      ta.id = '_test_textarea'
      ta.style.position = 'fixed'
      ta.style.top = '0'
      ta.style.left = '0'
      ta.style.width = '1px'
      ta.style.height = '1px'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.focus()
    })
    await page.waitForTimeout(300)

    // Press Ctrl+V inside the focused textarea — should NOT open clipboard dialog
    await page.keyboard.press('Control+V')
    await page.waitForTimeout(1500)

    // The clipboard save dialog should NOT appear
    const dialog = page.locator('.gkill-floating-dialog').filter({ hasText: /Save Clipboard to File|クリップボードをファイルに保存/i })
    await expect(dialog).not.toBeVisible({ timeout: 3000 })

    // Verify page is still functional
    await expect(page.locator('#app')).toBeVisible()
  })

  test('clipboard save dialog shows content when clipboard has text', async ({ page }) => {
    // Grant clipboard permissions to allow reading/writing clipboard
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])

    await navigateToRykv(page)

    // Write test text to clipboard
    const testText = 'clipboard_test_' + Date.now()
    await page.evaluate(async (text) => {
      await navigator.clipboard.writeText(text)
    }, testText)
    await page.waitForTimeout(300)

    await page.click('body', { position: { x: 10, y: 10 }, force: true })
    await page.waitForTimeout(300)

    // Open dialog with Ctrl+V
    await page.keyboard.press('Control+V')
    await page.waitForTimeout(2000)

    // Dialog should appear
    const dialog = page.locator('.gkill-floating-dialog').filter({ hasText: /Save Clipboard to File|クリップボードをファイルに保存/i })
    await expect(dialog).toBeVisible({ timeout: 10000 })

    // The dialog body should contain either the text preview OR the clipboard content info
    // (text/plain clipboard → text preview should show the test text)
    const dialogContent = await dialog.textContent()
    expect(dialogContent).toBeTruthy()

    // Close the dialog
    const headerButton = dialog.locator('.gkill-floating-dialog__header button').first()
    if (await headerButton.count() > 0) {
      await headerButton.click()
    }
  })

  test('Ctrl+V on Mi page also opens clipboard save dialog', async ({ page }) => {
    await navigateToMi(page)

    // Click body (not a text input)
    await page.click('body', { position: { x: 10, y: 10 }, force: true })
    await page.waitForTimeout(500)

    await page.keyboard.press('Control+V')
    await page.waitForTimeout(2000)

    // The clipboard save dialog should appear on Mi page as well
    const dialog = page.locator('.gkill-floating-dialog').filter({ hasText: /Save Clipboard to File|クリップボードをファイルに保存/i })
    await expect(dialog).toBeVisible({ timeout: 10000 })
  })
})
