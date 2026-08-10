/**
 * 保存済み検索条件のE2E。
 *
 * 設定画面（検索条件 → ライフログ検索条件）で名前付きの検索条件を登録し、
 * 設定の「適用」（ページリロード）後に、ライフログビュー画面のサイドバーに
 * 呼び出しFABが現れて、選択するとサイドバーへ条件が反映されることを通しで確認する。
 * タスク画面側は未登録のままなのでFABが出ないことも確認する（0件非表示）。
 */
import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { navigateToMi, navigateToRykv } from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Saved Find Query', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  // いちばん上に表示されているフローティングダイアログ
  const topDialog = (page: import('@playwright/test').Page) =>
    page.locator('.gkill-floating-dialog').last()

  test('タスク側に登録が無ければタスク画面のFABは表示されない', async ({ page }) => {
    await navigateToMi(page)
    const sidebar = page.locator('.v-navigation-drawer')
    await expect(sidebar.first()).toBeVisible({ timeout: 30000 })
    await expect(page.locator('.saved_find_query_fab')).toHaveCount(0)
  })

  test('設定画面で登録した検索条件をサイドバーFABから呼び出せる', async ({ page }) => {
    await navigateToRykv(page)

    // 設定 → 検索条件 → ライフログ検索条件
    await page.locator('button:has(.mdi-cog)').first().click()
    await page.getByRole('button', { name: '検索条件', exact: true }).click({ timeout: 15000 })
    await expect(topDialog(page).getByRole('button', { name: 'ライフログ検索条件', exact: true }))
      .toBeVisible({ timeout: 15000 })
    await topDialog(page).getByRole('button', { name: 'ライフログ検索条件', exact: true }).click()

    // 一覧管理: 追加して名前を付ける
    await topDialog(page).locator('button:has(.mdi-plus)').click({ timeout: 15000 })
    const nameField = topDialog(page).getByLabel('名前')
    await expect(nameField, '追加した行の名前欄が出ない').toBeVisible({ timeout: 15000 })
    await nameField.fill('E2E保存条件')

    // 検索条件を編集: キーワードを設定して保存
    await topDialog(page).getByRole('button', { name: '検索条件を編集', exact: true }).click()
    const editor = topDialog(page)
    const keywordCheckbox = editor.locator('.v-checkbox').filter({ hasText: 'キーワード' }).first().locator('input')
    await expect(keywordCheckbox, 'クエリエディタのキーワード節が出ない').toBeVisible({ timeout: 30000 })
    await keywordCheckbox.click()
    const keywordField = editor.locator('.v-text-field input').first()
    await expect(keywordField).toBeVisible({ timeout: 15000 })
    await keywordField.fill('E2E検索キーワード')
    await editor.getByRole('button', { name: '保存', exact: true }).click()

    // 一覧 → ハブ → 設定画面本体、と順に適用（設定はページリロードで確定する）
    await topDialog(page).getByRole('button', { name: '適用', exact: true }).click()
    await topDialog(page).getByRole('button', { name: '適用', exact: true }).click()
    await topDialog(page).getByRole('button', { name: '適用', exact: true }).click()

    // リロード後、サイドバーにFABが現れる
    const fab = page.locator('.saved_find_query_fab')
    await expect(fab, '登録したのに呼び出しFABが出ない').toBeVisible({ timeout: 60000 })

    // FABメニューから名前を選ぶと、サイドバーへ条件が反映される
    await fab.locator('button').click()
    await page.getByText('E2E保存条件', { exact: true }).click({ timeout: 15000 })
    const sidebar = page.locator('.rykv_query_editor_sidebar')
    await expect(
      sidebar.locator('.v-text-field input').first(),
      '保存したキーワードがサイドバーに反映されない',
    ).toHaveValue('E2E検索キーワード', { timeout: 30000 })
  })
})
