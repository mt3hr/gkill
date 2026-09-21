import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel, expectPageToContainText,
} from './crud-helpers'

/**
 * rykv サイドバーの「時間帯」の曜日ボタン。
 *
 * - 時間帯にチェックを入れた直後は曜日が未選択で、そのまま [] を送ると「0件指定」になり
 *   曜日を押すまで対象なしだった。未選択は全曜日(=曜日制限なし)として送る
 * - 1つ押せばその曜日だけになる（全点灯から外していく形にしない）
 * - 選択した曜日は塗り潰し、未選択は枠線。以前は :active のオーバーレイだけで、
 *   選んだ曜日のほうが薄く見えて反転して読めた
 *
 * 2026-09-16 に「11:45〜12:45 水曜で検索しても出ない」の調査で見つけた（主因は別で ADR-0220）。
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

/** 次の get_kyous の応答を待ち、そのリクエストのクエリを返す */
async function nextGetKyousQuery(page: Page, action: () => Promise<void>): Promise<Record<string, unknown>> {
  const response = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
  await action()
  return parseGetKyousQuery((await response).request().postData())
}

const ALL_WEEK_OF_DAYS = [0, 1, 2, 3, 4, 5, 6]

test.describe('rykv 時間帯の曜日', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  test('チェック直後は全曜日で検索し、1つ押すとその曜日だけになる', async ({ page }) => {
    const label = makeUniqueLabel('rykv_wod')
    await submitKftlText(page, label)

    await navigateToRykv(page)
    await expectPageToContainText(page, label)

    const sidebar = page.locator('.rykv_query_editor_sidebar')
    const checkbox = sidebar.locator('.v-checkbox').filter({ hasText: '時間帯' }).first().locator('input')
    await checkbox.scrollIntoViewIfNeeded()

    // チェックを入れた瞬間の検索は全曜日（[] の0件指定ではない）で、記録は消えない
    const checked_query = await nextGetKyousQuery(page, () => checkbox.click())
    expect(checked_query.period_of_time_week_of_days, 'チェック直後に [] (0件指定) を送っている').toEqual(ALL_WEEK_OF_DAYS)
    await expectPageToContainText(page, label)

    const buttons = sidebar.locator('.period_of_time_week_of_day_button')
    await expect(buttons).toHaveCount(7)
    await expect(sidebar.locator('.period_of_time_week_of_day_button[aria-pressed="true"]'), '未選択なのに押された曜日がある').toHaveCount(0)
    await expect(sidebar.locator('.period_of_time_week_of_day_button.v-btn--variant-outlined'), '未選択は枠線で描く').toHaveCount(7)

    // 今日以外の曜日を1つ押す → その曜日だけになり、今日作った記録は消える
    const today = new Date().getDay()
    const other = (today + 1) % 7
    const other_query = await nextGetKyousQuery(page, () => buttons.nth(other).click())
    expect(other_query.period_of_time_week_of_days, '1つ押したらその曜日だけになるべき').toEqual([other])
    await expect(buttons.nth(other)).toHaveAttribute('aria-pressed', 'true')
    await expect(buttons.nth(other), '選択した曜日は塗り潰しで描く').toHaveClass(/v-btn--variant-flat/)
    await expect(sidebar.locator('.period_of_time_week_of_day_button[aria-pressed="true"]')).toHaveCount(1)
    // 「消えたこと」は件数で見る。first() の本文を見る形だと、0件で要素が無いときに何も検査していない
    await expect(page.locator('.kyou_list_view_card_wrap', { hasText: label }), '別の曜日だけの検索に今日の記録が残っている')
      .toHaveCount(0, { timeout: 30000 })

    // 外せばまた全曜日で、記録が戻る
    const cleared_query = await nextGetKyousQuery(page, () => buttons.nth(other).click())
    expect(cleared_query.period_of_time_week_of_days).toEqual(ALL_WEEK_OF_DAYS)
    await expect(sidebar.locator('.period_of_time_week_of_day_button[aria-pressed="true"]'), '全曜日の往復で全点灯に化けた').toHaveCount(0)
    await expectPageToContainText(page, label)

    // 今日の曜日だけにしても記録は残る
    const today_query = await nextGetKyousQuery(page, () => buttons.nth(today).click())
    expect(today_query.period_of_time_week_of_days).toEqual([today])
    await expectPageToContainText(page, label)
  })
})
