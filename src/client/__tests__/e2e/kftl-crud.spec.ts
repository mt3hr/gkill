import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing, makeUniqueLabel, pageContainsText, expectPageToContainText, searchByKeyword } from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('KFTL CRUD Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('submit kmemo via KFTL and verify in RYKV', async ({ page }) => {
    const label = makeUniqueLabel('kmemo_kftl')
    await submitKftlText(page, label)
    await navigateToRykv(page)
    await expectPageToContainText(page, label)
  })

  test('submit kmemo with tag via KFTL', async ({ page }) => {
    const label = makeUniqueLabel('kmemo_tag_kftl')
    const tagName = makeUniqueLabel('tag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)
    // Check for either the kmemo content or the tag name on the page
    await expect.poll(async () => await pageContainsText(page, label) || await pageContainsText(page, tagName), { timeout: 30000 }).toBe(true)
  })

  test('submit lantana via KFTL', async ({ page }) => {
    // Lantana is mood value, doesn't have searchable text — just verify no error
    await submitKftlText(page, 'ーら\n7')
    // Navigate to rykv and verify page loads without error
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('submit mi via KFTL and verify in Mi board', async ({ page }) => {
    const label = makeUniqueLabel('mi_kftl')
    await submitKftlText(page, `ーみ\n${label}`)
    await navigateToMi(page)
    await expectPageToContainText(page, label)
  })

  test('submit mirekyou via KFTL and verify the memo is tasked', async ({ page }) => {
    // 「～～」で開いて閉じるブロック。板名は空にして既定の板へ落とすので、
    // 新しい板名の確認ダイアログを踏まない（submitKftlText はタグの確認しか通さない）
    const label = makeUniqueLabel('mirekyou_kftl')
    await submitKftlText(page, `${label}\n～～\n～～`)
    await navigateToRykv(page)
    await searchByKeyword(page, label)

    // rykv の一覧は行高 180px で、is_row_height の閾値(120)より大きいので is_compact にならない。
    // つまりタスク化した行は参照先のKyouを行の中に描くので、リポジトリ名のバッジは
    // 「タスク化した行に2つ（自分と参照先のメモ）＋元のメモの行に1つ」出る。
    // ページ全体で .kyou_rep_name を数えるとメモが2つに見えるため、行（.kyou_in_list）単位で数える
    const rows = page.locator('.kyou_in_list')
    await expect(rows, '元のメモとタスク化した行の2件にならない').toHaveCount(2, { timeout: 30000 })

    // タスク化した行は元の記録と同じ本文で表示されるので、参照先ブロックの有無で見分ける
    const taskedRow = rows.filter({ has: page.locator('.mirekyou_target') })
    await expect(taskedRow, 'タスク化した行が作られていない').toHaveCount(1)
    await expect(taskedRow.locator('.kyou_rep_name').first(), 'タスク化した行のリポジトリ名が違う')
      .toHaveText(/^\s*MiReKyou\s*$/)
    await expect(taskedRow.locator('.mirekyou_target .kyou_rep_name'), 'タスク化した行に参照先のメモが描かれていない')
      .toHaveText(/^\s*Kmemo\s*$/)

    const memoRow = rows.filter({ hasNot: page.locator('.mirekyou_target') })
    await expect(memoRow, '元のメモが作られていない').toHaveCount(1)
    await expect(memoRow.locator('.kyou_rep_name'), '元のメモのリポジトリ名が違う').toHaveText(/^\s*Kmemo\s*$/)
  })

  test('submit timeis start via KFTL', async ({ page }) => {
    const label = makeUniqueLabel('timeis_kftl')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaing(page)
    await expectPageToContainText(page, label)
  })

  test('submit nlog via KFTL', async ({ page }) => {
    // Nlog: amount and shop name
    // ーん の後は 店名 → 品目 → 金額 の3行 (kftl-nlog-*-statement-line.ts)
    await submitKftlText(page, 'ーん\nテスト店舗_kftl\nテスト品目\n999')
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 支払い(品名と金額のペア)1組ごとに1件の記録になり、金額の行のあとに書いたタグは
  // その支払いだけに付く。以前は Nlog の id とタグの target_id が食い違っていて、
  // エラーも警告も出ないままタグが1件も付いていなかった
  test('submit two nlog payments with their own tags via KFTL', async ({ page }) => {
    const marker = makeUniqueLabel('nlog_pair')
    const firstTag = `${marker}_tag1`
    const secondTag = `${marker}_tag2`
    await submitKftlText(page, `ーん\nテスト店舗_kftl\n${marker}_a\n150\n。${firstTag}\n${marker}_b\n120\n。${secondTag}`)

    await navigateToRykv(page)
    await searchByKeyword(page, marker)

    const taggedWithFirst = page.locator('.kyou_attached_tags').filter({ hasText: firstTag })
    const taggedWithSecond = page.locator('.kyou_attached_tags').filter({ hasText: secondTag })
    await expect(taggedWithFirst, '1件目の支払いにタグが付いていない').toHaveCount(1, { timeout: 30000 })
    await expect(taggedWithSecond, '2件目の支払いにタグが付いていない').toHaveCount(1, { timeout: 30000 })
    // 同じ行に両方のタグが乗っていたら、支払いごとに分けられていない
    await expect(taggedWithFirst.filter({ hasText: secondTag }), '2つの支払いのタグが同じ行に付いている').toHaveCount(0)
  })

  test('submit urlog via KFTL', async ({ page }) => {
    const label = makeUniqueLabel('urlog_kftl')
    await submitKftlText(page, `ーう\nhttps://example.com/${label}`)
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  test('submit multiple records via KFTL split', async ({ page }) => {
    const label1 = makeUniqueLabel('split1')
    const label2 = makeUniqueLabel('split2')
    await submitKftlText(page, `${label1}\n、\n${label2}`)
    await navigateToRykv(page)
    await expectPageToContainText(page, label1)
    await expectPageToContainText(page, label2)
  })
})
