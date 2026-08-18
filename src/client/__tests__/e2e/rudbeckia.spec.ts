import { test, expect, type Locator, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { clickContextMenuItem, makeUniqueLabel, MENU, navigateToRudbeckia, openRudbeckiaFabMenu } from './crud-helpers'
import { RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND } from '@/pages/views/rudbeckia-page-kind'

/**
 * ポート（開発コード rudbeckia）。
 *
 * 背景とFABだけの1画面で、ライフログビュー / タスク / 実行中 / ダッシュボードを
 * フローティングウィンドウとして開ける。
 *
 * ここで一番効く検証は「ホストしたビューのアプリバーとサイドバーが
 * ウィンドウの中に収まっているか」。Vuetify の入れ子レイアウト
 * (vuetify/lib/composables/layout.js:211) が効いていないと position: fixed のまま
 * 画面最上部へ飛ぶので、ダイアログの矩形からはみ出す。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const PAGE_DIALOG = '.gkill-floating-dialog.rudbeckia-page-dialog'

const SCREEN_MENU = {
  rykv: /^\s*ライフログビュー\s*$/,
  mi: /^\s*タスク\s*$/,
  plaing: /^\s*実行中\s*$/,
  dashboard: /^\s*ダッシュボード\s*$/,
}

/** ＋メニューの「画面」グループから1枚開く */
async function openScreenWindow(page: Page, label: RegExp): Promise<void> {
  const before = await page.locator(PAGE_DIALOG).count()
  await openRudbeckiaFabMenu(page)
  await clickContextMenuItem(page, label)
  await expect(page.locator(PAGE_DIALOG)).toHaveCount(before + 1)
}

/** ＋メニューの「記録」グループのメモ帳から1件足す */
async function addRecordFromFab(page: Page, label: string): Promise<void> {
  await openRudbeckiaFabMenu(page)
  await clickContextMenuItem(page, MENU.kftl)

  const textarea = page.locator('textarea.kftl_text_area').first()
  await expect(textarea, 'メモ帳が開かない').toBeVisible({ timeout: 30000 })
  await textarea.fill(label)
  await expect(textarea).toHaveValue(label, { timeout: 15000 })
  const save_button = page.locator('button').filter({ hasText: /保存/ }).first()
  await expect(save_button).toBeEnabled({ timeout: 30000 })
  await save_button.click()
  await expect(textarea, '保存が完了しない').toHaveValue('', { timeout: 60000 })
}

/** 列の初期検索が決着するまで待つ。飛行中に足すと結果の書き戻しで消える */
async function waitForWindowReady(dialog: Locator): Promise<void> {
  await expect(
    dialog.locator('[data-gkill-view-ready="true"]'),
    '列の準備が終わらない',
  ).toBeAttached({ timeout: 60000 })
}

test.describe('ポート', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
    await navigateToRudbeckia(page)
  })

  test('4つの画面をウィンドウとして開ける', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)
    await openScreenWindow(page, SCREEN_MENU.mi)
    await openScreenWindow(page, SCREEN_MENU.plaing)
    await openScreenWindow(page, SCREEN_MENU.dashboard)

    await expect(page.locator(PAGE_DIALOG), '4枚そろっていない').toHaveCount(4)
  })

  // 種類ごとの slot_index は保存キーを分けるためのもので、4種類とも 0 になる。
  // ずらす量をそれで決めると4枚が完全に重なって1枚に見える
  test('複数のウィンドウは重ならないようにずれて開く', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)
    await openScreenWindow(page, SCREEN_MENU.mi)

    const first = await page.locator(PAGE_DIALOG).nth(0).boundingBox()
    const second = await page.locator(PAGE_DIALOG).nth(1).boundingBox()
    expect(first, '1枚目の矩形が取れない').not.toBeNull()
    expect(second, '2枚目の矩形が取れない').not.toBeNull()

    const moved = Math.abs(second!.x - first!.x) + Math.abs(second!.y - first!.y)
    expect(moved, '2枚目が1枚目とぴったり重なっている').toBeGreaterThan(0)
  })

  /**
   * 同じ画面を並べられる。列の検索条件とスクロール位置の保存キーは
   * `slot_index` 由来の枝番で分けてあるので、2枚目が1枚目を上書きしない
   * （slot 0 は従来キーそのまま＝単独ページと同じ列を引き継ぐ）。
   */
  test('同じ画面を2枚並べられる', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)
    await openScreenWindow(page, SCREEN_MENU.rykv)

    await expect(page.locator(PAGE_DIALOG), '同じ画面が2枚開けない').toHaveCount(2)
  })

  // ライフログビュー1枚で数十万件の配列を持ちうるので、無制限にはしない。
  // 上限では新しく開かず、その種類の最前面へフォーカスを移す
  test('同じ画面は上限を超えて増えない', async ({ page }) => {
    for (let i = 0; i < RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND; i++) {
      await openScreenWindow(page, SCREEN_MENU.rykv)
    }

    await openRudbeckiaFabMenu(page)
    await clickContextMenuItem(page, SCREEN_MENU.rykv)

    await expect(
      page.locator(PAGE_DIALOG),
      '上限を超えてウィンドウが増えている',
    ).toHaveCount(RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND)
  })

  test('ホストしたビューのアプリバーがウィンドウの中に収まる', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)

    const dialog = page.locator(PAGE_DIALOG).first()
    const app_bar = dialog.locator('.v-app-bar').first()
    await expect(app_bar, 'ホストしたビューのアプリバーが見つからない').toBeVisible({ timeout: 30000 })

    const dialog_box = await dialog.boundingBox()
    const app_bar_box = await app_bar.boundingBox()
    expect(dialog_box, 'ウィンドウの矩形が取れない').not.toBeNull()
    expect(app_bar_box, 'アプリバーの矩形が取れない').not.toBeNull()

    // 入れ子レイアウトが効いていないと position: fixed のまま画面最上部(y≈0)へ飛ぶ。
    // 収まっていれば必ずウィンドウの上端より下に居る
    expect(
      app_bar_box!.y,
      'アプリバーがウィンドウの外（画面最上部）に描かれている＝入れ子レイアウトが効いていない',
    ).toBeGreaterThanOrEqual(dialog_box!.y - 1)
    expect(
      app_bar_box!.x,
      'アプリバーがウィンドウの左外にはみ出している',
    ).toBeGreaterThanOrEqual(dialog_box!.x - 1)
    expect(
      app_bar_box!.width,
      'アプリバーがウィンドウより広い（画面幅で描かれている）',
    ).toBeLessThanOrEqual(dialog_box!.width + 1)
  })

  test('ホストしたビューは自前のFABを出さない', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)

    const dialog = page.locator(PAGE_DIALOG).first()
    await expect(
      dialog.locator('.position-fixed'),
      'ウィンドウの中にFABが残っている（ポートのFABと重なる）',
    ).toHaveCount(0)
  })

  // FABはポートで唯一の操作導線。ウィンドウ(z-index 1100+)に覆われると
  // 記録の追加も画面の追加もできなくなる
  test('ウィンドウを開いてもFABは押せる', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)
    await openScreenWindow(page, SCREEN_MENU.dashboard)

    const fab = page.locator('.position-fixed-rudbeckia button, .position-fixed-rudbeckia .v-btn').first()
    // force を付けない = 実際にクリックが届くことの確認
    await fab.click()
    await expect(
      page.locator('.v-menu .v-list-item').first(),
      'ウィンドウに覆われてFABが押せない',
    ).toBeVisible({ timeout: 15000 })
  })

  /**
   * リサイズハンドルはダイアログの右下隅に後付けされる `<div>`。
   * ホストしたビューは Vuetify の入れ子レイアウトで z-index 900 台を占めるので、
   * ハンドルの z-index が小さいと**隅がビューに覆われてつまめない**。
   */
  for (const [kind, label] of Object.entries(SCREEN_MENU)) {
    test(`ウィンドウの右下をつまんで大きさを変えられる（${kind}）`, async ({ page }) => {
      await openScreenWindow(page, label)

      const dialog = page.locator(PAGE_DIALOG).first()
      const before = await dialog.boundingBox()
      expect(before, 'ウィンドウの矩形が取れない').not.toBeNull()

      const handle = dialog.locator('.gkill-floating-dialog__resize-handle')
      await expect(handle, 'リサイズハンドルが無い').toHaveCount(1)
      const handle_box = await handle.boundingBox()
      expect(handle_box, 'リサイズハンドルの矩形が取れない').not.toBeNull()

      const center = { x: handle_box!.x + handle_box!.width / 2, y: handle_box!.y + handle_box!.height / 2 }
      const topmost = await page.evaluate((point) => {
        const el = document.elementFromPoint(point.x, point.y)
        return el ? String(el.className) : ''
      }, center)
      expect(topmost, 'リサイズハンドルがホストした画面に覆われている').toContain('resize-handle')

      await page.mouse.move(center.x, center.y)
      await page.mouse.down()
      await page.mouse.move(center.x - 200, center.y - 150, { steps: 10 })
      await page.mouse.up()

      const after = await dialog.boundingBox()
      expect(after!.width, '幅が縮んでいない').toBeLessThan(before!.width - 50)
      expect(after!.height, '高さが縮んでいない').toBeLessThan(before!.height - 50)
    })
  }

  test('×で閉じられる', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.mi)

    const dialog = page.locator(PAGE_DIALOG).first()
    await dialog.locator('.gkill-floating-dialog__header button:has(.mdi-close)').first().click()

    await expect(page.locator(PAGE_DIALOG), '×で閉じられない').toHaveCount(0)
  })

  /**
   * ポートの本題。並べた画面のあいだで変更が伝わること。
   *
   * ポートのFABから追加した記録は `rudbeckia-page` を発生元として配られるので、
   * どのウィンドウも「自分が出したもの」とは見なさず受け取る。
   */
  test('FABから追加した記録が開いている画面へ出る', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)

    const dialog = page.locator(PAGE_DIALOG).first()
    await waitForWindowReady(dialog)

    const label = makeUniqueLabel('ポート伝播')
    await addRecordFromFab(page, label)

    await expect(
      dialog.getByText(label, { exact: false }).first(),
      'ポートで追加した記録が開いている画面へ届いていない',
    ).toBeVisible({ timeout: 60000 })
  })

  /**
   * ダイアログの中身は「カード1枚」を前提にした App.vue の子孫セレクタ
   * `.gkill-floating-dialog__body .v-card { display:flex; overflow:auto }` が、
   * 一覧の各行が描く v-card（kmemo-view.vue など）まで巻き込むと、
   * 行が1つずつ独立したスクロール箱になって単独ページと形が変わる。
   * ここに載っているのは画面まるごとなので、中の v-card は Vuetify の既定へ戻す。
   */
  test('一覧の記録は行ごとにスクロールしない（単独ページと同じ）', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)

    const dialog = page.locator(PAGE_DIALOG).first()
    await waitForWindowReady(dialog)

    const label = makeUniqueLabel('ポート行スクロール')
    await addRecordFromFab(page, label)

    // 行の中身は型ごとのビュー（kmemo-view.vue など）が描く v-card
    const row_card = dialog.locator('.kyou_view_root .v-card').first()
    await expect(row_card, '一覧に行が出ない').toBeVisible({ timeout: 60000 })

    const overflow = await row_card.evaluate((el) => {
      const style = getComputedStyle(el)
      return { x: style.overflowX, y: style.overflowY }
    })
    expect(overflow.y, '行が縦スクロールできてしまっている').toBe('hidden')
    expect(overflow.x, '行が横スクロールできてしまっている').toBe('hidden')
  })

  test('Escapeで最前面のウィンドウが閉じる', async ({ page }) => {
    await openScreenWindow(page, SCREEN_MENU.rykv)
    await openScreenWindow(page, SCREEN_MENU.mi)

    await page.keyboard.press('Escape')

    await expect(page.locator(PAGE_DIALOG), 'Escapeで1枚だけ閉じない').toHaveCount(1)
  })
})
