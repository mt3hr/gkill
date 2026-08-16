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
    // タスク化した行は元の記録と同じ本文で表示されるので、リポジトリ名で特定する
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*MiReKyou\s*$/ }), 'タスク化した行が作られていない')
      .toHaveCount(1, { timeout: 30000 })
    await expect(page.locator('.kyou_rep_name').filter({ hasText: /^\s*Kmemo\s*$/ }), '元のメモが作られていない')
      .toHaveCount(1, { timeout: 30000 })
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
