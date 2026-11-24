import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { submitLogin } from './helpers'
import { openApplicationConfigDialog } from './crud-helpers'
import { E2E_ADMIN_USER, E2E_USER, E2E_PASSWORD } from './e2e-credentials'

// プラグイン設定の導線（ADR-0714）: 設定 → サーバ設定 → アカウント管理 → 利用者の行の「プラグイン設定」。
// 記録のコンテキストメニューからは撤去したので、ここが唯一の入口。
//
// **管理者でなければ開けない。** 共有セッションの E2E_USER は管理者ではないので、
// 組み込みの admin で自前のセッションを作る（storageState を空にしてある）。
//
// E2E のホーム（$HOME/gkill_test）にはプラグインを置かないので、確認するのは
// 「5枚目のダイアログまで開き、対象の利用者名と『プラグインはまだありません』が出る」ところまで。
// 入口の配線が外れると、一覧が空なのではなくダイアログ自体が開かないので、ここで落ちる。

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

function topDialog(page: Page) {
  return page.locator('.gkill-floating-dialog').last()
}

test.describe('Plugin config (manage account)', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await submitLogin(page, E2E_ADMIN_USER, E2E_PASSWORD)
    await expect(page, '管理者でログインできない').not.toHaveURL(/\/(\?.*)?$/, { timeout: 30000 })
  })

  test('open the plugin list from an account row', async ({ page }) => {
    const settings = await openApplicationConfigDialog(page)
    await settings.getByRole('button', { name: 'サーバ設定', exact: true }).click()

    const serverConfig = topDialog(page)
    await expect(serverConfig.getByRole('button', { name: 'アカウント管理', exact: true }),
      'サーバ設定ダイアログが開かない').toBeVisible({ timeout: 30000 })
    await serverConfig.getByRole('button', { name: 'アカウント管理', exact: true }).click()

    const manageAccount = topDialog(page)
    // 管理者が他の利用者の行から開く（プラグインは利用者ごとの実体）
    const row = manageAccount.locator('tr', { hasText: E2E_USER }).first()
    await expect(row, 'アカウント管理ダイアログに行が出ない').toBeVisible({ timeout: 30000 })
    // 保存先割当とパスワードリセットの間に並ぶ
    await expect(row.getByRole('button', { name: '保存先割当', exact: true })).toBeVisible()
    await row.getByRole('button', { name: 'プラグイン設定', exact: true }).click()

    const pluginList = topDialog(page)
    // 見出しに出るのは「開いた行の利用者」。管理者自身の名前が出たら対象の受け渡しが外れている
    await expect(pluginList.getByText(`プラグイン設定: ${E2E_USER}`),
      'プラグイン一覧ダイアログが開かない').toBeVisible({ timeout: 30000 })
    await expect(pluginList.getByText('プラグインはまだありません', { exact: false }),
      'プラグインが無いことを伝えていない').toBeVisible({ timeout: 30000 })
  })
})

test.describe('Plugin config (non-admin guard)', () => {
  test('non-admin user does not see Server Config entry in settings', async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    const settings = await openApplicationConfigDialog(page)
    // 0件の確認は、描かれる前でも通ってしまう。同じ行に必ず出る「ログアウト」が見えてから数える
    await expect(settings.getByRole('button', { name: 'ログアウト', exact: true }),
      '設定画面の上部のボタンが描かれない').toBeVisible({ timeout: 30000 })
    await expect(settings.getByRole('button', { name: 'サーバ設定', exact: true }),
      '一般利用者にサーバ設定（プラグイン設定の入口）が出ている').toHaveCount(0)
  })
})
