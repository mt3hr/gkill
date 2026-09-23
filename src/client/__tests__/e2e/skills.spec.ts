import { test, expect, type Page } from '@playwright/test'
import { checkGkillServer, checkGkillApiViaVite } from './check-server'
import { loginAsAdmin } from './helpers'
import { openApplicationConfigDialog } from './crud-helpers'

// スキル（ADR-0634）の画面の一巡: 設定 → スキル → zip をアップロード（確認 → 適用）→ 一覧 → 表示 →
// ダウンロード → 同じ名前で上げ直すと「変わる・消える」が確認に出る → 削除。
// zip はテストの中で組み立てる（バイナリのフィクスチャをコミットしない）。

let apiReachable = false
test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
  apiReachable = await checkGkillApiViaVite()
})

// CRC-32（zip の各項目に必要）。Node の zlib.crc32 は版によって無いので自前で計算する
const CRC_TABLE = Array.from({ length: 256 }, (_, n) => {
  let c = n
  for (let k = 0; k < 8; k++) {
    c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
  }
  return c >>> 0
})
function crc32(data: Buffer): number {
  let crc = 0xffffffff
  for (const byte of data) {
    crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8)
  }
  return (crc ^ 0xffffffff) >>> 0
}

// makeStoredZip は無圧縮（stored）の zip を作る。files は「zip 内のパス → 中身」
function makeStoredZip(files: Record<string, string>): Buffer {
  const locals: Array<Buffer> = []
  const centrals: Array<Buffer> = []
  let offset = 0
  for (const [name, text] of Object.entries(files)) {
    const nameBytes = Buffer.from(name, 'utf8')
    const data = Buffer.from(text, 'utf8')
    const crc = crc32(data)
    const local = Buffer.alloc(30)
    local.writeUInt32LE(0x04034b50, 0)
    local.writeUInt16LE(20, 4)
    local.writeUInt16LE(0x0800, 6) // UTF-8 のファイル名
    local.writeUInt16LE(0, 8) // stored
    local.writeUInt32LE(crc, 14)
    local.writeUInt32LE(data.length, 18)
    local.writeUInt32LE(data.length, 22)
    local.writeUInt16LE(nameBytes.length, 26)
    locals.push(local, nameBytes, data)

    const central = Buffer.alloc(46)
    central.writeUInt32LE(0x02014b50, 0)
    central.writeUInt16LE(20, 4)
    central.writeUInt16LE(20, 6)
    central.writeUInt16LE(0x0800, 8)
    central.writeUInt16LE(0, 10)
    central.writeUInt32LE(crc, 16)
    central.writeUInt32LE(data.length, 20)
    central.writeUInt32LE(data.length, 24)
    central.writeUInt16LE(nameBytes.length, 28)
    central.writeUInt32LE(offset, 42)
    centrals.push(central, nameBytes)
    offset += 30 + nameBytes.length + data.length
  }
  const centralDirectory = Buffer.concat(centrals)
  const end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50, 0)
  end.writeUInt16LE(Object.keys(files).length, 8)
  end.writeUInt16LE(Object.keys(files).length, 10)
  end.writeUInt32LE(centralDirectory.length, 12)
  end.writeUInt32LE(offset, 16)
  return Buffer.concat([...locals, centralDirectory, end])
}

function manifest(name: string, description: string, body: string): string {
  return `---\nname: ${name}\ndescription: ${description}\n---\n${body}\n`
}

function topDialog(page: Page) {
  return page.locator('.gkill-floating-dialog').last()
}

async function uploadZip(page: Page, skillList: ReturnType<typeof topDialog>, zip: Buffer) {
  await skillList.locator('input[type="file"]').setInputFiles({ name: 'skill.zip', mimeType: 'application/zip', buffer: zip })
}

test.describe('Skills (settings)', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!apiReachable, 'gkill API not reachable via Vite dev server')
    test.setTimeout(180000)
    await loginAsAdmin(page)
  })

  test('upload with confirmation, view, download, re-upload and delete a skill', async ({ page }) => {
    const name = `e2e-skill-${Date.now()}`
    const settings = await openApplicationConfigDialog(page)
    await settings.getByRole('button', { name: 'スキル', exact: true }).click()
    const skillList = topDialog(page)
    await expect(skillList.getByText('zipをアップロード'), 'スキル管理ダイアログが開かない').toBeVisible({ timeout: 30000 })

    // 1段目: dry_run の結果が確認に出る（まだ一覧には無い）
    await uploadZip(page, skillList, makeStoredZip({
      [`${name}/SKILL.md`]: manifest(name, 'E2E で作るスキル', '# 手順\nreferences/tags.md を読む'),
      [`${name}/references/tags.md`]: '# タグの意味\n',
    }))
    const confirmNew = topDialog(page)
    await expect(confirmNew.getByText('新しいスキルを作ります。'), '新規作成の確認が出ない').toBeVisible({ timeout: 30000 })
    await expect(confirmNew.getByText('references/tags.md')).toBeVisible()
    await expect(skillList.locator('tr', { hasText: name }), '確認の前に書き込まれている').toHaveCount(0)

    // 2段目: 適用で一覧に出る
    await confirmNew.getByRole('button', { name: '適用', exact: true }).click()
    const row = skillList.locator('tr', { hasText: name })
    await expect(row, 'アップロードしたスキルが一覧に出ない').toBeVisible({ timeout: 30000 })
    await expect(row).toContainText('E2E で作るスキル')

    // 表示: SKILL.md の本文が素の文字で出て、付属ファイルも読める
    await row.getByRole('button', { name: '表示', exact: true }).click()
    const browse = topDialog(page)
    await expect(browse.locator('pre'), 'SKILL.md の本文が出ない').toContainText('references/tags.md を読む', { timeout: 30000 })
    await browse.getByText('references/tags.md', { exact: true }).click()
    await expect(browse.locator('pre')).toContainText('# タグの意味', { timeout: 30000 })
    await browse.locator('.gkill-floating-dialog__header button:has(.mdi-close)').click()
    await expect(browse.locator('pre')).toHaveCount(0)

    // ダウンロード: スキル名の zip が落ちてくる
    const downloadPromise = page.waitForEvent('download')
    await row.getByRole('button', { name: 'ダウンロード', exact: true }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toBe(`${name}.zip`)

    // 上げ直し: 変わる・消える・増えるが確認に出る
    await uploadZip(page, skillList, makeStoredZip({
      'SKILL.md': manifest(name, 'E2E で直したスキル', '# 手順（改）'),
      'scripts/run.py': "print('ok')\n",
    }))
    const confirmReplace = topDialog(page)
    await expect(confirmReplace.getByText('既存のスキルを丸ごと置き換えます。', { exact: false }), '置き換えの確認が出ない')
      .toBeVisible({ timeout: 30000 })
    await expect(confirmReplace.locator('.confirm-upload-skill-changed')).toContainText('SKILL.md')
    await expect(confirmReplace.locator('.confirm-upload-skill-removed')).toContainText('references/tags.md')
    await expect(confirmReplace.locator('.confirm-upload-skill-added')).toContainText('scripts/run.py')
    await confirmReplace.getByRole('button', { name: '適用', exact: true }).click()
    await expect(row, '置き換え後の説明が一覧に出ない').toContainText('E2E で直したスキル', { timeout: 30000 })

    // 削除: 確認してから丸ごと消える
    await row.getByRole('button', { name: '削除', exact: true }).click()
    const confirmDelete = topDialog(page)
    await expect(confirmDelete.getByText('スキルのフォルダを丸ごと消します。元には戻せません。')).toBeVisible({ timeout: 30000 })
    await confirmDelete.getByRole('button', { name: '削除', exact: true }).click()
    await expect(skillList.locator('tr', { hasText: name }), '削除したスキルが一覧に残っている').toHaveCount(0, { timeout: 30000 })
  })
})
