/**
 * delete_updated_gkill_caches のテスト。
 *
 * 別のブラウザが付けたタグを反映する唯一の経路。検索直前にサーバへ
 * 「前回以降に更新されたID」を問い合わせ、そのIDのSWキャッシュを捨てる。
 * ウォーターマーク(last_cache_update_time)を空振り時にも進めてしまうと、
 * その時間帯の更新は二度と通知されず古い応答が恒久的に焼き付く。
 *
 * これは**検索のたび**に走るので、往復回数と削除の待ち方もここで固定する。
 * 以前は (1) 空判定のためにKyouキャッシュの全キーを実体化し、
 * (2) cache_clear_count_limit を読むためだけに ApplicationConfig を
 *     not_load_all 無しで取って合計4往復し、
 * (3) 更新IDごとに逐次 await で削除していた。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n }))

const mocks = vi.hoisted(() => ({
  delete_gkill_kyou_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_kyou_caches: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: mocks.delete_gkill_kyou_cache,
  delete_gkill_kyou_caches: mocks.delete_gkill_kyou_caches,
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

import { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GetApplicationConfigResponse } from '@/classes/api/req_res/get-application-config-response'
import type { GetUpdatedDatasByTimeResponse } from '@/classes/api/req_res/get-updated-datas-by-time-response'
import { GkillError } from '@/classes/api/gkill-error'

const cache_clear_count_limit = 3

function stub_api(
  updated_datas_response: Partial<GetUpdatedDatasByTimeResponse>,
  options?: { saved_application_config?: boolean, last_cache_update_time?: Date | null },
) {
  const api = GkillAPI.get_instance()
  const get_application_config = vi.spyOn(api, 'get_application_config').mockResolvedValue({
    application_config: { cache_clear_count_limit },
  } as unknown as GetApplicationConfigResponse)
  vi.spyOn(api, 'get_saved_application_config').mockReturnValue(
    options?.saved_application_config
      ? ({ cache_clear_count_limit } as unknown as ApplicationConfig)
      : null,
  )
  const get_updated_datas_by_time = vi.spyOn(api, 'get_updated_datas_by_time').mockResolvedValue(
    updated_datas_response as GetUpdatedDatasByTimeResponse,
  )
  vi.spyOn(api, 'get_last_cache_update_time').mockReturnValue(
    options?.last_cache_update_time === undefined ? new Date(1_000) : options.last_cache_update_time,
  )
  const set_last_cache_update_time = vi
    .spyOn(api, 'set_last_cache_update_time')
    .mockImplementation(() => { })
  return { api, set_last_cache_update_time, get_application_config, get_updated_datas_by_time }
}

const no_errors = null as unknown as Array<GkillError>

describe('delete_updated_gkill_caches', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    mocks.delete_gkill_kyou_cache.mockClear()
    mocks.delete_gkill_kyou_cache.mockResolvedValue(undefined)
    mocks.delete_gkill_kyou_caches.mockClear()
    mocks.delete_gkill_kyou_caches.mockResolvedValue(undefined)
  })

  // 1件ずつ await していたのを1回にまとめた。
  // 既定の上限は3000件で、1IDにつき16回のcache.deleteが走るため、
  // 逐次だと検索の前に最大48,000回の待ちが並ぶ
  test('更新されたIDのキャッシュを1回でまとめて捨て、ウォーターマークを進める', async () => {
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: ['id-a', 'id-b'], errors: no_errors })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_caches).toHaveBeenCalledTimes(1)
    expect(mocks.delete_gkill_kyou_caches).toHaveBeenCalledWith(['id-a', 'id-b'])
    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })

  test('上限を超えたら個別削除ではなく全消しに切り替える', async () => {
    const updated_ids = ['a', 'b', 'c', 'd']
    expect(updated_ids.length).toBeGreaterThan(cache_clear_count_limit)
    const { api, set_last_cache_update_time } = stub_api({ updated_ids, errors: no_errors })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledTimes(1)
    expect(mocks.delete_gkill_kyou_cache).toHaveBeenCalledWith(null)
    expect(mocks.delete_gkill_kyou_caches).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })

  // これが退行すると「一度でもサーバがエラーを返したら、その時間帯の更新は
  // 二度とキャッシュ削除されない」状態になる
  test('エラー応答ではウォーターマークを進めない', async () => {
    const errors = [new GkillError()]
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: null as unknown as Array<string>, errors })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_caches).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).not.toHaveBeenCalled()
  })

  test('updated_idsが返ってこなければウォーターマークを進めない', async () => {
    const { api, set_last_cache_update_time } = stub_api({ updated_ids: undefined as unknown as Array<string>, errors: no_errors })

    await api.delete_updated_gkill_caches()

    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_caches).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).not.toHaveBeenCalled()
  })

  // 初回はまだ「前回以降の更新」が定義できない。往復しても消すものが無い
  test('ウォーターマークが無ければ問い合わせずに基準時刻だけ置く', async () => {
    const { api, set_last_cache_update_time, get_updated_datas_by_time } =
      stub_api({ updated_ids: ['id-a'], errors: no_errors }, { last_cache_update_time: null })

    await api.delete_updated_gkill_caches()

    expect(get_updated_datas_by_time).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_caches).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })

  // ApplicationConfigは cache_clear_count_limit を読むためだけに要る。
  // 保存済みがあるのに取りに行くと、内部の load_all() が
  // get_all_rep_names / get_all_tag_names / get_mi_board_list まで引いて
  // 検索の前に毎回4往復する
  test('保存済みApplicationConfigがあればサーバへ取りに行かない', async () => {
    const { api, get_application_config } =
      stub_api({ updated_ids: ['id-a'], errors: no_errors }, { saved_application_config: true })

    await api.delete_updated_gkill_caches()

    expect(get_application_config).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_caches).toHaveBeenCalledWith(['id-a'])
  })

  // 更新が無いときは削除もApplicationConfigの取得も要らない
  test('更新IDが0件なら削除もApplicationConfig取得もせずウォーターマークだけ進める', async () => {
    const { api, set_last_cache_update_time, get_application_config } =
      stub_api({ updated_ids: [], errors: no_errors })

    await api.delete_updated_gkill_caches()

    expect(get_application_config).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_cache).not.toHaveBeenCalled()
    expect(mocks.delete_gkill_kyou_caches).not.toHaveBeenCalled()
    expect(set_last_cache_update_time).toHaveBeenCalledTimes(1)
  })
})
