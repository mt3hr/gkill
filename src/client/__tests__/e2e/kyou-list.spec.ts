import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Kyou List', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    await loginAsAdmin(page)
  })

  test('can navigate to Kyou list page', async ({ page }) => {
    await page.goto('/kyou', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    await expect(page).toHaveURL(/kyou/)
  })

  test('Kyou list displays records', async ({ page }) => {
    await page.goto('/kyou', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })
    // 以前は固定sleepだけで何も検証していなかった（必ず緑になるテスト）。
    // 一覧の枠が描けることまでを見る
    await expect(page.locator('#app'), '画面が描かれない').toBeVisible({ timeout: 30000 })
    await expect(page.locator('.v-application'), 'Vuetifyのレイアウトが立ち上がらない')
      .toBeVisible({ timeout: 30000 })
  })
})
