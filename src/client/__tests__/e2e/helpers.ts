import { expect, type Page } from '@playwright/test'
import { E2E_USER, E2E_PASSWORD } from './e2e-credentials'

/**
 * 認証済みの状態で開始する。
 *
 * setup プロジェクトが作ったセッション Cookie（storageState）を使うので、ここでは
 * 「認証が生きていること」を確かめるだけ。**戻り値ではなく expect で落とす。**
 * 以前は boolean を返していたが、79ある呼び出し元のどれも戻り値を見ておらず、
 * セッションが死んでいても各テストが「要素が出ない」で個別にタイムアウトするだけで、
 * 本当の原因（セッション切れ）が大量のタイムアウトに埋もれていた。
 */
export async function loginAsAdmin(page: Page): Promise<void> {
  await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await expect(page, 'セッションが無効（ログイン画面へ戻された）。'
    + 'storageState は全テストで共有しているので、ログアウト等でセッションを壊すテストは'
    + 'test.use({ storageState: { cookies: [], origins: [] } }) で自前のセッションを持つこと')
    .toHaveURL(/\/(kftl|rykv|mi|saihate|dashboard|plaing|mkfl|rudbeckia)/, { timeout: 30000 })
}

/** ログイン画面を開いて、ユーザID / パスワードの入力欄が描けるまで待つ */
export async function openLoginPage(page: Page) {
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  const inputs = page.locator('input')
  // 2つ目（パスワード）が見えた時点で、両方そろっている
  await expect(inputs.nth(1), 'ログイン画面の入力欄が描かれない').toBeVisible({ timeout: 30000 })
  return inputs
}

/** ログイン画面から実際にログインする。**自前のセッションが要るテスト専用。** */
export async function submitLogin(page: Page, user: string, password: string): Promise<void> {
  const inputs = await openLoginPage(page)
  await inputs.nth(0).fill(user)
  await inputs.nth(1).fill(password)
  const loginButton = page.locator('button').filter({ hasText: /ログイン|login/i }).first()
  await expect(loginButton, 'ログインボタンが見つからない').toBeVisible({ timeout: 15000 })
  await loginButton.click()
}

/** E2E のテストユーザで自前のセッションを作る（storageState を空にした describe から呼ぶ） */
export async function loginAsE2EUser(page: Page): Promise<void> {
  await submitLogin(page, E2E_USER, E2E_PASSWORD)
  await expect(page, 'ログインできない').not.toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })
}
