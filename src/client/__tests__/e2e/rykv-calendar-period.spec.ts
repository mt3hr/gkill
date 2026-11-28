import { test, expect } from '@playwright/test'
import type { Locator, Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { navigateToRykv, navigateToMi } from './crud-helpers'

/**
 * サイドバーの「期間」（旧「日付」）と、その開始時刻・終了時刻。
 *
 * - メニュー名は「期間」。日付だけでなく時刻も指定できる
 * - 時刻を入れなければ従来どおり日の境界（00:00:00.000 / 23:59:59.999）。
 *   ここが変わると、保存済みの全列の検索結果が黙ってずれる
 * - 空の時刻欄が「未指定」を表す唯一の手段なので、clearable を外してはいけない
 * - 送るキーは calendar_start_time / calendar_end_time（旧 calendar_start_date / _end_date から改名。retired-spelling-ok（改名前のキー名を検査するため名指しする））
 *
 * calendar-query.vue は rykv と mi の共有なので、両方で確かめる。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

function parseGetKyousQuery(postData: string | null): Record<string, unknown> {
  const body = JSON.parse(postData ?? '{}')
  return body.query ?? {}
}

async function nextGetKyousQuery(page: Page, action: () => Promise<void>): Promise<Record<string, unknown>> {
  const response = page.waitForResponse((res) => res.url().includes('/api/get_kyous'), { timeout: 30000 })
  await action()
  return parseGetKyousQuery((await response).request().postData())
}

/** 「期間」のチェックボックス */
function periodCheckbox(sidebar: Locator): Locator {
  return sidebar.locator('.v-checkbox').filter({ hasText: '期間' }).first().locator('input')
}

/**
 * 期間の時刻欄。
 *
 * ラベルは「開始時刻」「終了時刻」だが、ラベルでは引かない。
 * 時間帯フィルタにも似たラベルの時刻欄があり、DOM 順ではそちらが先に来るので、
 * 期間節だけに付けたクラスで引く。
 */
function startTimeField(sidebar: Locator): Locator {
  return sidebar.locator('.calendar_query_start_time_field input[readonly]')
}

function endTimeField(sidebar: Locator): Locator {
  return sidebar.locator('.calendar_query_end_time_field input[readonly]')
}

test.describe('サイドバーの「期間」', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  test('rykv: メニュー名が「期間」で、時刻を入れると秒まで条件に乗り、クリアで日の境界へ戻る', async ({ page }) => {
    await navigateToRykv(page)

    const sidebar = page.locator('.rykv_query_editor_sidebar')
    await expect(sidebar.locator('.v-checkbox').filter({ hasText: '期間' }).first(),
      'メニュー名が「期間」になっていない').toBeVisible({ timeout: 30000 })
    await expect(sidebar.locator('.v-checkbox').filter({ hasText: '日付' }),
      '旧メニュー名「日付」が残っている').toHaveCount(0)

    const checkbox = periodCheckbox(sidebar)
    await checkbox.scrollIntoViewIfNeeded()
    await checkbox.check()

    // 時刻欄は既定で空（= 未指定）。clearable が無いと未指定へ戻せなくなる
    const start_field = startTimeField(sidebar)
    const end_field = endTimeField(sidebar)
    await expect(start_field, '開始時刻の欄が出ていない').toBeVisible()
    await expect(end_field, '終了時刻の欄が出ていない').toBeVisible()
    await expect(start_field, '時刻は未指定（空）が既定').toHaveValue('')

    // 日付を1日選ぶ。選んだ直後の条件は日の境界ちょうど
    const day_buttons = sidebar.locator('.calendar_query_date_picker .v-date-picker-month__day button:not([disabled])')
    const dated_query = await nextGetKyousQuery(page, async () => {
      await day_buttons.nth(15).click()
    })
    // retired-spelling-ok（改名前のキーを送っていないことの検査なので、旧キー名を書く必要がある）
    expect(dated_query, '改名前のキーを送っている').not.toHaveProperty('calendar_start_date') // retired-spelling-ok
    expect(dated_query, '改名前のキーを送っている').not.toHaveProperty('calendar_end_date') // retired-spelling-ok
    const start_default = new Date(dated_query.calendar_start_time as string)
    const end_default = new Date(dated_query.calendar_end_time as string)
    expect([start_default.getHours(), start_default.getMinutes(), start_default.getSeconds(), start_default.getMilliseconds()],
      '時刻未指定の開始が日の始まりでない（保存済みの全列の結果が黙ってずれる）').toEqual([0, 0, 0, 0])
    expect([end_default.getHours(), end_default.getMinutes(), end_default.getSeconds(), end_default.getMilliseconds()],
      '時刻未指定の終了が日の終わりでない').toEqual([23, 59, 59, 999])

    // 開始時刻に 9:00 を入れる
    await start_field.click()
    const clock_items = page.locator('.v-time-picker .v-time-picker-clock__item')
    await clock_items.filter({ hasText: /^9$/ }).first().click()
    const timed_query = await nextGetKyousQuery(page, async () => {
      await clock_items.filter({ hasText: /^00$/ }).first().click()
    })
    await expect(start_field).toHaveValue('09:00')
    const start_timed = new Date(timed_query.calendar_start_time as string)
    expect([start_timed.getHours(), start_timed.getMinutes(), start_timed.getSeconds(), start_timed.getMilliseconds()],
      '入れた時刻が条件に乗っていない').toEqual([9, 0, 0, 0])
    const end_untouched = new Date(timed_query.calendar_end_time as string)
    expect([end_untouched.getHours(), end_untouched.getMinutes()],
      '触っていない終了時刻が動いている').toEqual([23, 59])

    // クリアで未指定（日の始まり）へ戻る
    const cleared_query = await nextGetKyousQuery(page, async () => {
      await sidebar.locator('.calendar_query_start_time_field .v-field__clearable .v-icon').click()
    })
    await expect(start_field, 'クリアで空に戻らない（未指定を表せなくなる）').toHaveValue('')
    const start_cleared = new Date(cleared_query.calendar_start_time as string)
    expect([start_cleared.getHours(), start_cleared.getMinutes(), start_cleared.getSeconds(), start_cleared.getMilliseconds()])
      .toEqual([0, 0, 0, 0])
  })

  test('mi: 同じ「期間」と時刻欄が出る（calendar-query.vue は rykv と共有）', async ({ page }) => {
    await navigateToMi(page)

    const sidebar = page.locator('.mi_query_editor_sidebar')
    await expect(sidebar.locator('.v-checkbox').filter({ hasText: '期間' }).first(),
      'mi 側のメニュー名が「期間」になっていない').toBeVisible({ timeout: 30000 })
    await expect(sidebar.locator('.v-checkbox').filter({ hasText: '日付' }),
      'mi 側に旧メニュー名「日付」が残っている').toHaveCount(0)

    const checkbox = periodCheckbox(sidebar)
    await checkbox.scrollIntoViewIfNeeded()
    await checkbox.check()

    await expect(startTimeField(sidebar), 'mi 側に開始時刻の欄が出ていない').toBeVisible()
    await expect(endTimeField(sidebar), 'mi 側に終了時刻の欄が出ていない').toBeVisible()
  })
})
