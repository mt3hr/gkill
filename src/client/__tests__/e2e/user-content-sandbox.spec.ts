/**
 * 取り込んだ HTML ファイルを /files/ で開くと、中のスクリプトは動くが gkill のクッキーと localStorage には触れないこと。
 *
 * /files/ と /zip_cache/ は .html / .htm / .xhtml に `Content-Security-Policy: sandbox allow-scripts allow-downloads allow-modals`
 * を付ける（utils.go の userContentSecurityPolicy、ADR-0711）。allow-same-origin を付けないので文書は opaque origin になり、
 * document.cookie と localStorage は例外になる。セッションクッキーは JS が document.cookie で書いていて HttpOnly ではないので、
 * ここが gkill のオリジンのまま動くとセッションIDが読めてしまう。
 *
 * ヘッダの値そのものは Go のテスト（handle_file_serve_test.go / handle_zip_cache_file_serve_test.go）が拡張子ごとに固定している。
 * ここでは「実際のブラウザでその値がそう効くこと」を見る。
 */
import { test, expect } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

const PROBE_FILE_NAME = 'zz_e2e_user_content_sandbox_probe.html'

// 合成のテスト用 HTML。外部とは通信しない。
// 判定の分岐はページ側の文字列の中にだけ置く（spec に条件分岐を書かない規約）
const PROBE_HTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>sandbox probe</title></head>
<body>
<p id="ran">not run</p>
<p id="cookie">not checked</p>
<p id="storage">not checked</p>
<script>
document.getElementById('ran').textContent = 'ok'
try {
  document.getElementById('cookie').textContent = 'readable:' + document.cookie
} catch (e) {
  document.getElementById('cookie').textContent = 'blocked'
}
try {
  window.localStorage.getItem('probe')
  document.getElementById('storage').textContent = 'readable'
} catch (e) {
  document.getElementById('storage').textContent = 'blocked'
}
</script>
</body></html>
`

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

test.describe('利用者の HTML ファイルの sandbox', () => {
  test.beforeEach(async ({ page }) => {
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('/files/ の HTML はスクリプトが動き、クッキーと localStorage は例外になる', async ({ page }) => {
    // /api は Cookie ではなく本文の session_id で認証する（auth_middleware.go の sessionPeek）
    const [session] = (await page.context().cookies()).filter((cookie) => cookie.name === 'gkill_session_id')
    expect(session, 'gkill_session_id クッキーが無い（ログインできていない）').toBeDefined()

    // 初回登録で作られる directory 型の rep「Files」へ上げる（use-register-first-account-view.ts の do_initialize）
    const upload = await page.request.post('/api/upload_files', {
      data: {
        session_id: session.value,
        target_rep_name: 'Files',
        conflict_behavior: 'override',
        files: [{
          file_name: PROBE_FILE_NAME,
          data_base64: Buffer.from(PROBE_HTML, 'utf-8').toString('base64'),
          last_modified: '2024-01-01T00:00:00Z',
        }],
        locale_name: 'ja',
      },
    })
    expect(upload.ok(), `upload_files の HTTP ステータスが ${upload.status()}`).toBe(true)
    const upload_body = await upload.json() as { errors: Array<unknown> }
    expect(upload_body.errors, 'upload_files がエラーを返した').toEqual([])

    const response = await page.goto(`/files/Files/${PROBE_FILE_NAME}`, { waitUntil: 'load' })
    expect(response, '/files/ の応答が無い').not.toBeNull()
    expect(response?.status()).toBe(200)
    expect(response?.headers()['content-security-policy']).toBe('sandbox allow-scripts allow-downloads allow-modals')

    // スクリプトが動いたこと（sandbox だけだと真っ白のまま "not run" が残る）
    await expect(page.locator('#ran')).toHaveText('ok')
    // opaque origin なので、セッションクッキーも localStorage も読めない
    await expect(page.locator('#cookie')).toHaveText('blocked')
    await expect(page.locator('#storage')).toHaveText('blocked')
  })
})
