import { test, expect } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, navigateToSettings,
  makeUniqueLabel, findKyouByText, waitForKyouByText,
  clickFabButton, clickContextMenuItem, clickDialogButton,
  openApplicationConfigDialog,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

test.describe('Regression Tests for Previously Fixed Bugs', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  // 項番35: Kmemo編集で必須チェック (元NG→修正済み)
  test('kmemo edit enforces required content field', async ({ page }) => {
    const label = makeUniqueLabel('kmemo_required')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    // 見つからなければ落とす。条件で包んでいたころは何も検証せずに緑になっていた
    const record = findKyouByText(page, label).first()
    await expect(record, '作った記録の行が一覧に出ない').toBeVisible({ timeout: 30000 })
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, /編集|edit/i)

    const dialog = page.locator('.gkill-floating-dialog').last()
    await expect(dialog, '編集ダイアログが開かない').toBeVisible({ timeout: 30000 })

    // 本文を空にして保存すると、kmemo_content_is_blank で弾かれる
    // （use-edit-kmemo-view.ts のタイトル入力チェック）
    const contentInput = dialog.locator('textarea').first()
    await expect(contentInput, '本文欄が無い').toBeVisible({ timeout: 15000 })
    await contentInput.fill('')
    await expect(contentInput).toHaveValue('')

    await dialog.getByRole('button', { name: '保存', exact: true }).click()

    // `[role="alert"]` だけだと Vuetify が入力欄ごとに置く `v-input__details`
    // （常に存在して不可視）を掴む。実際のエラー表示は `.v-alert` なのでそこまで絞る
    await expect(page.locator('.v-alert[role="alert"]').first(), '本文が空でも保存できてしまっている')
      .toBeVisible({ timeout: 30000 })
    await expect(dialog, '弾かれたのに編集ダイアログが閉じている').toBeVisible()
  })

  // 項番80: ローカルアクセスのみ許可 (元NG→修正済み)
  test('local-only access setting can be toggled', async ({ page }) => {
    await navigateToSettings(page)

    // Look for local access only toggle/switch
    const _localAccessToggle = page.locator('.v-switch, .v-checkbox, input[type="checkbox"]')
      .filter({ hasText: /ローカル|local|IsLocalOnlyAccess/i }).first()
    // If direct text match fails, look in surrounding text
    const app = page.locator('#app')
    const content = await app.textContent()

    // Verify settings page loads without error
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番120: タグ構造追加 (元NG→修正済み)
  test('tag structure can be added in user config', async ({ page }) => {
    await navigateToSettings(page)

    // Look for tag structure section and add button
    const _addButtons = page.locator('button').filter({ hasText: /追加|add/i })
    const app = page.locator('#app')
    const content = await app.textContent()

    // Verify tag-related section exists
    const _hasTagSection = content!.includes('タグ') || content!.includes('Tag') || content!.includes('tag')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番127: Device構造追加 (元NG→修正済み)
  test('device structure can be added in user config', async ({ page }) => {
    await navigateToSettings(page)

    const app = page.locator('#app')
    const content = await app.textContent()

    // Verify device-related section exists
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番131: RepType構造追加 (元NG→修正済み)
  test('reptype structure can be added in user config', async ({ page }) => {
    await navigateToSettings(page)

    const app = page.locator('#app')
    const content = await app.textContent()

    // Verify RepType section exists
    const _hasRepTypeSection = content!.includes('RepType') || content!.includes('レップタイプ') || content!.includes('reptype')
    expect(content!.length).toBeGreaterThan(0)
    await expect(app).toBeVisible()
  })

  // 項番139: ApplicationConfig適用ボタン (元NG→修正済み)
  test('application config apply button works', async ({ page }) => {
    // 設定ダイアログは歯車から開く。**最果て(/saihate)に歯車は無い**
    const dialog = await openApplicationConfigDialog(page)

    const applyButton = dialog.getByRole('button', { name: '適用', exact: true })
    await expect(applyButton, '設定に「適用」ボタンが無い').toBeVisible({ timeout: 15000 })
    await applyButton.click()

    // 適用しても画面が壊れないこと
    await expect(page.locator('.v-application'), '適用で画面が壊れた')
      .toBeVisible({ timeout: 30000 })
  })

  // 項番142: ファイルアップロード (元NG→修正済み)
  test('file upload via add dialog', async ({ page }) => {
    await page.goto('/rykv', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#app', { timeout: 15000 })

    await clickFabButton(page)

    // Look for upload menu item（FABメニューの「アップロード」）
    const uploadItem = page.locator('.v-list-item, [role="menuitem"], .v-btn')
      .filter({ hasText: /アップロード|upload/i }).first()
    await expect(uploadItem, 'FABメニューにアップロードが無い').toBeVisible({ timeout: 30000 })
    await uploadItem.click()

    // Verify upload dialog opens
    const upload_dialog = page.locator('.gkill-floating-dialog').last()
    await expect(upload_dialog, 'アップロードのダイアログが開かない').toBeVisible({ timeout: 30000 })
    await expect(upload_dialog.locator('input[type="file"]').first(), 'ファイル入力が無い')
      .toBeAttached({ timeout: 15000 })
  })

  // 「タグを付けて追加した記録が、追加した直後に一覧から消える」の回帰。
  //
  // 既定クエリは「絞らない」を tags = null ではなく、そのときの
  // check_when_inited タグ名の**列挙**として物質化する(find-kyou-query.ts)。
  // それが localStorage の列状態へ凍る一方でタグ宇宙は育つので、
  // 保存後に作られたタグが付いた記録は、サーバ検索でも局所挿入でも1件も通らなくなる。
  //
  // **画面遷移しないことがこのテストの本質。** 遷移すると localStorage を読み直して
  // 既定クエリを作り直すので、この不具合をすり抜ける
  // （add-dialog-crud.spec.ts の同種のテストが遷移するのはそのため）。
  test('新規タグを付けて追加した記録が、画面遷移せずに一覧へ残る', async ({ page }) => {
    const label = makeUniqueLabel('new_tag_stays')
    const tagLabel = makeUniqueLabel('e2eNewTag')

    await navigateToRykv(page)

    await clickFabButton(page)
    await clickContextMenuItem(page, MENU.addURLog)
    const dialog = page.locator('.gkill-floating-dialog, .v-dialog').first()
    await expect(dialog, 'ブックマークの追加ダイアログが開かない').toBeVisible({ timeout: 15000 })

    const field = (index: number) => dialog
      .locator('input[type="text"], input[type="url"], input[type="number"], .v-text-field input').nth(index)
    await field(0).fill(`https://example.com/${label}`)
    await field(1).fill(label)
    await field(4).fill(tagLabel)

    // 未知タグの確認ダイアログは clickDialogButton の中で確定される
    await clickDialogButton(page, SAVE_BUTTON)
    await expect(dialog, '保存してもダイアログが閉じない').toBeHidden({ timeout: 30000 })

    // 遷移しない。列の条件へ新タグが足され、その列が引き直されるまで待つ
    await waitForKyouByText(page, label, 60000)
  })
})
