import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel, expectPageToContainText,
} from './crud-helpers'

/**
 * rykv のデフォルト検索条件と「プロファイル×記録分類→記録先詳細」の算出。
 *
 * - 列を追加すると ApplicationConfig 由来のデフォルト検索条件
 *   (記録先/プロファイル/記録分類/タグ)が適用された検索が飛ぶ
 * - サマリ(記録分類)のチェック変更で記録先詳細が再計算され、検索条件にも反映される
 *
 * 2026-08-10 のフリーズ修正後、実環境でこの2つが死ぬ回帰があった
 * (古い世代の検索条件JSONによる parse/clone の例外と、デフォルト算出が
 *  永続化ツリーの古い is_checked を見ていたことが原因)。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/** get_kyous リクエストのクエリ部分を取り出す */
function parseGetKyousQuery(postData: string | null): Record<string, unknown> {
  const body = JSON.parse(postData ?? '{}')
  return body.query ?? {}
}

function column(page: Page, index: number) {
  return page.locator('.kyou_list_view_card_wrap').nth(index)
}

test.describe('rykv sidebar defaults', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  test('列追加時にApplicationConfig由来のデフォルト検索条件が適用される', async ({ page }) => {
    const label = makeUniqueLabel('rykv_default')
    await submitKftlText(page, label)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)

    const added_search = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
    await page.locator('.rykv_add_column_button').first().click()
    await expect(page.locator('.kyou_list_view_card_wrap'), '2列目が追加されない').toHaveCount(2, { timeout: 15000 })

    const query = parseGetKyousQuery((await added_search).request().postData())
    // 「条件が全部空の列」ではなく、初期チェック設定から算出された条件で検索される
    expect((query.reps as string[]).length, '記録先詳細が算出されていない').toBeGreaterThan(0)
    expect((query.devices_in_sidebar as string[]).length, 'プロファイルの既定が適用されていない').toBeGreaterThan(0)
    expect((query.rep_types_in_sidebar as string[]).length, '記録分類の既定が適用されていない').toBeGreaterThan(0)
    expect((query.tags as string[]).length, 'タグの既定が適用されていない').toBeGreaterThan(0)
    // 新しい列にも検索結果が表示される
    await expect(column(page, 1), '追加した列に結果が出ない').toContainText(label, { timeout: 30000 })
  })

  test('記録分類のチェック変更で記録先詳細が再計算される', async ({ page }) => {
    const label = makeUniqueLabel('rykv_summary')
    await submitKftlText(page, label)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)

    // 記録先詳細ツリーのチェック数を数える(記録先タブは eager 描画なので非表示でも読める)
    const count_rep_detail_checks = async () => {
      return await page.evaluate(() => {
        const tables = document.querySelectorAll('.replist table')
        const detail_table = tables[tables.length - 1]
        const boxes = detail_table?.querySelectorAll('input.checkbox_in_foldable_struct') ?? []
        let checked = 0
        boxes.forEach((box) => {
          checked += (box as HTMLInputElement).checked ? 1 : 0
        })
        return { total: boxes.length, checked }
      })
    }

    const before = await count_rep_detail_checks()
    expect(before.total, '記録先詳細ツリーが描画されていない').toBeGreaterThan(0)
    expect(before.checked, '初期状態で記録先詳細が算出されていない').toBeGreaterThan(0)

    // 記録分類ツリーのルートチェックを外す → 記録先詳細が全て外れ、reps=[] の検索になる
    const uncheck_search = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
    await page.locator('.replist .typelist input.checkbox_in_foldable_struct').first().click()
    const uncheck_query = parseGetKyousQuery((await uncheck_search).request().postData())
    expect(uncheck_query.reps, '記録分類OFFが記録先詳細へ伝播していない').toEqual([])
    await expect.poll(async () => (await count_rep_detail_checks()).checked, {
      message: '記録先詳細ツリーのチェックが外れない',
      timeout: 15000,
    }).toBe(0)

    // 戻すと記録先詳細も復元される
    const recheck_search = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
    await page.locator('.replist .typelist input.checkbox_in_foldable_struct').first().click()
    const recheck_query = parseGetKyousQuery((await recheck_search).request().postData())
    expect((recheck_query.reps as string[]).length, '記録分類ONで記録先詳細が再計算されない').toBeGreaterThan(0)
    await expect.poll(async () => (await count_rep_detail_checks()).checked, {
      message: '記録先詳細ツリーのチェックが戻らない',
      timeout: 15000,
    }).toBe(before.checked)
  })
})
