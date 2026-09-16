/**
 * useSharedMiView のダイアログまわり。
 *
 * 共有ページも rykv / mi / dashboard と同じ RykvDialogHost を持つ。開いた直後の引き直しは
 * id キーの「引き直し中」表示を一覧の行にも点けるので、開くたびに親の一覧が読み込み中に
 * 見えていた（2026-09-14 修正）。4画面のうちここだけテストが無かったので固定する。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/kyou-reload', () => ({
  new_reload_batch: vi.fn(() => 0),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { useSharedMiView } from '@/classes/use-shared-mi-view'
import type { Kyou } from '@/classes/datas/kyou'

async function flush(): Promise<void> {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
  }
}

function build() {
  const api = {
    generate_uuid: vi.fn(() => 'dialog-1'),
    delete_updated_gkill_caches: vi.fn().mockResolvedValue(undefined),
    get_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: [], errors: [] }),
  }
  const emits = vi.fn()
  const view = useSharedMiView({
    props: { gkill_api: api, share_title: '共有', app_content_height: 600 } as never,
    emits: emits as never,
  })
  return { api, emits, view }
}

describe('useSharedMiView ダイアログ', () => {
  test('開いた直後には引き直さず、開いた時点の Kyou をそのまま出す', async () => {
    const { view } = build()
    await flush()
    vi.mocked(refresh_kyou).mockClear()
    vi.mocked(refresh_kyou_in_list).mockClear()
    const kyou = { id: 'kyou-1', clone: () => ({ id: 'kyou-1' }) } as unknown as Kyou

    view.open_rykv_dialog('kyou', kyou)
    await flush()

    expect(vi.mocked(refresh_kyou)).not.toHaveBeenCalled()
    expect(vi.mocked(refresh_kyou_in_list)).not.toHaveBeenCalled()
    expect(view.opened_dialogs.value).toHaveLength(1)
    expect(view.opened_dialogs.value[0].kyou.id).toBe('kyou-1')
    expect(view.opened_dialogs.value[0].kind).toBe('kyou')
  })

  test('closed で該当ダイアログだけ閉じる', async () => {
    const { api, view } = build()
    await flush()
    api.generate_uuid.mockReturnValueOnce('dialog-1').mockReturnValueOnce('dialog-2')
    view.open_rykv_dialog('kyou', { id: 'kyou-1', clone: () => ({ id: 'kyou-1' }) } as unknown as Kyou)
    view.open_rykv_dialog('add_tag', { id: 'kyou-2', clone: () => ({ id: 'kyou-2' }) } as unknown as Kyou)
    expect(view.opened_dialogs.value).toHaveLength(2)

    view.close_rykv_dialog('dialog-1')

    expect(view.opened_dialogs.value.map(dialog => dialog.id)).toEqual(['dialog-2'])
  })
})
