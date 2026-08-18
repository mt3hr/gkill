import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { dismissFloatingDialogs, makeUniqueLabel } from './crud-helpers'

/**
 * メモ帳のタブ。
 *
 * タブの状態は localStorage（キー `kftl_tabs`）に入る。Playwright は
 * storageState をコンテキスト生成時に流し込むだけなので、テストごとに
 * 「タブ1枚・中身なし」から始まる。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const TAB = '.kftl_tab'
const TAB_CLOSE = '.kftl_tab .kftl_tab_close'
const TEXT_AREA = 'textarea.kftl_text_area'
const FLOATING_DIALOG = '.gkill-floating-dialog'

/**
 * /kftl を開き、入力できる状態（設定の読み込み完了）まで待つ。
 *
 * チュートリアルダイアログが残っていると `.gkill-floating-dialog` の件数が
 * 1 から始まってしまい、「確認ダイアログが出ていないこと」を見られない
 */
async function openKftl(page: Page): Promise<void> {
  await page.goto('/kftl', { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#app', { timeout: 15000 })
  await expect(page.locator(TEXT_AREA)).toBeVisible({ timeout: 90000 })
  const saveButton = page.locator('button').filter({ hasText: /保存|送信|submit|save/i }).first()
  await expect(saveButton).toBeEnabled({ timeout: 30000 })
  await dismissFloatingDialogs(page)
  await expect(page.locator(FLOATING_DIALOG)).toHaveCount(0)
}

async function addTab(page: Page): Promise<void> {
  const before = await page.locator(TAB).count()
  await page.locator('.kftl_tab_add').first().click()
  await expect(page.locator(TAB)).toHaveCount(before + 1)
}

test.describe('KFTL Tabs', () => {
  test.beforeEach(async ({ page }) => {
    // 設定を読み終えるまで入力欄は readonly なので、どのテストもAPIが要る
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  test('+ でタブが増え、タブごとに別々の内容を持つ', async ({ page }) => {
    await openKftl(page)
    await expect(page.locator(TAB)).toHaveCount(1)

    const first = makeUniqueLabel('tab_first')
    const second = makeUniqueLabel('tab_second')

    const textarea = page.locator(TEXT_AREA)
    await textarea.fill(first)
    await expect(textarea).toHaveValue(first)

    await addTab(page)
    // 新しいタブは空で始まる
    await expect(textarea).toHaveValue('')
    await textarea.fill(second)
    await expect(textarea).toHaveValue(second)

    // 1枚目へ戻すと元の内容が残っている
    await page.locator(TAB).first().click()
    await expect(textarea).toHaveValue(first)
  })

  // タブ列はタイトル行に同居させている。title_height が実寸より大きいと、
  // タイトルとテキストエリアのあいだに何も無い帯ができる
  test('タイトル行とテキストエリアのあいだに余白ができない', async ({ page }) => {
    await openKftl(page)

    const title_box = await page.locator('.kftl_title').boundingBox()
    const tabs_box = await page.locator('.kftl_title .v-tabs').boundingBox()
    const textarea_box = await page.locator(TEXT_AREA).boundingBox()
    expect(title_box).not.toBeNull()
    expect(tabs_box).not.toBeNull()
    expect(textarea_box).not.toBeNull()

    const gap = textarea_box!.y - (title_box!.y + title_box!.height)
    expect(gap, 'タイトルとテキストエリアのあいだに帯ができている').toBeLessThanOrEqual(8)
    expect(title_box!.height - tabs_box!.height, 'タイトル行がタブ列より背が高すぎる').toBeLessThanOrEqual(8)
  })

  test('タブ名は本文の1行目になる', async ({ page }) => {
    await openKftl(page)
    await page.locator(TEXT_AREA).fill('買い物メモ\n2行目')
    await expect(page.locator(TAB).first()).toHaveText(/買い物メモ/)
  })

  test('空のタブは確認なしで閉じる', async ({ page }) => {
    await openKftl(page)
    await addTab(page)
    await expect(page.locator(TAB)).toHaveCount(2)

    await page.locator(TAB_CLOSE).last().click()

    await expect(page.locator(TAB)).toHaveCount(1)
    await expect(page.locator(FLOATING_DIALOG)).toHaveCount(0)
  })

  test('内容が残っているタブを閉じるときは確認する', async ({ page }) => {
    await openKftl(page)
    await addTab(page)
    await page.locator(TEXT_AREA).fill(makeUniqueLabel('closing'))

    await page.locator(TAB_CLOSE).last().click()

    const dialog = page.locator(FLOATING_DIALOG)
    await expect(dialog).toBeVisible()
    // キャンセルするとタブは残る
    await dialog.locator('button').filter({ hasText: /キャンセル|cancel/i }).first().click()
    await expect(page.locator(TAB)).toHaveCount(2)

    // もう一度閉じて、今度は確定する
    await page.locator(TAB_CLOSE).last().click()
    await expect(dialog).toBeVisible()
    await dialog.locator('button').filter({ hasText: /タブを閉じる|close tab/i }).first().click()
    await expect(page.locator(TAB)).toHaveCount(1)
  })

  test('保存すると そのタブだけ閉じ、他のタブは残る', async ({ page }) => {
    await openKftl(page)

    const keep = makeUniqueLabel('tab_keep')
    const textarea = page.locator(TEXT_AREA)
    await textarea.fill(keep)
    await expect(textarea).toHaveValue(keep)

    await addTab(page)
    const saved = makeUniqueLabel('tab_saved')
    await textarea.fill(saved)
    await expect(textarea).toHaveValue(saved)

    await page.locator('button').filter({ hasText: /^\s*保存\s*$/ }).first().click()

    await expect(page.locator(TAB)).toHaveCount(1)
    await expect(textarea).toHaveValue(keep)
  })

  // 保存マーカー（「！」だけの行）で終わる本文を打つと、保存ボタンを押さなくても送信される。
  //
  // マーカーの部分は `fill()` ではなく実際の打鍵で入れること。`fill()` の合成 input は
  // Vue の v-model 側だけが先に走り、`@input`（＝自動送信の印を立てる onTextAreaInput）が
  // watch より後に着地するので、利用者の打鍵を再現できない
  test('保存マーカーで終わる本文を打つと自動で保存される', async ({ page }) => {
    await openKftl(page)

    const keep = makeUniqueLabel('marker_keep')
    const textarea = page.locator(TEXT_AREA)
    await textarea.fill(keep)
    await expect(textarea).toHaveValue(keep)

    await addTab(page)
    await textarea.fill(makeUniqueLabel('marker_saved'))
    await textarea.press('End')
    await textarea.pressSequentially('\n！\n')

    // 保存ボタンは押していないが、保存できたタブは閉じる
    await expect(page.locator(TAB)).toHaveCount(1)
    await expect(textarea).toHaveValue(keep)
  })

  // IMEで打ったときの回帰。実機で「順当にIMEから入力すると保存が走らないのに、
  // バックスペースを押すと走る」と報告された形。
  //
  // IMEでは「変換の確定」と「改行」が別のEnterになる。確定は compositionend 経由で
  // Vue が合成した input として着地するので、DOMイベントの並びが素の打鍵と変わる。
  // このとき**同じ input イベントのリスナーとリスナーの間でマイクロタスクが走る**ため、
  // Vue の post flush（本文の watch）が `@input` ハンドラより先に新しい本文を観測する。
  // 判定や基準を watch 側に置いていると、ここで黙って落ちる。
  //
  // `pressSequentially` では再現しない（打鍵ごとにイベントループが回るので中間の
  // 本文を必ず観測してしまう）。CDPで実際にIME合成を起こすこと。
  test('IMEで確定してから改行しても自動で保存される', async ({ page }) => {
    await openKftl(page)

    const keep = makeUniqueLabel('ime_keep')
    const textarea = page.locator(TEXT_AREA)
    await textarea.fill(keep)
    await expect(textarea).toHaveValue(keep)

    await addTab(page)
    await textarea.fill('てすと')
    await textarea.press('End')
    await textarea.focus()

    const client = await page.context().newCDPSession(page)
    await page.keyboard.press('Enter')
    // 「！」をIMEの変換中の文字として置き、確定させる
    await client.send('Input.imeSetComposition', { text: '！', selectionStart: 1, selectionEnd: 1 })
    await client.send('Input.insertText', { text: '！' })
    await page.keyboard.press('Enter')

    // 保存ボタンは押していないが、保存できたタブは閉じる
    await expect(page.locator(TAB)).toHaveCount(1)
    await expect(textarea).toHaveValue(keep)
  })

  test('リロードしてもタブが復元される', async ({ page }) => {
    await openKftl(page)
    const first = makeUniqueLabel('reload_first')
    const second = makeUniqueLabel('reload_second')

    const textarea = page.locator(TEXT_AREA)
    await textarea.fill(first)
    await addTab(page)
    await textarea.fill(second)
    await expect(textarea).toHaveValue(second)

    await openKftl(page)

    await expect(page.locator(TAB)).toHaveCount(2)
    await expect(page.locator(TEXT_AREA)).toHaveValue(second)
  })
})
