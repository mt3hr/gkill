import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToPlaying,
  makeUniqueLabel, expectPageToContainText, expectPageNotToContainText,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

// 終了系（ーえ / ーいえ / ーたえ / ーいたえ）は「エラーが出ない」だけでなく、
// 終了したあと実行中画面からその打刻が**消えている**ことまで見る。
// 2026-09-16 まではエラーの有無しか見ておらず、終了対象の選び方が不定順で
// いま走っている打刻が終わらない不具合（ADR-0509）を素通ししていた。
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
    await navigateToPlaying(page)
    await expectPageToContainText(page, label)

    // End it by title
    await submitKftlText(page, `ーえ\n${label}`)
    // 終了した打刻は実行中画面から消える
    await navigateToPlaying(page)
    await expectPageNotToContainText(page, label)
    await navigateToRykv(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番19: TimeIs終了(タイトル存在すれば) — splitter: ーいえ
  test('end timeis by title if exists via KFTL', async ({ page }) => {
    // Start a TimeIs
    const label = makeUniqueLabel('timeis_end_ifexist')
    await submitKftlText(page, `ーた\n${label}`)
    await navigateToPlaying(page)
    await expectPageToContainText(page, label)

    // End it with "if exists" — should succeed without error
    await submitKftlText(page, `ーいえ\n${label}`)
    // 終了した打刻は実行中画面から消える
    await navigateToPlaying(page)
    await expectPageNotToContainText(page, label)
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
    await navigateToPlaying(page)
    await expectPageToContainText(page, label)

    // End the running TimeIs with that tag
    await submitKftlText(page, `ーたえ\n${tagName}`)
    // 終了した打刻は実行中画面から消える
    await navigateToPlaying(page)
    await expectPageNotToContainText(page, label)
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
    await navigateToPlaying(page)
    await expectPageToContainText(page, label)

    // End with "if tag exists" — should succeed
    await submitKftlText(page, `ーいたえ\n${tagName}`)
    // 終了した打刻は実行中画面から消える
    await navigateToPlaying(page)
    await expectPageNotToContainText(page, label)
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
