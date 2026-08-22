import { beforeEach, describe, expect, test, vi } from 'vitest'

/**
 * Kyou.load_typed_datas() のディスパッチのテスト。
 *
 * 以前は11本の `if (data_type.startsWith(...))` を並べたうえで、
 * 末尾のプラグイン判定で同じ11個の startsWith をもう一度評価していた。
 * 単一ディスパッチへ寄せたので、次の2つを固定しておく。
 *   - data_type ごとに **1つだけ** 型別データを読むこと
 *   - `mirekyou_*` は `mi` で始まるので、**Mi ではなく MiReKyou** を読むこと
 *     (AGENTS.md の「mirekyou を mi より先に判定する」)。
 *     いまは判定表を長いプレフィックス順に並べ替えることで構造的に保証している
 */

const mockApi: Record<string, ReturnType<typeof vi.fn>> = {}

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_gkill_api: () => mockApi,
    get_instance: () => mockApi,
  },
}))

const typed_api_methods = [
  'get_kmemo',
  'get_kc',
  'get_urlog',
  'get_nlog',
  'get_timeis',
  'get_mi',
  'get_lantana',
  'get_idf_kyou',
  'get_git_commit_log',
  'get_rekyou',
  'get_mirekyou',
] as const

// data_type は実際に FindKyous が返す文字列（射影の接尾辞つきを含む）
const dispatch_cases: Array<[string, string]> = [
  ['kmemo', 'get_kmemo'],
  ['kc', 'get_kc'],
  ['urlog', 'get_urlog'],
  ['nlog', 'get_nlog'],
  ['timeis_start', 'get_timeis'],
  ['timeis_end', 'get_timeis'],
  ['mi_create', 'get_mi'],
  ['mi_check', 'get_mi'],
  ['mi_limit', 'get_mi'],
  ['mi_start', 'get_mi'],
  ['mi_end', 'get_mi'],
  ['mirekyou_create', 'get_mirekyou'],
  ['mirekyou_check', 'get_mirekyou'],
  ['mirekyou_limit', 'get_mirekyou'],
  ['mirekyou_start', 'get_mirekyou'],
  ['mirekyou_end', 'get_mirekyou'],
  ['lantana', 'get_lantana'],
  ['idf', 'get_idf_kyou'],
  ['git_commit_log', 'get_git_commit_log'],
  ['rekyou', 'get_rekyou'],
]

describe('Kyou.load_typed_datas のディスパッチ', () => {
  beforeEach(() => {
    for (const key of Object.keys(mockApi)) {
      delete mockApi[key]
    }
    for (const method of typed_api_methods) {
      mockApi[method] = vi.fn().mockResolvedValue({ errors: [] })
    }
  })

  test.each(dispatch_cases)('data_type=%s は %s だけを呼ぶ', async (data_type, expected_method) => {
    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = 'kyou-1'
    kyou.data_type = data_type

    await kyou.load_typed_datas()

    for (const method of typed_api_methods) {
      expect(mockApi[method], `${data_type} -> ${method}`)
        .toHaveBeenCalledTimes(method === expected_method ? 1 : 0)
    }
    expect(kyou.typed_plugin).toBeNull()
    expect(kyou.is_typed_data_loaded).toBe(true)
  })

  test('既知のプレフィックスに当たらない data_type はプラグインKyouとして扱う', async () => {
    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = 'kyou-2'
    kyou.rep_name = 'gkill_plugin_claudecode'
    kyou.data_type = 'claude_conversation'

    await kyou.load_typed_datas()

    for (const method of typed_api_methods) {
      expect(mockApi[method]).not.toHaveBeenCalled()
    }
    expect(kyou.typed_plugin).toEqual({ rep_name: 'gkill_plugin_claudecode' })
    expect(kyou.is_typed_data_loaded).toBe(true)
  })

  test('idが空のKyouは何も読まず、プラグインとも判定しない', async () => {
    const { Kyou } = await import('@/classes/datas/kyou')
    const kyou = new Kyou()
    kyou.id = ''
    kyou.data_type = ''

    await kyou.load_typed_datas()

    for (const method of typed_api_methods) {
      expect(mockApi[method]).not.toHaveBeenCalled()
    }
    expect(kyou.typed_plugin).toBeNull()
    // 中身の入ったKyouへ差し替わったときに読み直させるので、立ててはいけない
    expect(kyou.is_typed_data_loaded).toBe(false)
  })
})
