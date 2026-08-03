import { beforeEach, describe, expect, test, vi } from 'vitest'

/**
 * Kyou.load_all() が1件あたりに飛ばすリクエスト数の回帰テスト。
 *
 * 一覧は取得した全件に load_all() を撒くので、ここが1本増えるだけで
 * 1,000件表示なら1,000本増える。実際に次の2つが起きていた。
 *   - load_attached_histories が load_all と load_attached_datas の
 *     両方から呼ばれ、/api/get_kyou が2回飛んでいた
 *   - load_attached_timeis が Kyou 1件ごとに get_application_config を呼び、
 *     その内部の load_all() でさらに5〜6往復していた
 *
 * どちらも結果は正しいままなので、回数を数えないと気づけない。
 */

// GkillAPI はシングルトンで import しただけで副作用が走るためモジュールごと差し替える
const mockApi: Record<string, ReturnType<typeof vi.fn>> = {}

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_gkill_api: () => mockApi,
    get_instance: () => mockApi,
  },
}))

describe('Kyou.load_all のリクエスト本数', () => {
  beforeEach(() => {
    for (const key of Object.keys(mockApi)) {
      delete mockApi[key]
    }
    mockApi.get_kyou = vi.fn().mockResolvedValue({ errors: [], kyou_histories: [] })
    mockApi.get_kmemo = vi.fn().mockResolvedValue({ errors: [], kmemo_histories: [] })
    mockApi.get_tags_by_target_id = vi.fn().mockResolvedValue({ errors: [], tags: [] })
    mockApi.get_texts_by_target_id = vi.fn().mockResolvedValue({ errors: [], texts: [] })
    mockApi.get_notifications_by_target_id = vi.fn().mockResolvedValue({ errors: [], notifications: [] })
    mockApi.get_kyous = vi.fn().mockResolvedValue({ errors: [], kyous: [] })
    mockApi.get_application_config = vi.fn().mockResolvedValue({
      errors: [],
      application_config: { for_share_kyou: true },
    })
    // 各ページが読み込み時に保存している想定のアプリ設定
    mockApi.get_saved_application_config = vi.fn().mockReturnValue({ for_share_kyou: true })
  })

  test('履歴取得(/api/get_kyou)は1回だけ', async () => {
    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = 'kyou-1'
    kyou.data_type = 'kmemo'

    await kyou.load_all()

    expect(mockApi.get_kyou).toHaveBeenCalledTimes(1)
  })

  test('保存済みのアプリ設定があれば get_application_config を呼ばない', async () => {
    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = 'kyou-2'
    kyou.data_type = 'kmemo'

    await kyou.load_all()

    expect(mockApi.get_application_config).not.toHaveBeenCalled()
  })

  test('保存済みのアプリ設定が無いときは従来どおり取りに行く', async () => {
    mockApi.get_saved_application_config = vi.fn().mockReturnValue(null)

    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = 'kyou-3'
    kyou.data_type = 'kmemo'

    await kyou.load_all()

    expect(mockApi.get_application_config).toHaveBeenCalled()
  })
})
