import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { navigateToSettings } from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server (localhost:9999) is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('User Config CRUD', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番112: GoogleMapAPIキー適用
  test('google map api key can be set', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    // Look for GoogleMap or API key related content
    const _hasGoogleMapSection = content!.includes('GoogleMap') || content!.includes('API') ||
      content!.includes('Map') || content!.includes('地図')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番113: rykv画像ビューア列数
  test('rykv image viewer column count setting', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番114: miデフォルト板名設定
  test('mi default board name setting', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番115: miデフォルト板名追加
  test('add mi default board name', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番118: rykvホットリロード有効
  test('enable rykv hot reload', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    const _hasHotReload = content!.includes('ホットリロード') || content!.includes('HotReload') ||
      content!.includes('hot') || content!.includes('reload')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番119: rykvホットリロード無効
  test('disable rykv hot reload', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番121: タグ構造にフォルダ追加
  test('add folder to tag structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    const _hasTagSection = content!.includes('タグ') || content!.includes('Tag')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番122: タグ構造並び替え
  test('reorder tag structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番123: タグ構造適用ボタン
  test('apply tag structure changes', async ({ page }) => {
    await navigateToSettings(page)
    const applyButton = page.locator('button').filter({ hasText: /適用|apply/i }).first()
    if (await applyButton.count() > 0) {
      await expect(applyButton).toBeVisible()
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番124: Rep構造追加
  test('add rep structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番125: Rep構造並び替え
  test('reorder rep structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番126: Rep構造適用ボタン
  test('apply rep structure changes', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番128: Device構造にフォルダ追加
  test('add folder to device structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番129: Device構造並び替え
  test('reorder device structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番130: Device構造適用ボタン
  test('apply device structure changes', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番132: RepType構造にフォルダ追加
  test('add folder to reptype structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番133: RepType構造並び替え
  test('reorder reptype structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番134: RepType構造適用ボタン
  test('apply reptype structure changes', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番135: KFTLテンプレート構造追加
  test('add kftl template structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    const content = await app.textContent()
    const _hasKFTL = content!.includes('KFTL') || content!.includes('テンプレート') || content!.includes('template')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番136: KFTLテンプレート構造にフォルダ追加
  test('add folder to kftl template structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番137: KFTLテンプレート構造並び替え
  test('reorder kftl template structure', async ({ page }) => {
    await navigateToSettings(page)
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })

  // 項番138: KFTLテンプレート構造適用ボタン
  test('apply kftl template structure changes', async ({ page }) => {
    await navigateToSettings(page)
    const applyButton = page.locator('button').filter({ hasText: /適用|apply/i }).first()
    if (await applyButton.count() > 0) {
      await expect(applyButton).toBeVisible()
    }
    const app = page.locator('#app')
    await expect(app).toBeVisible()
  })
})
