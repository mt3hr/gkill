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
 * 「時間帯と曜日を指定して検索しても出ない」の調査で見つけた（主因は別で ADR-0220）。
 *
 * 開始・終了の時刻欄を空に戻す手段は、欄の中の ×（Vuetify の clearable）だけ（ADR-0419）。
 * 節の「クリア」は曜日も落とすので、「開始だけ未指定に戻す」が表せなかった。
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

  test('時刻の × で開始時刻だけが未指定へ戻る', async ({ page }) => {
    await navigateToRykv(page)

    const sidebar = page.locator('.rykv_query_editor_sidebar')
    const checkbox = sidebar.locator('.v-checkbox').filter({ hasText: '時間帯' }).first().locator('input')
    await checkbox.scrollIntoViewIfNeeded()
    const checked_query = await nextGetKyousQuery(page, () => checkbox.check())
    expect(checked_query.period_of_time_start_time_second, 'チェック直後は時刻未指定のはず').toBeNull()

    const start_field = sidebar.locator('.period_of_time_query_start_time_field input[readonly]')
    const end_field = sidebar.locator('.period_of_time_query_end_time_field input[readonly]')
    await expect(start_field, '開始時刻の欄が出ていない').toBeVisible({ timeout: 30000 })
    await expect(start_field, '時刻は未指定（空）が既定').toHaveValue('')

    // 空の欄には × が出ない（clearable の × は v-show="dirty"）
    await expect(sidebar.locator('.period_of_time_query_start_time_field .v-field__clearable'),
      '空の欄に × が出ている').toBeHidden()

    // 開始時刻に 9:00 を入れる
    await start_field.click()
    const clock_items = page.locator('.v-time-picker .v-time-picker-clock__item')
    await clock_items.filter({ hasText: /^9$/ }).first().click()
    const timed_query = await nextGetKyousQuery(page, async () => {
      await clock_items.filter({ hasText: /^00$/ }).first().click()
    })
    await expect(start_field).toHaveValue('09:00')
    expect(timed_query.period_of_time_start_time_second, '入れた時刻が条件に乗っていない').not.toBeNull()

    // × を押すと、その欄だけが未指定へ戻って検索がやり直される。
    // 飛行中の応答を取り違えないよう、条件の中身で待つ
    const cleared = page.waitForResponse((res) => res.url().includes('/api/get_kyous')
      && parseGetKyousQuery(res.request().postData()).period_of_time_start_time_second === null, { timeout: 30000 })
    await sidebar.locator('.period_of_time_query_start_time_field .v-field__clearable .v-icon').click()
    await cleared
    await expect(start_field, '× で空に戻らない（未指定を表せなくなる）').toHaveValue('')
    await expect(end_field, '触っていない終了時刻が動いている').toHaveValue('')
    // onClear が stopPropagation するので、× では activator のメニューが開かない
    await expect(page.locator('.v-time-picker'), '× でピッカーのメニューが開いた').toHaveCount(0)
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
