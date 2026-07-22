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
 *
 * 保存完了は固定sleepではなく実シグナルで待つ。
 * 成功時のみ clear() が走って textarea が空になり、エラー時は内容が残る
 * (use-kftl-view.ts の submit()/clear() を参照)。
 * エラーを期待する呼び出しでは expectSuccess: false を渡すこと。
 */
export async function submitKftlText(
  page: Page,
  text: string,
  options: { expectSuccess?: boolean } = {},
): Promise<void> {
  const { expectSuccess = true } = options
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
  if (expectSuccess) {
    await expect(textarea).toHaveValue('', { timeout: 30000 })
  } else {
    await page.waitForTimeout(2000)
  }
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
 * 一度読むだけでリトライしない。アサーションには使わず、
 * expectPageToContainText / waitForPageText を使うこと。
 */
export async function pageContainsText(page: Page, text: string): Promise<boolean> {
  const app = page.locator('#app')
  const content = await app.textContent()
  return content != null && content.includes(text)
}

/**
 * #app に text が現れるまで待って検証する (自動リトライ)。
 * 描画待ちの固定sleepに依存しないための土台。
 */
export async function expectPageToContainText(page: Page, text: string, timeout = 30000): Promise<void> {
  await expect(page.locator('#app')).toContainText(text, { timeout })
}

/**
 * #app から text が消えるまで待って検証する (削除確認用)。
 */
export async function expectPageNotToContainText(page: Page, text: string, timeout = 30000): Promise<void> {
  await expect(page.locator('#app')).not.toContainText(text, { timeout })
}

/**
 * text が現れれば true、timeoutまで待って現れなければ false。条件分岐用。
 */
export async function waitForPageText(page: Page, text: string, timeout = 15000): Promise<boolean> {
  try {
    await expect(page.locator('#app')).toContainText(text, { timeout })
    return true
  } catch {
    return false
  }
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
  // チュートリアルダイアログの閉じるボタンはアイコンのみ (mdi-close) でテキストを持たないため、
  // hasText だけでは掴めない。掴み損ねると iframe やチェックボックスが
  // 後続クリックのポインタイベントを奪ってテストが落ちる。
  // 閉じたことを確認するまでリトライする。
  for (let attempt = 0; attempt < 5; attempt++) {
    if (await floatingDialogs.count() === 0) {
      return
    }
    const closeBtn = floatingDialogs.first()
      .locator('button:has(.mdi-close), button:has-text("×"), button:has-text("閉じる"), button:has-text("close")')
      .first()
    if (await closeBtn.count() > 0) {
      await closeBtn.click({ force: true }).catch(() => { /* 閉じかけている最中は無視 */ })
    } else {
      await page.keyboard.press('Escape')
    }
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

/**
 * findKyouByText の待機つき版。
 * リストの描画が終わるまで待ってから locator を返す。
 * 並列実行時は描画が間に合わずに count 0 になることがあるため、
 * 「対象が見つかっていること」を前提にするテストはこちらを使う。
 */
export async function waitForKyouByText(page: Page, text: string, timeout = 30000) {
  const record = findKyouByText(page, text)
  await record.waitFor({ state: 'visible', timeout })
  return record
}
