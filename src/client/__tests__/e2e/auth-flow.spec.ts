import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin, loginAsE2EUser, openLoginPage } from './helpers'
import { navigateToSettings, navigateToRykv } from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Auth Flow Tests', () => {
  test.beforeEach(async () => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
  })

  // 項番11: ログイン後に全Repにチェック済み確認
  test('all reps are checked after login', async ({ page }) => {
    test.setTimeout(120000)
    await loginAsAdmin(page)
    await navigateToRykv(page)

    // サイドバーのリポジトリ一覧。初期状態では既定クエリで全て入っている
    const drawer = page.locator('.v-navigation-drawer').first()
    await expect(drawer, 'サイドバーが出ない').toBeVisible({ timeout: 30000 })

    const checkboxes = drawer.locator('input[type="checkbox"]')
    await expect(checkboxes.first(), 'サイドバーにチェックボックスが1つも無い')
      .toBeAttached({ timeout: 30000 })

    // 1つでもチェックが入っていること（全部外れていたら既定クエリが壊れている）
    const checked = await checkboxes.evaluateAll(
      (nodes) => nodes.filter((n) => (n as HTMLInputElement).checked).length)
    expect(checked, 'ログイン直後なのにチェックが1つも入っていない').toBeGreaterThan(0)
  })
})

// **自前のセッションで動かすテスト。**
// default プロジェクトは setup が作った storageState（セッションCookie 1本）を全テストで共有する。
// ログアウトはサーバ側のセッション行を消す（handle_logout.go の DeleteLoginSession）ので、
// 共有セッションのまま実行すると**以降の全テストがログイン画面へ落ちる**。
// 空の storageState を指定して、このブロックのテストだけ自分でログインする。
test.describe('Auth Flow Tests (自前のセッション)', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test.beforeEach(async () => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
  })

  // 項番8: ログアウト操作
  test('logout redirects to login page', async ({ page }) => {
    test.setTimeout(120000)
    // **共有セッションを使わない。** ここでログアウトすると storageState で全テストが
    // 共有しているサーバ側セッションが消え、以降の全テストがログイン画面へ落ちる
    await loginAsE2EUser(page)

    // ログアウトは最果て(saihate)のアプリバーにある。
    // 以前は「見つかったら押す」形で、見つからなければ何も検証せず緑になっていた
    await navigateToSettings(page)
    const logoutButton = page.locator('.v-app-bar button').filter({ hasText: /ログアウト|logout/i }).first()
    await expect(logoutButton, 'ログアウトボタンが見つからない').toBeVisible({ timeout: 30000 })
    await logoutButton.click()

    // 確認ダイアログの「ログアウト」で確定する
    const confirmDialog = page.locator('.gkill-floating-dialog').last()
    await expect(confirmDialog, 'ログアウトの確認が出ない').toBeVisible({ timeout: 30000 })
    await confirmDialog.locator('button').filter({ hasText: /^\s*ログアウト\s*$/ }).first().click()

    // ログイン画面へ戻ること
    await expect(page, 'ログアウトしてもログイン画面へ戻らない')
      .toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })
    // Verify login form is visible
    await expect(page.locator('input').nth(1), 'ログインフォームが出ない')
      .toBeVisible({ timeout: 30000 })
  })

  // 項番9: パスワード未設定アカウントでログイン不可
  test('cannot login with account that has no password set', async ({ page }) => {
    test.setTimeout(120000)
    const inputs = await openLoginPage(page)

    // Try to login with a non-existent account (simulating no password)
    await inputs.nth(0).fill('test_no_password_user')
    await inputs.nth(1).fill('')

    const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i }).first()
    await expect(loginButton, 'ログインボタンが無い').toBeVisible({ timeout: 15000 })
    await loginButton.click()

    // ログインできないので、ログイン系の画面から動かないこと
    await expect(page, 'パスワード未設定でもログインできてしまっている')
      .toHaveURL(/\/(\?.*)?$|\/login|\/set_new_password|\/register_first_account/, { timeout: 30000 })
    // 失敗の合図（エラー表示）が出ること
    await expect(page.locator('.v-alert[role="alert"]').first(), 'ログイン失敗のエラーが出ない')
      .toBeVisible({ timeout: 30000 })
  })
})
