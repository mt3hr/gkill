import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import {
  submitKftlText, navigateToRykv, makeUniqueLabel,
  clickContextMenuItem, clickDialogButton, confirmDelete,
  expectPageToContainText, waitForKyouByText, waitForAttachedNotification, pickNotificationTime,
  openKyouDetailPane,
  MENU, SAVE_BUTTON,
} from './crud-helpers'

/**
 * 記録に付ける通知の追加・編集・削除。
 *
 * 以前はこのファイルの5本中3本が
 *   - 通知を追加も編集も削除もせず
 *   - `expect(app).toBeVisible()` / `innerHTML.length > 100` を確認するだけ
 * という状態で、テスト名と実際に見ているものが一致していなかった。
 * ここでは追加 → 編集 → 削除を通しで確認する。
 */

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

const DIALOG = '.gkill-floating-dialog, .v-dialog'

/** 表示中のダイアログの最初のテキスト欄を value で埋める。 */
async function fillDialogText(page: import('@playwright/test').Page, value: string): Promise<void> {
  // ダイアログは重なって開くことがあるので、表示中の最前面を対象にする
  const dialog = page.locator(DIALOG).filter({ visible: true }).last()
  await expect(dialog, 'ダイアログが開かない').toBeVisible({ timeout: 15000 })

  const field = dialog.locator('textarea, input[type="text"]').first()
  await expect(field, 'ダイアログに入力欄が無い').toBeVisible({ timeout: 15000 })
  await field.fill(value)
  await expect(field).toHaveValue(value)
}

test.describe('Notification CRUD Flows', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(120000)
    await loginAsAdmin(page)
  })

  /** 記録を作って通知を1件付ける。作った記録のラベルを返す。 */
  async function createRecordWithNotification(page: Page, prefix: string, content: string): Promise<string> {
    const label = makeUniqueLabel(prefix)

    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.addNotification)
    await fillDialogText(page, content)
    await pickNotificationTime(page)
    await clickDialogButton(page, SAVE_BUTTON)

    return label
  }

  test('記録に通知を追加すると詳細ペインに表示される', async ({ page }) => {
    const notification = makeUniqueLabel('notifContent')
    const label = await createRecordWithNotification(page, 'notif_add_target', notification)

    await navigateToRykv(page)
    const pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))
    await waitForAttachedNotification(pane, notification)
  })

  test('通知を編集すると内容が置き換わる', async ({ page }) => {
    const original = makeUniqueLabel('notifOrig')
    const edited = makeUniqueLabel('notifEdited')
    const label = await createRecordWithNotification(page, 'notif_edit_target', original)

    await navigateToRykv(page)
    let pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))

    const notificationElement = await waitForAttachedNotification(pane, original)
    await notificationElement.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.editNotification)
    await fillDialogText(page, edited)
    await clickDialogButton(page, SAVE_BUTTON)

    await navigateToRykv(page)
    pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))
    await waitForAttachedNotification(pane, edited)
    await expect(pane.locator('.notification_content').filter({ hasText: original }), '編集前の通知が残っている')
      .toHaveCount(0, { timeout: 30000 })
  })

  test('通知を削除すると詳細ペインから消える', async ({ page }) => {
    const notification = makeUniqueLabel('notifToDelete')
    const label = await createRecordWithNotification(page, 'notif_delete_target', notification)

    await navigateToRykv(page)
    let pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))

    const notificationElement = await waitForAttachedNotification(pane, notification)
    await notificationElement.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.deleteNotification)
    await confirmDelete(page)

    await navigateToRykv(page)
    pane = await openKyouDetailPane(page, await waitForKyouByText(page, label))
    await expect(pane.locator('.notification_content').filter({ hasText: notification }), '通知が消えていない')
      .toHaveCount(0, { timeout: 30000 })
    // 通知を消しても記録自体は残る
    await expectPageToContainText(page, label)
  })

  test('記録の履歴ダイアログをコンテキストメニューから開ける', async ({ page }) => {
    const label = makeUniqueLabel('notif_hist_target')
    await submitKftlText(page, label)
    await navigateToRykv(page)

    const record = await waitForKyouByText(page, label)
    await record.click({ button: 'right', force: true })
    await clickContextMenuItem(page, MENU.histories)

    const dialog = page.locator(DIALOG).first()
    await expect(dialog, '履歴ダイアログが開かない').toBeVisible({ timeout: 15000 })
    // 履歴には対象の記録が並ぶ
    await expect(dialog).toContainText(label, { timeout: 15000 })
  })
})
