/**
 * delete_updated_gkill_caches のテスト。
 *
 * 別のブラウザが付けたタグを反映する唯一の経路。検索直前にサーバへ
 * 「前回以降に更新されたID」を問い合わせ、そのIDのSWキャッシュを捨てる。
 * ウォーターマーク(last_cache_update_time)を空振り時にも進めてしまうと、
 * その時間帯の更新は二度と通知されず古い応答が恒久的に焼き付く。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n }))

const mocks = vi.hoisted(() => ({
  delete_gkill_kyou_cache: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: mocks.delete_gkill_kyou_cache,
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

import { GkillAPI } from '@/classes/api/gkill-api'
import type { GetApplicationConfigResponse } from '@/classes/api/req_res/get-application-config-response'
import type { GetUpdatedDatasByTimeResponse } from '@/classes/api/req_res/get-updated-datas-by-time-response'
import { GkillError } from '@/classes/api/gkill-error'

const cache_clear_count_limit = 3

function stub_caches(entry_count: number): void {
  Object.defineProperty(globalThis, 'caches', {
    value: {
      open: vi.fn().mockResolvedValue({
        keys: vi.fn().mockResolvedValue(new Array(entry_count).fill('/cache/api/kyou/x')),
      }),
      delete: vi.fn().mockResolvedValue(true),
    },
    writable: true,
    configurable: true,
  })
}

function stub_api(updated_datas_response: Partial<GetUpdatedDatasByTimeResponse>) {
  const api = GkillAPI.get_instance()
  vi.spyOn(api, 'get_application_config').mockResolvedValue({
    application_config: { cache_clear_count_limit },
  } as unknown as GetApplicationConfigResponse)
  vi.spyOn(api, 'get_updated_datas_by_time').mockResolvedValue(
    updated_datas_response as GetUpdatedDatasByTimeResponse,
  )
  vi.spyOn(api, 'get_last_cache_update_time').mockReturnValue(new Date(1_000))
  const set_last_cache_update_time = vi
    .spyOn(api, 'set_last_cache_update_time')
    .mockImplementation(() => { })
  return { api, set_last_cache_update_time }
}

describe('delete_updated_gkill_caches', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    mocks.delete_gkill_kyou_cache.mockClear()
    mocks.delete_gkill_kyou_cache.mockResolvedValue(undefined)
    stub_caches(1)
  })

  test('更新されたIDのキャッシュを1件ずつ捨て、ウォーターマークを進める', async () => {
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: ['id-a', 'id-b'], errors: null as unknown as Array<GkillError> })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledTimes(2)
    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledWith('id-a')
    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledWith('id-b')
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })

  test('上限を超えたら個別削除ではなく全消しに切り替える', async () => {
    const updated_ids = ['a', 'b', 'c', 'd']
    expect(updated_ids.length).toBeGreaterThan(cache_clear_count_limit)
    const { api, set_last_cache_update_time } = stub_api({ updated_ids, errors: null as unknown as Array<GkillError> })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledTimes(1)
    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledWith(null)
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })

  // これが退行すると「一度でもサーバがエラーを返したら、その時間帯の更新は
  // 二度とキャッシュ削除されない」状態になる
  test('エラー応答ではウォーターマークを進めない', async () => {
    const errors = [new GkillError()]
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: null as unknown as Array<string>, errors })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).not.toHaveBeenCalled()
  })

  test('updated_idsが返ってこなければウォーターマークを進めない', async () => {
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: undefined as unknown as Array<string>, errors: null as unknown as Array<GkillError> })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).not.toHaveBeenCalled()
  })

  test('キャッシュが空なら問い合わせずに終わる', async () => {
    stub_caches(0)
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: ['id-a'], errors: null as unknown as Array<GkillError> })

    await api.delete_updated_gkill_caches()

    expect(api.get_updated_datas_by_time).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).not.toHaveBeenCalled()
  })
})
