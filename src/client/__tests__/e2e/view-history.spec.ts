import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv,
  makeUniqueLabel, expectPageToContainText, findKyouByText,
  openContextMenu, clickContextMenuItem, dismissFloatingDialogs,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

/**
 * 本文で行を特定して「履歴」を開く。
 *
 * 以前は boolean を返し、呼び出し側が `if (opened)` で包んでいたため、
 * 行やメニューが見つからないと**何も検証せずに緑**になっていた。
 * 見つからなければここで落ちるようにしてある。
 */
async function openHistoryFor(page: Page, text: string): Promise<void> {
  const record = findKyouByText(page, text).first()
  await expect(record, `「${text}」の行が一覧に出ない`).toBeVisible({ timeout: 30000 })
  await dismissFloatingDialogs(page)
  await record.click({ button: 'right', force: true })
  await clickContextMenuItem(page, /履歴|histor/i)
  await expect(page.locator('.gkill-floating-dialog').last(), '履歴ダイアログが開かない')
    .toBeVisible({ timeout: 30000 })
}

/** 本文で行を特定してリポスト（ReKyou 作成）のダイアログを開く */
async function repostRecord(page: Page, text: string): Promise<void> {
  const record = findKyouByText(page, text).first()
  await expect(record, `「${text}」の行が一覧に出ない`).toBeVisible({ timeout: 30000 })
  await dismissFloatingDialogs(page)
  await record.click({ button: 'right', force: true })
  await clickContextMenuItem(page, /リキョウ|リポスト|repost/i)
  await expect(page.locator('.gkill-floating-dialog').last(), 'リポストのダイアログが開かない')
    .toBeVisible({ timeout: 30000 })
}

test.describe('View/Browse History Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番57: Lantana閲覧+履歴+リポスト+スクロールバー確認
  test('view lantana with history and repost', async ({ page }) => {
    // Create a Lantana
    await submitKftlText(page, 'ーら\n5')
    await navigateToRykv(page)

    // Verify page renders without unnecessary scrollbars
    const app = page.locator('#app')
    await expect(app).toBeVisible()
    const content = await app.innerHTML()
    expect(content.length).toBeGreaterThan(100)
  })

  // 項番58: Mi閲覧+履歴+リポスト
  test('view mi with history and repost', async ({ page }) => {
    const label = makeUniqueLabel('mi_view_hist')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToRykv(page)

    await openHistoryFor(page, label)

    await navigateToRykv(page)
    await repostRecord(page, label)
  })

  // 項番59: Nlog閲覧+履歴+リポスト
  test('view nlog with history and repost', async ({ page }) => {
    const shopName = makeUniqueLabel('nlog_view_shop')
    // ーん の後は 店名 → 品目 → 金額 の3行 (kftl-nlog-*-statement-line.ts)
    await submitKftlText(page, `ーん\n${shopName}\nテスト品目\n500`)
    await navigateToRykv(page)

    await openHistoryFor(page, shopName)

    await navigateToRykv(page)
    await repostRecord(page, shopName)
  })

  // 項番61: URLog閲覧+履歴+NoImage確認
  test('view urlog with history and NoImage fallback', async ({ page }) => {
    const label = makeUniqueLabel('urlog_view_hist')
    await submitKftlText(page, `ーう\nhttps://example.com/${label}\n${label}`)
    await navigateToRykv(page)

    // Verify the record appears
    await expectPageToContainText(page, label)

    // NoImage フォールバックを含め、描かれた画像の src が欠けていないこと。
    // 画像が1枚も無い状態は正常なので、枚数そのものは条件にしない
    const srcs = await page.locator('#app img').evaluateAll(
      (nodes) => nodes.slice(0, 5).map((n) => (n as HTMLImageElement).getAttribute('src')))
    expect(srcs.every((src) => src !== null && src !== ''), '画像の src が空になっている').toBe(true)

    await openHistoryFor(page, label)
  })

  // 項番63: ReKyou閲覧+履歴
  test('view rekyou with history', async ({ page }) => {
    const label = makeUniqueLabel('rekyou_view_hist')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // Create a repost
    await repostRecord(page, label)

    // Navigate and verify repost is visible
    await navigateToRykv(page)
    await expectPageToContainText(page, label)

    await openHistoryFor(page, label)
  })

  // 項番64: Tag閲覧+履歴+レイアウト確認
  test('view tag with history and layout', async ({ page }) => {
    const label = makeUniqueLabel('tag_view_hist')
    const tagName = makeUniqueLabel('viewtag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)

    // タグは行の中に .tag / .highlighted_tag として描かれる（attached-tag.vue）
    const tagElement = page.locator('.tag, .highlighted_tag').filter({ hasText: tagName }).first()
    await expect(tagElement, `タグ「${tagName}」が表示されない`).toBeVisible({ timeout: 30000 })

    await openContextMenu(page, `.tag:has-text("${tagName}"), .highlighted_tag:has-text("${tagName}")`)
    await clickContextMenuItem(page, /履歴|histor/i)
    await expect(page.locator('.gkill-floating-dialog').last(), 'タグの履歴ダイアログが開かない')
      .toBeVisible({ timeout: 30000 })
  })

  // 項番65: Text閲覧+履歴
  test('view text with history', async ({ page }) => {
    const label = makeUniqueLabel('text_view_hist')
    const text_body = makeUniqueLabel('テスト閲覧テキスト')
    // **付随テキストの書式は `ーー` で囲むブロック**（kftl_factory.go の splitterStartText）。
    // もとは `、、` を使っていたが、あれは splitterSplitNextSecond ＝
    // 「記録を分けて時刻を1秒進める」で、テキストは1つも作られず**メモが2件できるだけ**だった。
    // 以前のテストは `if (count > 0)` で包んで最後に `expect(app).toBeVisible()` を見ていたので、
    // テキストが存在しなくても緑になっていた。
    // 書式は kftl_statement_test.go:527 の "メモ内容 / ーー / テキスト本文 / ーー" と同じ
    await submitKftlText(page, `${label}\nーー\n${text_body}\nーー`)
    await navigateToRykv(page)

    // **付随テキストは一覧の行には出ない。** kyou-list-view.vue は
    // `:show_attached_texts="false"` で KyouView へ渡すので、テキストが描かれるのは
    // 詳細ペイン（rykv-view.vue の `:show_attached_texts="true"`）だけ。
    // 詳細ペインは既定で閉じている（use-rykv-view.ts の is_show_kyou_detail_view = false）ので、
    // アプリバーの mdi-file-document で開いてから、行をクリックしてフォーカスを当てる。
    // （付随タグのほうが一覧に出るのは show_tags_in_list が既定 true だから）
    const record = findKyouByText(page, label).first()
    await expect(record, '作った記録の行が一覧に出ない').toBeVisible({ timeout: 30000 })

    // 詳細ペインは `is_show_kyou_detail_view && focused_kyou` の両方が要る。
    // **先に行をクリックしてフォーカスを当ててから**アプリバーの mdi-file-document で開く
    await record.click()
    await page.locator('button:has(.mdi-file-document)').first().click()
    const detailPane = page.locator('.rykv_kyou_detail_view_wrap')
    await expect(detailPane, '詳細ペインが開かない').toBeVisible({ timeout: 30000 })

    // 一覧側の要素を掴まないよう詳細ペインの中だけを見る
    const textElement = detailPane.locator('.text, .highlighted_text').filter({ hasText: text_body }).first()
    await expect(textElement, '詳細ペインに付随テキストが表示されない').toBeVisible({ timeout: 30000 })

    await openContextMenu(page, `.text:has-text("${text_body}"), .highlighted_text:has-text("${text_body}")`)
    await clickContextMenuItem(page, /履歴|histor/i)
    await expect(page.locator('.gkill-floating-dialog').last(), 'テキストの履歴ダイアログが開かない')
      .toBeVisible({ timeout: 30000 })
  })
})
