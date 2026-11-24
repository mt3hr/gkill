/**
 * E2E で使うテストアカウント。
 *
 * `auth.setup.ts` はテストファイル（setup プロジェクト）なので、
 * ここから定数を出す。Playwright は「テストファイルが別のテストファイルを import する」のを
 * エラーにするため、spec 側が auth.setup.ts を直接 import するとスイート全体が起動しない。
 */
// ── 共有セッションの約束 ──────────────────────────────────────────────
// `default` プロジェクトは setup（auth.setup.ts）が作った storageState を全テストで共有する。
// 中身は **セッションCookie 1本** で、サーバ側の session 行と1対1に対応している。
//
// **セッションを壊す操作を含むテストは、必ず自前のセッションを持つこと。**
//     test.describe('...', () => {
//       test.use({ storageState: { cookies: [], origins: [] } })
//       test('...', async ({ page }) => { await loginAsE2EUser(page); ... })
//     })
// 対象は「ログアウト」「アカウントの無効化」「パスワード変更」など。
// 共有セッションのまま実行すると、以降の全テストが最初のAPI呼び出しで ERR000013 を受け、
// gkill-api.ts の check_auth が `/` へ飛ばすので、**大量の「要素が出ない」タイムアウト**になり
// 本当の原因が埋もれる（2026-08-19 に実際に34件が巻き添えになった）。
export const E2E_USER = 'e2e_user'
export const E2E_PASSWORD = 'e2etest'

// 組み込みの管理者アカウント。サーバが初回起動時に自分で作り（gkill_server_api.go）、
// auth.setup.ts の初回登録フォームの「管理者パスワード」欄が E2E_PASSWORD を設定する。
// **E2E_USER は管理者ではない**ので、管理者しか開けない画面（サーバ設定・アカウント管理・
// プラグイン設定）を触るテストはこちらで自前のセッションを作ること。
export const E2E_ADMIN_USER = 'admin'
