/**
 * Cascade delete tests.
 *
 * Kyou削除時に、付随するTag/Text/Notificationと、それを参照しているReKyou/MiReKyouも
 * 連鎖して論理削除する。探索と1つのトランザクションでの確定はサーバ（/api/delete_kyou_cascade →
 * usecase.DeleteKyouCascade）の1実装が持ち、その挙動は Go 側の handle_delete_kyou_cascade_test.go が固定する（ADR-0636）。
 * ここで守るのは画面側の約束: API を1回だけ呼ぶ・画面から取り除く行の id を返す・消した全 id の
 * Service Worker キャッシュを落とす・失敗したら何も取り除かない・共有画面では何もしない・errors: null で落ちない。
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

import { build_cascade_delete_failed_error, build_deleted_kyou_stub, cascade_delete_kyou } from '@/classes/cascade-delete-kyou'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

interface DeletedRecordStub {
  id: string
  data_type: string
}

interface CascadeRequestStub {
  target_id: string
  data_type: string
  update_app: string
}

function create_api(response: { deleted?: Array<DeletedRecordStub> | null, errors?: Array<{ error_code: string }> | null, messages?: Array<never> | null }) {
  return {
    delete_kyou_cascade: vi.fn(async (_req: CascadeRequestStub) => ({
      messages: response.messages === undefined ? [] : response.messages,
      errors: response.errors === undefined ? [] : response.errors,
      deleted: response.deleted === undefined ? [] : response.deleted,
    })),
  }
}

const application_config = { device: 'test-device', user_id: 'admin', for_share_kyou: false }

function run(kyou: { id: string, data_type: string }, api: ReturnType<typeof create_api>, config: object = application_config) {
  return cascade_delete_kyou({
    kyou: kyou as never,
    gkill_api: api as never,
    application_config: config as never,
  })
}

describe('cascade_delete_kyou', () => {
  beforeEach(() => {
    vi.mocked(delete_gkill_kyou_cache).mockClear()
  })

  it('サーバの連鎖削除を1回だけ呼び、起点の id と data_type（射影名のまま）を渡す', async () => {
    const api = create_api({ deleted: [{ id: 'mi-1', data_type: 'mi' }] })

    await run({ id: 'mi-1', data_type: 'mi_create' }, api)

    expect(api.delete_kyou_cascade).toHaveBeenCalledTimes(1)
    const req = api.delete_kyou_cascade.mock.calls[0][0]
    expect(req.target_id).toBe('mi-1')
    expect(req.data_type).toBe('mi_create')
    expect(req.update_app).toBe('gkill')
  })

  it('画面から取り除く行は起点と参照元（ReKyou / MiReKyou）だけで、付随データは含めない', async () => {
    const api = create_api({
      deleted: [
        { id: 'kmemo-1', data_type: 'kmemo' },
        { id: 'tag-1', data_type: 'tag' },
        { id: 'text-1', data_type: 'text' },
        { id: 'notification-1', data_type: 'notification' },
        { id: 'rekyou-1', data_type: 'rekyou' },
        { id: 'mirekyou-1', data_type: 'mirekyou' },
      ],
    })

    const result = await run({ id: 'kmemo-1', data_type: 'kmemo' }, api)

    expect(result.errors).toEqual([])
    expect(result.deleted_ids.sort()).toEqual(['kmemo-1', 'mirekyou-1', 'rekyou-1'])
  })

  it('消した全idのキャッシュを落とす（付随データも含む）', async () => {
    const api = create_api({
      deleted: [
        { id: 'kmemo-1', data_type: 'kmemo' },
        { id: 'tag-1', data_type: 'tag' },
        { id: 'rekyou-B', data_type: 'rekyou' },
      ],
    })

    await run({ id: 'kmemo-1', data_type: 'kmemo' }, api)

    const cleared = vi.mocked(delete_gkill_kyou_cache).mock.calls.map(call => call[0]).sort()
    expect(cleared).toEqual(['kmemo-1', 'rekyou-B', 'tag-1'])
  })

  it('サーバがエラーを返したら何も取り除かず、キャッシュも落とさない（1つのトランザクションなので何も消えていない）', async () => {
    const api = create_api({ deleted: [], errors: [{ error_code: 'ERR000451' }] })

    const result = await run({ id: 'kmemo-1', data_type: 'kmemo' }, api)

    expect(result.deleted_ids).toEqual([])
    expect(result.errors).toHaveLength(1)
    expect(result.errors[0].error_code).toBe('ERR000451')
    expect(delete_gkill_kyou_cache).not.toHaveBeenCalled()
  })

  // SW キャッシュに残った古い応答や古いサーバとの組み合わせでは errors / deleted が null で来うる。
  // spread や map で展開すると TypeError になり、呼び出し元のダイアログのクローズまで巻き込んで落ちる
  it('errors / deleted が null でも throw しない', async () => {
    const api = create_api({ deleted: null, errors: null, messages: null })

    const result = await run({ id: 'kmemo-1', data_type: 'kmemo' }, api)

    expect(result.errors).toEqual([])
    expect(result.deleted_ids).toEqual([])
  })

  it('共有画面では何もしない', async () => {
    const api = create_api({ deleted: [{ id: 'kmemo-1', data_type: 'kmemo' }] })

    const result = await run({ id: 'kmemo-1', data_type: 'kmemo' }, api, { device: '', user_id: '', for_share_kyou: true })

    expect(result.deleted_ids).toEqual([])
    expect(api.delete_kyou_cascade).not.toHaveBeenCalled()
  })
})

describe('build_deleted_kyou_stub', () => {
  it('idだけを持つKyouを作る', () => {
    const stub = build_deleted_kyou_stub('kmemo-1')
    expect(stub.id).toBe('kmemo-1')
    expect(stub.is_deleted).toBe(true)
  })
})

describe('build_cascade_delete_failed_error', () => {
  it('「何も削除されていない」の文言を持つ', () => {
    const error = build_cascade_delete_failed_error()
    expect(error.error_code).toBe('ERR900094')
    expect(error.error_message).toBe('FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE')
  })
})
