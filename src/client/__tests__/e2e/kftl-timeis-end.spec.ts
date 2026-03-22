import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToPlaing,
  makeUniqueLabel, pageContainsText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('KFTL TimeIs End Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番18: TimeIs終了(タイトル指定) — splitter: ーえ
  test('end timeis by title via KFTL', async ({ page }) => {
    // First, start a TimeIs
    const label = makeUniqueLabel('timeis_end_title')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaing(page)
    const started = await pageContainsText(page, label)
    expect(started).toBe(true)

    // End it by title
    await submitKftlText(page, `ーえ\n${label}`)
    // Verify page still renders correctly
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番19: TimeIs終了(タイトル存在すれば) — splitter: ーいえ
  test('end timeis by title if exists via KFTL', async ({ page }) => {
    // Start a TimeIs
    const label = makeUniqueLabel('timeis_end_ifexist')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaing(page)
    const started = await pageContainsText(page, label)
    expect(started).toBe(true)

    // End it with "if exists" — should succeed without error
    await submitKftlText(page, `ーいえ\n${label}`)
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()

    // Also test with non-existent title — should not cause error
    const nonExistent = makeUniqueLabel('timeis_noexist')
    await submitKftlText(page, `ーいえ\n${nonExistent}`)
    await navigateToRykv(page)
    await expect(app).toBeVisible()
  })

  // 項番20: TimeIs終了(タグ指定) — splitter: ーたえ
  test('end timeis by tag via KFTL', async ({ page }) => {
    // Start a TimeIs with a tag
    const label = makeUniqueLabel('timeis_end_tag')
    const tagName = makeUniqueLabel('endtag')
    await submitKftlText(page, `。${tagName}\nーた\n${label}`)
    await navigateToPlaing(page)
    const started = await pageContainsText(page, label)
    expect(started).toBe(true)

    // End all TimeIs with that tag
    await submitKftlText(page, `ーたえ\n${tagName}`)
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番21: TimeIs終了(タグ存在すれば) — splitter: ーいたえ (元NG→修正済み回帰テスト)
  test('end timeis by tag if exists via KFTL (regression)', async ({ page }) => {
    // Start a TimeIs with a tag
    const label = makeUniqueLabel('timeis_end_tagifexist')
    const tagName = makeUniqueLabel('endtagie')
    await submitKftlText(page, `。${tagName}\nーた\n${label}`)
    await navigateToPlaing(page)
    const started = await pageContainsText(page, label)
    expect(started).toBe(true)

    // End with "if tag exists" — should succeed
    await submitKftlText(page, `ーいたえ\n${tagName}`)
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()

    // Also test with non-existent tag — should not cause error
    const nonExistentTag = makeUniqueLabel('notag')
    await submitKftlText(page, `ーいたえ\n${nonExistentTag}`)
    await navigateToRykv(page)
    await expect(app).toBeVisible()
  })
})
