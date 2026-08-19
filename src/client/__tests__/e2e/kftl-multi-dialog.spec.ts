import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { clickContextMenuItem, makeUniqueLabel, navigateToRykv, MENU } from './crud-helpers'

/**
 * メモ帳ウィンドウは複数枚開ける。
 *
 * タブの一覧と中身は共有シングルトン（use-kftl-tabs.ts）で、
 * 「いま映しているタブ」だけがウィンドウごと（use-kftl-view.ts）。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const KFTL_DIALOG = '.gkill-floating-dialog.kftl_dialog'
const TEXT_AREA = 'textarea.kftl_text_area'
const TAB = '.kftl_tab'

/**
 * ＋メニューから「メモ帳」を選ぶ。呼ぶたびにウィンドウが1枚増える。
 *
 * 共通の `clickFabButton` は先に `dismissFloatingDialogs()` を呼ぶので使えない
 * ―― すでに開いているメモ帳ウィンドウを閉じてしまい、いつまでも1枚のままになる
 */
async function openKftlWindow(page: Page): Promise<void> {
  const before = await page.locator(KFTL_DIALOG).count()

  const fab = page.locator('.position-fixed button, .position-fixed .v-btn').first()
  await expect(fab, 'FAB(＋ボタン)が見つからない').toBeVisible({ timeout: 15000 })

  // **押して、開くまで押し直す。**
  // Vuetify の useActivator は「閉じた直後 50ms のあいだアクティベータのクリックを
  // 黙って捨てる」(reopenLock)。2枚目を開くクリックがその窓に入ると
  // メニューは開かず、以降なにも起きないまま待ち続けることになる。
  //
  // 「前のメニューが閉じきるのを待ってから押す」では塞げない ――
  // reopenLock を立てるのは isActive が false になった瞬間の watch なので、
  // **aria-expanded が "false" になる時刻はその 50ms の始まりそのもの**。
  // 閉じたことを確かめてから押すほど、むしろ確実に窓の中へ入る。
  // 開いたか(aria-expanded="true")だけを見て、開くまで押し直すのが唯一の塞ぎ方。
  // aria-expanded は VMenu が必ずアクティベータへ出しているので、製品側に印は要らない
  await expect(async () => {
    await fab.click({ force: true })
    await expect(fab, 'FABメニューが開かない').toHaveAttribute('aria-expanded', 'true', { timeout: 2000 })
  }).toPass({ timeout: 30000 })

  await clickContextMenuItem(page, MENU.kftl)
  await expect(page.locator(KFTL_DIALOG)).toHaveCount(before + 1)
  await expect(page.locator(`${KFTL_DIALOG} ${TEXT_AREA}`)).toHaveCount(before + 1)
}

function windowTextArea(page: Page, index: number) {
  return page.locator(KFTL_DIALOG).nth(index).locator(TEXT_AREA)
}

async function zIndexOf(page: Page, index: number): Promise<number> {
  const value = await page.locator(KFTL_DIALOG).nth(index).evaluate((el) => (el as HTMLElement).style.zIndex)
  return Number(value)
}

/** ヘッダを掴んで前面へ出す */
async function bringToFront(page: Page, index: number): Promise<void> {
  await page.locator(KFTL_DIALOG).nth(index).locator('.gkill-floating-dialog__header').click()
}

test.describe('KFTL Multi Dialog', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
    await navigateToRykv(page)
  })

  test('＋メニューから何度でも開けて、ウィンドウがずれて重なる', async ({ page }) => {
    await openKftlWindow(page)
    await openKftlWindow(page)

    const first_box = await page.locator(KFTL_DIALOG).nth(0).boundingBox()
    const second_box = await page.locator(KFTL_DIALOG).nth(1).boundingBox()
    expect(first_box).not.toBeNull()
    expect(second_box).not.toBeNull()
    expect(second_box!.x, '2枚目が1枚目に完全に重なっている').not.toBe(first_box!.x)
  })

  test('ウィンドウごとに別のタブを選べる', async ({ page }) => {
    await openKftlWindow(page)
    const first_label = makeUniqueLabel('win_first')
    await windowTextArea(page, 0).fill(first_label)

    // 1枚目でタブを増やしてから2枚目を開く
    await page.locator(KFTL_DIALOG).nth(0).locator('.kftl_tab_add').click()
    await expect(page.locator(KFTL_DIALOG).nth(0).locator(TAB)).toHaveCount(2)
    const second_label = makeUniqueLabel('win_second')
    await windowTextArea(page, 0).fill(second_label)

    await openKftlWindow(page)
    // 2枚目は直近のタブを映す。1枚目のタブへ切り替えると別の下書きが出る
    await page.locator(KFTL_DIALOG).nth(1).locator(TAB).first().click()

    await expect(windowTextArea(page, 1)).toHaveValue(first_label)
    await expect(windowTextArea(page, 0)).toHaveValue(second_label)
  })

  test('同じタブを映していれば入力が同期される', async ({ page }) => {
    await openKftlWindow(page)
    await openKftlWindow(page)

    const label = makeUniqueLabel('sync')
    await windowTextArea(page, 0).fill(label)

    await expect(windowTextArea(page, 1)).toHaveValue(label)
  })

  test('クリックしたウィンドウが前面に来る', async ({ page }) => {
    await openKftlWindow(page)
    await openKftlWindow(page)

    // 一発読みにしないこと。z-index は enter_z_order が
    // watch(container_ref, …, { flush: "post" }) の中で走るので、
    // **最初の描画では indexOf が -1 で 1100**（＝2枚とも同じ値）に見える窓がある
    await expect.poll(async () => await zIndexOf(page, 1) > await zIndexOf(page, 0)).toBe(true)

    await bringToFront(page, 0)

    await expect.poll(async () => await zIndexOf(page, 0) > await zIndexOf(page, 1)).toBe(true)
  })

  // 積んだ順の末尾ではなく、見た目の最前面を閉じる
  test('ブラウザバックは最前面のウィンドウを閉じる', async ({ page }) => {
    await openKftlWindow(page)
    await openKftlWindow(page)

    const second_box = await page.locator(KFTL_DIALOG).nth(1).boundingBox()
    expect(second_box).not.toBeNull()

    // 後から開いた2枚目が前面。1枚目をクリックして前面へ入れ替える
    await bringToFront(page, 0)
    await expect.poll(async () => await zIndexOf(page, 0) > await zIndexOf(page, 1)).toBe(true)

    await page.goBack()

    await expect(page.locator(KFTL_DIALOG)).toHaveCount(1)
    const remaining_box = await page.locator(KFTL_DIALOG).boundingBox()
    expect(Math.round(remaining_box!.x), '前面ではないほうが閉じている').toBe(Math.round(second_box!.x))
  })

  // 閉じるボタンはヘッダに限定する。タブの × も mdi-close なので、
  // ダイアログ全体から探すと2つ当たって strict mode 違反になる
  test('片方を閉じてももう片方は残る', async ({ page }) => {
    await openKftlWindow(page)
    await openKftlWindow(page)

    await page.locator(KFTL_DIALOG).nth(1).locator('.gkill-floating-dialog__header button:has(.mdi-close)').click()
    await expect(page.locator(KFTL_DIALOG)).toHaveCount(1)

    await page.locator(KFTL_DIALOG).nth(0).locator('.gkill-floating-dialog__header button:has(.mdi-close)').click()
    await expect(page.locator(KFTL_DIALOG)).toHaveCount(0)
  })
})
