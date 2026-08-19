import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToMi,
  makeUniqueLabel, searchByKeyword, clickSidebarSearchButton,
  findKyouByText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Search and Summary Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番66: 記録された情報を検索する
  test('search records by keyword on rykv page', async ({ page }) => {
    const label = makeUniqueLabel('search_test')
    await submitKftlText(page, label)

    await navigateToRykv(page)

    // サイドバーを開く・キーワードを有効にする・入力するまでを searchByKeyword が引き受ける。
    // 見つからなければそこで落ちるので、条件で包まない
    await searchByKeyword(page, label)
    // rykv_hot_reload の設定に依らず、検索ボタンで確実に走らせる
    await clickSidebarSearchButton(page)

    await expect(findKyouByText(page, label).first(), '検索した記録が一覧に出ない')
      .toBeVisible({ timeout: 30000 })
  })

  // 項番69: 一日の記録サマリを閲覧する (rykv の集計ビュー)
  test('toggle dnote summary panel on rykv page', async ({ page }) => {
    const label = makeUniqueLabel('dnote_test')
    await submitKftlText(page, label)

    await navigateToRykv(page)

    // 集計ビューの開閉ボタン。rykv-view.vue の mdi-file-chart-outline を持つボタン
    const dnoteToggle = page.locator('.v-app-bar button').filter({ has: page.locator('.mdi-file-chart-outline') }).first()
    await expect(dnoteToggle, '集計ビューの開閉ボタンが見つからない').toBeVisible({ timeout: 30000 })

    const dnote = page.locator('.rykv_dnote_wrap')
    await expect(dnote, '押す前から集計ビューが出ている').toHaveCount(0)

    await dnoteToggle.click()
    await expect(dnote, '集計ビューが開かない').toBeVisible({ timeout: 30000 })

    // もう一度押すと閉じる（開きっぱなしだと後続のテストの列幅が変わる）
    await dnoteToggle.click()
    await expect(dnote, '集計ビューが閉じない').toHaveCount(0, { timeout: 30000 })
  })

  // 項番70: タスク情報を検索する (Mi board search)
  test('search tasks by keyword on mi board page', async ({ page }) => {
    const label = makeUniqueLabel('mi_search_test')
    await submitKftlText(page, `ーみ\n${label}`)

    await navigateToMi(page)

    await searchByKeyword(page, label)
    await clickSidebarSearchButton(page)

    await expect(findKyouByText(page, label).first(), '検索したタスクが板に出ない')
      .toBeVisible({ timeout: 30000 })
  })
})
