import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { checkGkillServer } from './check-server'
import { loginAsAdmin } from './helpers'

/**
 * タスク画面（mi）の「HTML で出力」。
 *
 * 画面に依らない約束（通信をしない・外部を取りに行く形が残っていない）は
 * kyou-export-html.spec.ts が見ている。ここで見るのは **mi として出したときに
 * mi として開けるか** ―― 焼き込みの印でマウント先が選ばれ、板の見出しが出て、
 * 読み込み中のまま止まらないこと。
 *
 * 印が落ちると「mi を出力したのに rykv として開く」になり、例外もエラーも出ない。
 * 板ツリーを組み忘れると板を1つも選べない。どちらもここでしか気づけない。
 *
 * ダイアログの操作は通らない（共有フッタの表示は利用者の設定で、E2E から変えると
 * 他の spec に影響する）。導線はユニット（export-kyou-html-view.test.ts）が見ている。
 */

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

/**
 * mi の出力の要求。
 *
 * **サイドバーが生成するのと同じ形で送ること**（`for_mi` と `include_*_mi`）。
 * 欠けた条件を焼くと、出力先のサイドバーが初めに生成した条件と食い違った瞬間に
 * その板が空になる。板名は null（「すべて」）にして、作った記録が必ず入るようにする。
 */
function mi_export_request(customize?: (body: Record<string, unknown>) => void): Record<string, unknown> {
  const body: Record<string, unknown> = {
    locale_name: 'ja',
    screen: 'mi',
    title: 'E2E の mi 出力',
    columns: [{
      query_id: 'e2e-mi-export-column',
      for_mi: true,
      mi_board_name: null,
      include_create_mi: true,
      include_check_mi: false,
      include_limit_mi: false,
      include_start_mi: false,
      include_end_mi: false,
    }],
    include_attached_tags: true,
    include_attached_texts: true,
    include_attached_notifications: true,
    include_attached_timeiss: false,
    include_gps_log: false,
    include_plugin_content: false,
    // mi に集計とマップは無い。ダイアログも選択肢を出さない
    include_dnote: false,
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
 * そのバイナリが古いとこの API や画面の印を持たない。そのときは飛ばす。
 */
async function post_export(
  page: Page,
  body: Record<string, unknown>,
  options?: { expect_rejected?: boolean },
): Promise<Record<string, unknown>> {
  // /api は Cookie ではなく本文の session_id で認証する（auth_middleware.go の sessionPeek）
  const [session] = (await page.context().cookies()).filter((cookie) => cookie.name === 'gkill_session_id')
  expect(session, 'gkill_session_id クッキーが無い（ログインできていない）').toBeDefined()

  const response = await page.context().request.post('/api/export_kyou_html', {
    data: { ...body, session_id: session.value },
    timeout: 120000,
  })
  const text = await response.text()
  test.skip(!text.trimStart().startsWith('{'), '出力の API を持たないビルド（PATH 上の gkill_server が古い）')

  const parsed = JSON.parse(text) as Record<string, unknown>
  const errors = (parsed.errors ?? []) as Array<{ error_code?: string }>
  // ガワが embed に無いビルドでは、この API だけが 500 を返す（ほかの機能は動く）
  test.skip(errors.some((e) => e.error_code === 'ERR000455'), '出力用のガワが embed に無いビルド')
  // 画面の印を知らない古いバイナリは、mi の指定を未知の指定として弾く。
  // 弾かれることを確かめたいテスト（expect_rejected）では飛ばさない
  if (!options?.expect_rejected) {
    test.skip(errors.some((e) => e.error_code === 'ERR000454'),
      '画面の印を持たないビルド（PATH 上の gkill_server が古い）')
  }
  return parsed
}

test.describe('mi HTML export', () => {
  // 出力 1 回で数十秒かかることがある
  test.setTimeout(180000)

  test('焼き込みに mi の印が乗る', async ({ page }) => {
    await loginAsAdmin(page)

    const body = await post_export(page, mi_export_request())
    const errors = (body.errors ?? []) as Array<{ error_code?: string }>
    expect(errors, `出力に失敗した: ${JSON.stringify(errors)}`).toEqual([])

    const html = (body.html ?? '') as string
    expect(html.length, 'HTML が空').toBeGreaterThan(1000)
    // 焼き込みの JSON の先頭近くに印が入る。出力先のページはこれだけを見てマウント先を選ぶ
    expect(html, '画面の印が焼き込みに無い').toContain('"screen":"mi"')
  })

  test('file:// で開くと板として読める', async ({ page }) => {
    await loginAsAdmin(page)

    // **ここで記録を作らない。** E2E は共有のホームを使うので、mi のタスクを1件増やすと
    // 他の spec の一覧が仮想スクロールの窓から溢れる（`waitForKyouByText` は
    // 描かれている行だけを掴むので、末尾に並んだ新しいタスクが見えなくなる）。
    // この spec が見るのは「焼き込みの印で mi として開けるか」だけで、板の中身には依らない
    // （板の見出しは列の条件から出るので、空の板でも出る）
    const body = await post_export(page, mi_export_request())
    expect((body.errors ?? []) as Array<unknown>).toEqual([])

    // **リポジトリの中へ書かないこと。** Vite の依存スキャナが *.html をエントリとして
    // 拾い、次回以降の E2E でビルドが落ちる。OS の一時領域へ出して、終わったら消す
    const saved_dir = mkdtempSync(join(tmpdir(), 'gkill-mi-export-e2e-'))
    const saved = join(saved_dir, 'gkill_mi_export.html')
    writeFileSync(saved, body.html as string, 'utf-8')
    const saved_url = pathToFileURL(saved).href

    // 約束は「通信をしない」。この要求は include_map: false なので出てよい先は1つも無い
    const network_attempts: Array<string> = []
    await page.route(/^https?:\/\//, (route) => {
      network_attempts.push(route.request().url())
      route.abort()
    })

    const page_errors: Array<string> = []
    page.on('pageerror', (error) => page_errors.push(error.message))

    await page.goto(saved_url, { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app > *', { timeout: 30000 })

    // 読み込み中のまま止まらないこと。
    // 板ツリーを組む前に画面を描くと、サイドバーの各節が「チェック0個」を生成して
    // 列が空になる。サイドバーの入れ物を v-if で消すと列が1本も作られず、ここが回り続ける
    const spinner = page.locator('.v-overlay .v-progress-circular').first()
    await expect(spinner, '読み込み中のまま止まっている').toBeHidden({ timeout: 30000 })

    // mi として開けていること。板の見出しは mi の列にしか無いので、
    // 印が落ちて rykv として開いたらここで落ちる
    await expect(page.locator('.mi_view_wrap'), 'mi として開いていない').toBeAttached({ timeout: 30000 })
    await expect(page.locator('.mi_board_column_title').first(), '板の見出しが出ていない')
      .toBeVisible({ timeout: 30000 })

    expect(page.url(), '出力 HTML が自分以外の URL へ移っている').toBe(saved_url)
    expect(page_errors, `出力 HTML で例外が出ている: ${page_errors.join(' / ')}`).toEqual([])
    expect(network_attempts, `出力 HTML が通信している: ${network_attempts.join(' / ')}`).toEqual([])

    // 開いたままだとファイルを掴んだままになるので、先に離してから消す
    await page.goto('about:blank')
    rmSync(saved_dir, { recursive: true, force: true })
  })

  test('未知の画面は弾く', async ({ page }) => {
    await loginAsAdmin(page)

    const body = await post_export(page, mi_export_request((request_body) => {
      request_body.screen = 'dashboard'
    }), { expect_rejected: true })
    const errors = (body.errors ?? []) as Array<{ error_code?: string }>

    // 通してしまうと「出した画面と開く画面が食い違う」が静かに起きる
    expect(errors.map((e) => e.error_code), '弾かれていない').toContain('ERR000454')
    expect(body.html ?? '', '弾いたのに HTML を返している').toBe('')
  })
})
