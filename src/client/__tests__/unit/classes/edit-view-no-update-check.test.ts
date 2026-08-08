/**
 * 編集ビューの「更新がありません」判定のテスト。
 *
 * 判定に関連日時が入っていないと、日時だけ変えたときにエラーになって
 * 保存もクローズもされない（＝ユーザからは操作が効かないように見える）。
 * kmemo / kc / lantana がこの形だった。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import moment from 'moment'

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

import { useEditKmemoView } from '@/classes/use-edit-kmemo-view'
import { useEditKCView } from '@/classes/use-edit-kc-view'

const base_time = new Date('2026-01-01T09:00:00')
const application_config = { device: 'test-device', user_id: 'admin', for_share_kyou: false }
const ok_response = { messages: null, errors: null }

function make_typed_stub(extra: Record<string, unknown>) {
  const stub = {
    id: 'kyou-1',
    related_time: base_time,
    is_deleted: false,
    update_time: base_time,
    ...extra,
    clone: () => ({ ...stub, clone: stub.clone }),
  }
  return stub
}

function make_kyou(typed_key: string, typed: Record<string, unknown>) {
  const kyou: Record<string, unknown> = {
    id: 'kyou-1',
    related_time: base_time,
    abort_controller: new AbortController(),
    load_typed_datas: vi.fn().mockResolvedValue([]),
  }
  kyou[typed_key] = typed
  kyou.clone = () => {
    const cloned = { ...kyou }
    cloned.abort_controller = new AbortController()
    cloned.clone = kyou.clone
    return cloned
  }
  return kyou
}

function emitted(emits: ReturnType<typeof vi.fn>, name: string) {
  return emits.mock.calls.filter(call => call[0] === name)
}

/** 「更新がありません」エラーが出たか */
function has_no_update_error(emits: ReturnType<typeof vi.fn>) {
  return emitted(emits, 'received_errors').some(call => {
    const errors = call[1] as Array<{ error_message: string }>
    return errors.some(e => e.error_message.includes('NO_UPDATE'))
  })
}

describe('編集ビューは関連日時だけ変えても保存できる', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => { })
  })

  it('kmemo: 本文はそのままで日時だけ変えたら更新リクエストが飛ぶ', async () => {
    const emits = vi.fn()
    const api = { update_kmemo: vi.fn(async () => ({ ...ok_response, updated_kyou: { id: 'kyou-1' } })) }
    const kyou = make_kyou('typed_kmemo', make_typed_stub({ content: '本文' }))
    const view = useEditKmemoView({
      props: { kyou: kyou as never, gkill_api: api as never, application_config: application_config as never } as never,
      emits: emits as never,
    })

    // 日時だけ1時間ずらす
    view.related_time_string.value = moment(base_time).add(1, 'hour').format('HH:mm:ss')
    await view.save()

    expect(has_no_update_error(emits), '日時だけ変えたのに「更新なし」になっている').toBe(false)
    expect(api.update_kmemo).toHaveBeenCalledTimes(1)
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
  })

  it('kmemo: 本文も日時も変えなければ「更新なし」で閉じない', async () => {
    const emits = vi.fn()
    const api = { update_kmemo: vi.fn(async () => ({ ...ok_response })) }
    const kyou = make_kyou('typed_kmemo', make_typed_stub({ content: '本文' }))
    const view = useEditKmemoView({
      props: { kyou: kyou as never, gkill_api: api as never, application_config: application_config as never } as never,
      emits: emits as never,
    })

    await view.save()

    expect(has_no_update_error(emits)).toBe(true)
    expect(api.update_kmemo).not.toHaveBeenCalled()
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(0)
  })

  it('kc: タイトルと数値はそのままで日時だけ変えたら更新リクエストが飛ぶ', async () => {
    const emits = vi.fn()
    const api = { update_kc: vi.fn(async () => ({ ...ok_response, updated_kyou: { id: 'kyou-1' } })) }
    const kyou = make_kyou('typed_kc', make_typed_stub({ title: 'タイトル', num_value: 1 }))
    const view = useEditKCView({
      props: { kyou: kyou as never, gkill_api: api as never, application_config: application_config as never } as never,
      emits: emits as never,
    })

    view.related_time_string.value = moment(base_time).add(1, 'hour').format('HH:mm:ss')
    await view.save()

    expect(has_no_update_error(emits), '日時だけ変えたのに「更新なし」になっている').toBe(false)
    expect(api.update_kc).toHaveBeenCalledTimes(1)
    expect(emitted(emits, 'requested_close_dialog')).toHaveLength(1)
  })
})
