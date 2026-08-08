/**
 * 削除確認ビューのテスト。
 *
 * 削除リクエストはサーバに届いているのに、途中で例外が飛んでダイアログのクローズまで
 * 到達しない（＝「消えているのに閉じない」）状態にならないことを確認する。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      generate_uuid: vi.fn(() => 'mock-uuid'),
    })),
    get_gkill_api: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
    })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { useConfirmDeleteKyouView } from '@/classes/use-confirm-delete-kyou-view'

const base_time = new Date('2026-01-01T00:00:00Z')

const application_config = { device: 'test-device', user_id: 'admin', for_share_kyou: false }

function make_mirekyou_kyou(id: string) {
  const typed_mirekyou = {
    id,
    target_id: 'kmemo-1',
    is_deleted: false,
    update_time: base_time,
    clone: () => ({ ...typed_mirekyou, clone: typed_mirekyou.clone }),
  }
  const kyou = {
    id,
    data_type: 'mirekyou_create',
    typed_mirekyou,
    load_typed_datas: vi.fn().mockResolvedValue([]),
    clone: () => kyou,
  }
  return kyou
}

/** 逆引きも更新も成功し、errorsはサーバ実物と同じく null で返るAPI */
function create_api() {
  const empty = { messages: null, errors: null }
  return {
    get_tags_by_target_id: vi.fn(async () => ({ tags: [], ...empty })),
    get_texts_by_target_id: vi.fn(async () => ({ texts: [], ...empty })),
    get_notifications_by_target_id: vi.fn(async () => ({ notifications: [], ...empty })),
    get_rekyous_by_target_id: vi.fn(async () => ({ rekyous: [], ...empty })),
    get_mirekyous_by_target_id: vi.fn(async () => ({ mirekyous: [], ...empty })),
    update_mirekyou: vi.fn(async () => empty),
  }
}

type ApiStub = ReturnType<typeof create_api>

function build_view(api: ApiStub, kyou: ReturnType<typeof make_mirekyou_kyou>) {
  const emits = vi.fn()
  const view = useConfirmDeleteKyouView({
    props: {
      kyou: kyou as never,
      gkill_api: api as never,
      application_config: application_config as never,
    } as never,
    emits: emits as never,
  })
  return { view, emits }
}

/** emitsのモックから、指定イベント名の呼び出しだけ取り出す */
function emitted(emits: ReturnType<typeof vi.fn>, name: string) {
  return emits.mock.calls.filter(call => call[0] === name)
}

describe('useConfirmDeleteKyouView', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => { })
  })

  it('削除に成功したらdeleted_kyouとrequested_close_dialogをemitする', async () => {
    const api = create_api()
    const { view, emits } = build_view(api, make_mirekyou_kyou('mirekyou-1'))

    await view.delete_kyou()

    expect(api.update_mirekyou).toHaveBeenCalledTimes(1)
    expect(emitted(emits, 'received_errors')).toHaveLength(0)
    expect(emitted(emits, 'deleted_kyou')).toHaveLength(1)
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
  })

  // サーバ側の削除は既に飛んでいるので、閉じずに残るとユーザからは
  // 「消えているのにダイアログが閉じない」に見える
  it('途中で例外が飛んでもエラーを出したうえでrequested_close_dialogをemitする', async () => {
    const api = create_api()
    api.update_mirekyou.mockRejectedValueOnce(new Error('boom'))
    const { view, emits } = build_view(api, make_mirekyou_kyou('mirekyou-1'))

    await expect(view.delete_kyou()).resolves.toBeUndefined()

    expect(emitted(emits, 'received_errors')).toHaveLength(1)
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
  })

  it('削除中の二重押しではリクエストを重ねて投げない', async () => {
    const api = create_api()
    let release: () => void = () => { }
    const blocker = new Promise<void>(resolve => { release = resolve })
    api.update_mirekyou.mockImplementationOnce(async () => {
      await blocker
      return { messages: null, errors: null }
    })
    const { view, emits } = build_view(api, make_mirekyou_kyou('mirekyou-1'))

    const first = view.delete_kyou()
    const second = view.delete_kyou()
    release()
    await Promise.all([first, second])

    expect(api.update_mirekyou).toHaveBeenCalledTimes(1)
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
    expect(view.is_requested_submit.value).toBe(false)
  })
})
