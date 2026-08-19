import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, waitForKyouByText,
} from './crud-helpers'

/**
 * ダイアログ履歴の不変条件テスト (案C: history 駆動クローズ)
 *
 * 検証する不変条件:
 * - ×ボタン/Escape で閉じても、ブラウザバックのスタックに使用済み
 *   エントリが残らない (全ダイアログを閉じた後、戻る1回でページを離れる)
 * - ブラウザバック(①)と×ボタン(②)を混ぜて閉じても同様
 */

let apiReachable = false
test.beforeAll(async () => {
  apiReachable = await checkGkillServer()
  test.skip(!apiReachable, 'gkill server is not running')
})

/**
 * Helper: open history dialog for a record found by text.
 * (view-history.spec.ts と同じ操作。RykvDialogHost 管理のダイアログが開く)
 */
async function openHistoryFor(page: Page, text: string): Promise<boolean> {
  const dialogs = page.locator('.gkill-floating-dialog')
  const before = await dialogs.count()

  // 並列実行だとリスト描画が間に合わないことがあるので、対象が出るまで待つ
  const record = await waitForKyouByText(page, text)
  await record.click({ button: 'right', force: true })

  const historyMenuItem = page.locator('.v-list-item, [role="menuitem"]').filter({ hasText: /履歴|histor/i }).first()
  await expect(historyMenuItem).toBeVisible({ timeout: 15000 })
  await historyMenuItem.click()

  // ダイアログが1枚増えるまで待つ
  await expect(dialogs).toHaveCount(before + 1, { timeout: 15000 })
  return true
}

/**
 * Helper: 最上位のフローティングダイアログを×ボタンで閉じる。
 */
async function close_top_dialogWithX(page: Page): Promise<boolean> {
  const dialogs = page.locator('.gkill-floating-dialog')
  const n = await dialogs.count()
  if (n === 0) return false
  const closeBtn = dialogs.nth(n - 1).locator('.gkill-floating-dialog__header button:has(.mdi-close)').first()
  if (await closeBtn.count() === 0) return false
  await closeBtn.click()
  // popstate 経由の非同期クローズを待つ
  await expect(dialogs).toHaveCount(n - 1, { timeout: 15000 })
  return true
}

test.describe('Dialog History Invariants', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // ×クローズ後、戻る1回でページを離れられる (スタック残無し)
  test('close by X button leaves no back-stack entry', async ({ page }) => {
    const label = makeUniqueLabel('dlg_hist_x')
    await submitKftlText(page, label) // history: [..., /kftl]
    await navigateToRykv(page) //        history: [..., /kftl, /rykv]

    const opened = await openHistoryFor(page, label)
    expect(opened).toBe(true)
    await expect(page.locator('.gkill-floating-dialog').last()).toBeVisible()

    const closed = await close_top_dialogWithX(page)
    expect(closed).toBe(true)
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 5000 })
    // ダイアログクローズでページ遷移していないこと
    expect(page.url()).toContain('/rykv')

    // 戻る1回で /rykv を離れて /kftl に戻れること (エントリが残っていると /rykv のまま)
    await page.goBack({ waitUntil: 'domcontentloaded' })
    await page.waitForURL(/\/kftl/, { timeout: 15000 })
  })

  // ブラウザバックでダイアログが閉じ、ページには留まる
  test('browser back closes dialog and stays on page', async ({ page }) => {
    const label = makeUniqueLabel('dlg_hist_back')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    const opened = await openHistoryFor(page, label)
    expect(opened).toBe(true)
    await expect(page.locator('.gkill-floating-dialog').last()).toBeVisible()

    await page.goBack({ waitUntil: 'commit' })
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 15000 })
    expect(page.url()).toContain('/rykv')

    // もう1回戻ると /kftl へ
    await page.goBack({ waitUntil: 'domcontentloaded' })
    await page.waitForURL(/\/kftl/, { timeout: 15000 })
  })

  // ①バック ②× ③Escape を混ぜて開閉してもスタックが残らない (報告された再現手順)
  test('mixing back / X / Escape closes leaves clean back-stack', async ({ page }) => {
    const label = makeUniqueLabel('dlg_hist_mix')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // 1回目: ブラウザバックで閉じる (①)
    let opened = await openHistoryFor(page, label)
    expect(opened).toBe(true)
    await page.goBack({ waitUntil: 'commit' })
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 15000 })
    expect(page.url()).toContain('/rykv')

    // 2回目: ×ボタンで閉じる (②)
    opened = await openHistoryFor(page, label)
    expect(opened).toBe(true)
    const closed = await close_top_dialogWithX(page)
    expect(closed).toBe(true)
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 5000 })
    expect(page.url()).toContain('/rykv')

    // 3回目: Escape で閉じる
    opened = await openHistoryFor(page, label)
    expect(opened).toBe(true)
    await page.keyboard.press('Escape')
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(0, { timeout: 15000 })
    expect(page.url()).toContain('/rykv')

    // 混在開閉の後でも、戻る1回で /kftl に戻れること
    await page.goBack({ waitUntil: 'domcontentloaded' })
    await page.waitForURL(/\/kftl/, { timeout: 15000 })
  })

  // 複数ダイアログを開いたまま APP_BAR プルダウンで別ページへ遷移できること
  // (reset_dialog_history の popstate 会計バグ回帰テスト: go(-N) は popstate 1回)
  test('app bar pulldown navigates away while multiple dialogs are open', async ({ page }) => {
    const label1 = makeUniqueLabel('dlg_nav_1')
    const label2 = makeUniqueLabel('dlg_nav_2')
    await submitKftlText(page, label1)
    await submitKftlText(page, label2)
    await navigateToRykv(page)

    // ダイアログを1枚ずつ確実に開く (2枚目を1枚目の上に開く競合を避けるため、
    // 各段階で枚数の確定を待ってから次へ進める)
    expect(await openHistoryFor(page, label1)).toBe(true)
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(1, { timeout: 15000 })
    expect(await openHistoryFor(page, label2)).toBe(true)
    await expect(page.locator('.gkill-floating-dialog')).toHaveCount(2, { timeout: 15000 })

    // APP_BAR タイトルのプルダウンから タスク(mi) を選ぶ
    await page.locator('.v-toolbar-title').first().click()
    // 固定sleepではなくメニューの項目が出るのを待つ（下の toBeVisible が自動でリトライする）
    const item = page.locator('.v-overlay .v-list-item-title').filter({ hasText: /タスク|task/i }).first()
    await expect(item, '画面切替のメニューが出ない').toBeVisible({ timeout: 15000 })
    await item.click()

    // 遷移が実行されること (会計バグがあると /rykv のまま止まる)
    await page.waitForURL(/\/mi/, { timeout: 15000 })
  })

  /**
   * マニュアル（ヘルプ）は本文が丸ごと iframe なので、
   * 本文で起きた pointerdown / focusin は**親のDOMへ一切伝わらない**。
   * ヘッダを掴んだときしか前面化しないと、他のウィンドウの下に潜ったまま
   * マニュアルを読み進められなくなる。
   */
  test('マニュアルは本文をクリックしても前面に来る', async ({ page }) => {
    await navigateToRykv(page)

    // マニュアル → メモ帳 の順に開く。あとから開いたほうが前
    await page.locator('.v-app-bar button:has(.mdi-help-circle-outline)').first().click()
    const manual = page.locator('.gkill-floating-dialog.help-dialog')
    await expect(manual, 'マニュアルが開かない').toBeVisible({ timeout: 15000 })

    const fab = page.locator('.position-fixed button, .position-fixed .v-btn').first()
    await fab.click({ force: true })
    const kftl_item = page.locator('.v-list-item').filter({ hasText: /^\s*メモ帳\s*$/ }).first()
    await expect(kftl_item).toBeVisible({ timeout: 15000 })
    await kftl_item.click()
    const kftl = page.locator('.gkill-floating-dialog.kftl_dialog')
    await expect(kftl, 'メモ帳が開かない').toBeVisible({ timeout: 30000 })

    const z_of = async (locator: typeof manual): Promise<number> =>
      Number(await locator.evaluate((el) => (el as HTMLElement).style.zIndex))

    expect(await z_of(manual), 'あとから開いたメモ帳が前に来ていない')
      .toBeLessThan(await z_of(kftl))

    // マニュアル本文（iframe の中）をクリックする
    const manual_frame = manual.frameLocator('iframe')
    await manual_frame.locator('body').click({ position: { x: 40, y: 40 } })

    await expect
      .poll(async () => (await z_of(manual)) > (await z_of(kftl)), {
        message: 'マニュアルの本文をクリックしても前面に来ない',
        timeout: 15000,
      })
      .toBe(true)
  })
})
