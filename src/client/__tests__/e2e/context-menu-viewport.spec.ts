import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, waitForKyouByText,
} from './crud-helpers'

/**
 * コンテキストメニューが画面領域に収まること。
 *
 * 以前は各composableが `left: min(innerWidth - 130, x)` /
 * `top: min(max(50, innerHeight - (8 + 48 * 項目数)), y)` を25箇所にコピペしており、
 * メニューの実寸を一度も測っていなかった。今は座標を `<v-menu :target>` に渡し、
 * Vuetify が実寸を測って flip / shift する。
 *
 * **このテストの位置づけ**: 実測すると Kyou のコンテキストメニューは 79×270px で、
 * 見積り定数（幅130 / 1項目48px）より小さい。つまりこのメニューに関しては
 * 旧実装の定数がたまたま安全側で、はみ出していなかった。
 * 実際にはみ出していたのは項目数の見積りが実態より小さかった構成ツリー系
 * （`*-struct-context-menu`、5項目に対し `48 * 2`）で、こちらは設定ダイアログの
 * 中にあり e2e から開くのに別途導線が要る。
 *
 * よってこれは「旧実装で落ちる回帰テスト」ではなく、**新方式が満たすべき不変条件
 * （外接矩形がビューポート内）を固定する番人**である。メニューの項目や
 * ラベルが増えても収まり続けることを保証する。
 *
 * 横方向のケースは入れていない。記録の行は幅400pxの固定ブロックで、
 * 右端付近を右クリックしても `contextmenu` が届かず（幅1280/960/800すべてで確認）、
 * 行の右端を画面右端に寄せようと幅を420pxまで詰めると、今度は行のどこを
 * 右クリックしてもメニューが出なくなる。安定して開けない座標でのテストは
 * フレークにしかならないので、開ける条件で測れる縦方向だけを見る。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/** 表示中のコンテキストメニュー本体（v-list を載せている overlay content）。 */
function contextMenuContent(page: Page) {
  return page.locator('.v-overlay--active:has(.gkill_context_menu_list) .v-overlay__content').first()
}

/** メニューの外接矩形がビューポートに収まっていること。 */
async function expectMenuInsideViewport(page: Page): Promise<void> {
  const menu = contextMenuContent(page)
  await expect(menu, 'コンテキストメニューが開かない').toBeVisible({ timeout: 15000 })

  // 開いた直後は Vuetify が位置を確定する前の可能性があるので、
  // 収まるまでポーリングする（1pxはサブピクセル丸めの許容）
  await expect(async () => {
    const box = await menu.boundingBox()
    const viewport = page.viewportSize()
    expect(box, 'メニューの外接矩形が取れない').not.toBeNull()
    expect(viewport).not.toBeNull()
    expect(box!.x, '左にはみ出している').toBeGreaterThanOrEqual(-1)
    expect(box!.y, '上にはみ出している').toBeGreaterThanOrEqual(-1)
    expect(box!.x + box!.width, '右にはみ出している').toBeLessThanOrEqual(viewport!.width + 1)
    expect(box!.y + box!.height, '下にはみ出している').toBeLessThanOrEqual(viewport!.height + 1)
    // 幅0で「収まっている」ことにならないよう、実体があることも見る
    expect(box!.width, 'メニューの幅が0').toBeGreaterThan(0)
    expect(box!.height, 'メニューの高さが0').toBeGreaterThan(0)
  }).toPass({ timeout: 15000 })
}

test.describe('コンテキストメニューがビューポートに収まる', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('縦に狭い画面で開いてもメニューが下にはみ出さない', async ({ page }) => {
    const label = makeUniqueLabel('ctxmenu_bottom')
    await submitKftlText(page, label)

    // 記録は一覧の先頭に出るので、通常の画面高では下端に届かない。
    // ビューポートを縦に潰して、行より下にメニューの高さが残らない状況を作る。
    // メニューは `.gkill_context_menu_list` の max-height: 70vh で頭打ちになるので、
    // 400px なら最大280px。ヘッダのぶん行は中ほどに来るため、下向きには収まらない。
    // リサイズは **rykv を開く前** に行う。開いたあとに縮めると一覧が組み直され、
    // 測った直後の行がクリック前に別ノードへ差し替わってフレークになる。
    await page.setViewportSize({ width: 1000, height: 400 })
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)

    // 行そのものを右クリックする（余白を狙うとメニューが出ないことがある）
    await record.click({ button: 'right', force: true })

    await expectMenuInsideViewport(page)
  })
})
