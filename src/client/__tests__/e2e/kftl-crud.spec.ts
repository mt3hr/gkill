import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { submitKftlText, navigateToRykv, navigateToMi, navigateToPlaing, makeUniqueLabel, pageContainsText } from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
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
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)
  })

  test('submit kmemo with tag via KFTL', async ({ page }) => {
    const label = makeUniqueLabel('kmemo_tag_kftl')
    const tagName = makeUniqueLabel('tag')
    await submitKftlText(page, `。${tagName}\n${label}`)
    await navigateToRykv(page)
    // Check for either the kmemo content or the tag name on the page
    const foundLabel = await pageContainsText(page, label)
    const foundTag = await pageContainsText(page, tagName)
    expect(foundLabel || foundTag).toBe(true)
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
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)
  })

  test('submit timeis start via KFTL', async ({ page }) => {
    const label = makeUniqueLabel('timeis_kftl')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaing(page)
    const found = await pageContainsText(page, label)
    expect(found).toBe(true)
  })

  test('submit nlog via KFTL', async ({ page }) => {
    // Nlog: amount and shop name
    await submitKftlText(page, 'ーん\n999\nテスト店舗_kftl')
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
    const found1 = await pageContainsText(page, label1)
    const found2 = await pageContainsText(page, label2)
    expect(found1).toBe(true)
    expect(found2).toBe(true)
  })
})
