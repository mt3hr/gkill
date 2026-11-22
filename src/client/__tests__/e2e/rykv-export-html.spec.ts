import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

/**
 * rykv の「HTML で出力」。
 *
 * 出力した HTML の約束は1つだけ: **通信をしない**（地図だけ例外）。
 * これが破れても画面は普通に出るので、気づけるのはここだけになる。
 *
 * 組み立ての中身（API キーを焼き込まないこと・`</script>` で割れないこと等）は
 * Go 側の handle_export_rykv_html_test.go / export_html_render_test.go が見ている。
 * ここでは**出来上がった HTML そのもの**を検査する。
 *
 * ダイアログの操作は通らない。共有フッタの表示は利用者の設定で、E2E から
 * 設定を変えると他の spec に影響する。導線はユニット（export-rykv-html-view.test.ts）が見ている。
 */

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

/** 出力の要求。サイドバーが生成するのと同じ形（タグと記録保管場所を絞らない）で送る */
function export_request(customize?: (body: Record<string, unknown>) => void): Record<string, unknown> {
  const body: Record<string, unknown> = {
    locale_name: 'ja',
    title: 'E2E の出力',
    columns: [{ query_id: 'e2e-export-column' }],
    include_attached_tags: true,
    include_attached_texts: true,
    include_attached_notifications: true,
    include_attached_timeiss: false,
    include_gps_log: false,
    include_plugin_content: false,
    include_dnote: true,
    include_map: false,
    include_sidebar: true,
    media_mode: 'thumbnail',
    output_format: 'single',
  }
  if (customize) {
    customize(body)
  }
  return body
}

/**
 * 出力を1回投げる。E2E は PATH 上の gkill_server を起動するので、
 * そのバイナリが古いとこの API を持たない（JSON ではなく 404 の HTML が返る）。
 * そのときは飛ばす —— バイナリが新しくなれば、ここは自動で走り出す。
 */
async function post_export(page: Page, body: Record<string, unknown>): Promise<Record<string, unknown>> {
  // /api は Cookie ではなく本文の session_id で認証する（auth_middleware.go の sessionPeek）
  const [session] = (await page.context().cookies()).filter((cookie) => cookie.name === 'gkill_session_id')
  expect(session, 'gkill_session_id クッキーが無い（ログインできていない）').toBeDefined()

  const response = await page.context().request.post('/api/export_rykv_html', {
    data: { ...body, session_id: session.value },
  })
  const text = await response.text()
  test.skip(!text.trimStart().startsWith('{'), '出力の API を持たないビルド（PATH 上の gkill_server が古い）')

  const parsed = JSON.parse(text) as Record<string, unknown>
  const errors = (parsed.errors ?? []) as Array<{ error_code?: string }>
  // ガワが embed に無いビルドでは、この API だけが 500 を返す（ほかの機能は動く）
  test.skip(errors.some((e) => e.error_code === 'ERR000455'), '出力用のガワが embed に無いビルド')
  return parsed
}

test.describe('rykv HTML export', () => {
  test('出力した HTML は gkill を参照しない', async ({ page }) => {
    await loginAsAdmin(page)

    const body = await post_export(page, export_request())
    const errors = (body.errors ?? []) as Array<{ error_code?: string }>
    expect(errors, `出力に失敗した: ${JSON.stringify(errors)}`).toEqual([])
    const html = (body.html ?? '') as string
    expect(html.length, 'HTML が空').toBeGreaterThan(1000)

    // 焼き込みの JSON の中には /api/ も /files/ も現れうる（記録の file_url など）。
    // 見るのは「読み込み時に取りに行く形」だけ
    const fetching = [
      /<script[^>]+src=["'](?!data:)/i,
      /<link[^>]+rel=["']stylesheet["'][^>]+href=["'](?!data:)/i,
      /<img[^>]+src=["']\/(?:files|api)\//i,
      /serviceWorker\s*\.\s*register/,
    ]
    for (const pattern of fetching) {
      expect(html, `外部を取りに行く形が残っている: ${pattern}`).not.toMatch(pattern)
    }
  })

  test('通信を全部止めても画面が出る', async ({ page }) => {
    await loginAsAdmin(page)

    const body = await post_export(page, export_request())
    expect((body.errors ?? []) as Array<unknown>).toEqual([])

    // 地図（Google Maps）以外の通信を落とす。data: と about: は通す
    await page.route('**', (route) => {
      const url = route.request().url()
      if (url.startsWith('data:') || url.startsWith('about:') || url.startsWith('file:')) {
        route.continue()
        return
      }
      route.abort()
    })

    const console_errors: Array<string> = []
    page.on('pageerror', (error) => console_errors.push(error.message))

    await page.setContent(body.html as string, { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 30000 })

    // 読み込み中のまま止まらないこと（焼き込みの is_loaded / rykv_hot_reload の取り違えで固まる）
    const spinner = page.locator('.v-overlay .v-progress-circular').first()
    await expect(spinner, '読み込み中のまま止まっている').toBeHidden({ timeout: 30000 })

    expect(console_errors, `出力 HTML で例外が出ている: ${console_errors.join(' / ')}`).toEqual([])
  })

  test('メディアを埋め込まない指定で同名フォルダは選べない', async ({ page }) => {
    await loginAsAdmin(page)

    const body = await post_export(page, export_request((request_body) => {
      request_body.media_mode = 'none'
      request_body.output_format = 'sidecar'
    }))
    const errors = (body.errors ?? []) as Array<{ error_code?: string }>

    // 中身の無いフォルダを作らせない。弾くのは収集を始める前
    expect(errors.map((e) => e.error_code), '弾かれていない').toContain('ERR000454')
    expect(body.html ?? '', '弾いたのに HTML を返している').toBe('')
    expect(body.zip_base64 ?? '', '弾いたのに ZIP を返している').toBe('')
  })
})
