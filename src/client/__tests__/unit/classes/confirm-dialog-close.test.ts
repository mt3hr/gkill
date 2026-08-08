/**
 * 確認ダイアログ(タグ/テキスト/通知の削除、リポスト作成)のテスト。
 *
 * どれも「サーバには届いているのに例外でクローズまで到達しない」形になっていた。
 * Kyou削除(use-confirm-delete-kyou-view.ts)と同じく、
 * 何があってもダイアログを閉じること・連打で多重リクエストにならないことを確認する。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
    get_gkill_api: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { useConfirmDeleteTagView } from '@/classes/use-confirm-delete-tag-view'
import { useConfirmDeleteTextView } from '@/classes/use-confirm-delete-text-view'
import { useConfirmDeleteNotificationView } from '@/classes/use-confirm-delete-notification-view'
import { useConfirmReKyouView } from '@/classes/use-confirm-re-kyou-view'

const application_config = { device: 'test-device', user_id: 'admin', for_share_kyou: false }

/** サーバ実物と同じく、成功時 errors/messages が null で返るレスポンス */
const ok_response = { messages: null, errors: null }

function make_meta_stub(id: string) {
  const stub = {
    id,
    target_id: 'kyou-1',
    is_deleted: false,
    update_time: new Date('2026-01-01T00:00:00Z'),
    generate_info_identifier: () => ({ id }),
    clone: () => ({ ...stub, clone: stub.clone }),
  }
  return stub
}

function make_kyou_stub() {
  return { id: 'kyou-1', data_type: 'kmemo', clone: () => make_kyou_stub() }
}

function emitted(emits: ReturnType<typeof vi.fn>, name: string) {
  return emits.mock.calls.filter(call => call[0] === name)
}

/**
 * 4種類を同じ形で回す。payload のキーと API 名だけが違う
 */
const cases = [
  {
    name: 'タグ削除',
    use: useConfirmDeleteTagView,
    api_name: 'update_tag',
    props_key: 'tag',
    method: 'delete_tag',
    response_extra: { updated_tag: make_meta_stub('tag-1') },
  },
  {
    name: 'テキスト削除',
    use: useConfirmDeleteTextView,
    api_name: 'update_text',
    props_key: 'text',
    method: 'delete_text',
    response_extra: { updated_text: make_meta_stub('text-1') },
  },
  {
    name: '通知削除',
    use: useConfirmDeleteNotificationView,
    api_name: 'update_notification',
    props_key: 'notification',
    method: 'delete_notification',
    response_extra: { updated_notification: make_meta_stub('notification-1') },
  },
  {
    name: 'リポスト作成',
    use: useConfirmReKyouView,
    api_name: 'add_rekyou',
    props_key: null,
    method: 'rekyou',
    response_extra: { added_kyou: make_kyou_stub() },
  },
] as const

type CaseSpec = typeof cases[number]

function build(spec: CaseSpec, api_impl: () => Promise<unknown>) {
  const emits = vi.fn()
  const api = {
    [spec.api_name]: vi.fn(api_impl),
    generate_uuid: vi.fn(() => 'mock-uuid'),
  }
  const props: Record<string, unknown> = {
    kyou: make_kyou_stub(),
    gkill_api: api,
    application_config,
  }
  if (spec.props_key) {
    props[spec.props_key] = make_meta_stub(`${spec.props_key}-1`)
  }
  const view = spec.use({ props: props as never, emits: emits as never }) as unknown as Record<string, unknown>
  return { view, emits, api }
}

describe('確認ダイアログは操作が終わったら必ず閉じる', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => { })
  })

  for (const spec of cases) {
    it(`${spec.name}: 成功したら閉じる`, async () => {
      const { view, emits, api } = build(spec, async () => ({ ...ok_response, ...spec.response_extra }))

      await (view[spec.method] as () => Promise<void>)()

      expect(api[spec.api_name]).toHaveBeenCalledTimes(1)
      expect(emitted(emits, 'received_errors')).toHaveLength(0)
      expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
    })

    // サーバ側の処理は既に走っているので、閉じずに残るとユーザからは操作が効かなく見える
    it(`${spec.name}: 例外が飛んでもエラーを出したうえで閉じる`, async () => {
      const { view, emits } = build(spec, async () => { throw new Error('boom') })

      await expect((view[spec.method] as () => Promise<void>)()).resolves.toBeUndefined()

      expect(emitted(emits, 'received_errors')).toHaveLength(1)
      expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
    })

    it(`${spec.name}: 連打してもリクエストは1回だけ`, async () => {
      let release: () => void = () => { }
      const blocker = new Promise<void>(resolve => { release = resolve })
      const { view, emits, api } = build(spec, async () => {
        await blocker
        return { ...ok_response, ...spec.response_extra }
      })

      const first = (view[spec.method] as () => Promise<void>)()
      const second = (view[spec.method] as () => Promise<void>)()
      release()
      await Promise.all([first, second])

      expect(api[spec.api_name]).toHaveBeenCalledTimes(1)
      expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
      expect((view.is_requested_submit as { value: boolean }).value).toBe(false)
    })
  }

  // 他のadd系と同じく、作ったリポストが一覧に出るようにする
  it('リポスト作成は registered_kyou と requested_reload_list を上げる', async () => {
    const spec = cases[3]
    const { view, emits } = build(spec, async () => ({ ...ok_response, ...spec.response_extra }))

    await (view.rekyou as () => Promise<void>)()

    expect(emitted(emits, 'registered_kyou')).toHaveLength(1)
    expect(emitted(emits, 'requested_reload_list')).toHaveLength(1)
  })
})
