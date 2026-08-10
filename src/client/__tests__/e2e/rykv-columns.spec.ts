import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel, searchByKeyword,
  clickSidebarSearchButton, expectPageToContainText,
} from './crud-helpers'

/**
 * rykv の複数列×検索。
 *
 * 「検索時の列に検索時の結果が表示される」を固定する。
 * 以前は列のidentity(query_id)が操作のたびに振り直され、さらにサイドバーが
 * 別列のクエリに乗っ取られてquery_idが重複し、検索結果が別の列に
 * 表示されることがあった(列リロード・列削除・別列検索との組み合わせ)。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/** rykv の列(KyouListView のルート)。左からの位置で掴む */
function column(page: Page, index: number) {
  return page.locator('.kyou_list_view_card_wrap').nth(index)
}

test.describe('rykv columns', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  test('別列で検索した結果は検索した列だけに反映され、列リロードでも混ざらない', async ({ page }) => {
    // 一覧は仮想スクロールで描画窓の外の記録が見えないため、
    // 列0も共通プレフィックスで絞ってから列間の独立性を確認する
    const base = makeUniqueLabel('rykv_col')
    const labelA = `${base}_a`
    const labelB = `${base}_b`
    await submitKftlText(page, labelA)
    await submitKftlText(page, labelB)

    await navigateToRykv(page)
    await expectPageToContainText(page, labelB)

    // 列0を共通プレフィックスで絞る(A・Bの2件だけになる)
    await searchByKeyword(page, base)
    await clickSidebarSearchButton(page)
    await expect(column(page, 0), '列0に検索結果が出ない').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 0), '列0に検索結果が出ない').toContainText(labelB, { timeout: 30000 })

    // 2列目を追加し(フォーカスは新しい列へ移る)、labelAだけに絞る
    await page.locator('.rykv_add_column_button').first().click()
    await expect(page.locator('.kyou_list_view_card_wrap'), '2列目が追加されない').toHaveCount(2, { timeout: 15000 })
    await searchByKeyword(page, labelA)
    await clickSidebarSearchButton(page)

    // 検索した列にだけ絞り込みが効く
    await expect(column(page, 1), '検索した列に結果が出ない').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 1), '検索した列に絞り込みが効いていない').not.toContainText(labelB, { timeout: 30000 })
    // 検索していない列0は影響を受けない
    await expect(column(page, 0), '検索していない列から記録が消えた').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 0), '検索していない列から記録が消えた').toContainText(labelB, { timeout: 30000 })

    // 列0のリロードは列1の絞り込みを壊さない
    const reloaded = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
    await column(page, 0).locator('button:has(.mdi-reload)').first().click()
    await reloaded
    await expect(column(page, 0), 'リロードした列に記録が戻らない').toContainText(labelB, { timeout: 30000 })
    await expect(column(page, 1), '列リロードで別列の絞り込みが失われた').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 1), '列リロードで別列に結果が混ざった').not.toContainText(labelB, { timeout: 30000 })
  })

  test('列を閉じても残った列の検索結果は保たれる', async ({ page }) => {
    const base = makeUniqueLabel('rykv_close')
    const labelA = `${base}_a`
    const labelB = `${base}_b`
    await submitKftlText(page, labelA)
    await submitKftlText(page, labelB)

    await navigateToRykv(page)
    await expectPageToContainText(page, labelB)

    // 列0を共通プレフィックスで絞ってから2列目を作る(仮想スクロールの描画窓対策)
    await searchByKeyword(page, base)
    await clickSidebarSearchButton(page)
    await expect(column(page, 0), '列0に検索結果が出ない').toContainText(labelA, { timeout: 30000 })

    await page.locator('.rykv_add_column_button').first().click()
    await expect(page.locator('.kyou_list_view_card_wrap'), '2列目が追加されない').toHaveCount(2, { timeout: 15000 })

    await searchByKeyword(page, labelA)
    await clickSidebarSearchButton(page)
    await expect(column(page, 1), '検索した列に結果が出ない').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 1), '検索した列に絞り込みが効いていない').not.toContainText(labelB, { timeout: 30000 })

    // 列0を閉じると、絞り込み済みの列がそのまま残る
    await column(page, 0).locator('button:has(.mdi-close)').first().click()
    await expect(page.locator('.kyou_list_view_card_wrap'), '列が閉じない').toHaveCount(1, { timeout: 15000 })
    await expect(column(page, 0), '残った列の検索結果が失われた').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 0), '残った列に別の検索結果が混ざった').not.toContainText(labelB, { timeout: 30000 })
  })

  test('検索中に別列をクリックしても、飛行中の検索は中断されず追加検索も発生しない', async ({ page }) => {
    // 列クリック→サイドバーのprops同期の残響が実検索になると、hot reload既定ONでは
    // 飛行中の検索をabortして最初からやり直すループになり、実環境(数百rep)では
    // サイドバー全再同期の反復でタブが無反応になる(2026-08-10の回帰)。
    // get_kyousを遅延させて「検索中」を作り、その間に別列をクリックして固定する
    const base = makeUniqueLabel('rykv_click')
    const labelA = `${base}_a`
    const labelB = `${base}_b`
    await submitKftlText(page, labelA)
    await submitKftlText(page, labelB)

    await navigateToRykv(page)
    await expectPageToContainText(page, labelB)

    await searchByKeyword(page, base)
    await clickSidebarSearchButton(page)
    await expect(column(page, 0), '列0に検索結果が出ない').toContainText(labelA, { timeout: 30000 })

    await page.locator('.rykv_add_column_button').first().click()
    await expect(page.locator('.kyou_list_view_card_wrap'), '2列目が追加されない').toHaveCount(2, { timeout: 15000 })
    await searchByKeyword(page, labelA)
    await clickSidebarSearchButton(page)
    await expect(column(page, 1), '検索した列に結果が出ない').toContainText(labelA, { timeout: 30000 })

    // ここから本題: get_kyousを遅延させ、列1の再検索が飛行中のうちに列0をクリックする
    let get_kyous_count = 0
    await page.route('**/api/get_kyous', async (route) => {
      get_kyous_count++
      await new Promise((resolve) => setTimeout(resolve, 1500))
      await route.continue()
    })

    const in_flight = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
    await column(page, 1).locator('button:has(.mdi-reload)').first().click()
    // 検索中に別列(列0)をクリックしてフォーカスを移す(ボタンの無い左上を突く)
    await column(page, 0).click({ position: { x: 10, y: 10 } })
    await in_flight

    // 飛行中だった列1の検索がabortされず、結果がそのまま正しい列に届く
    await expect(column(page, 1), '飛行中の検索結果が失われた').toContainText(labelA, { timeout: 30000 })
    await expect(column(page, 1), '別列クリックで絞り込みが壊れた').not.toContainText(labelB, { timeout: 30000 })
    // 列クリックの残響が追加の検索になっていない
    expect(get_kyous_count, '列クリックが余計な検索を発火した').toBe(1)
  })
})
