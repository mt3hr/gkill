import { vi } from 'vitest'

/**
 * 各データモデルの load_attached_histories / load_attached_datas は
 *   1. 自分のIDで履歴取得APIを呼ぶ
 *   2. errors が返ったら attached_histories を触らずにそのまま返す
 *   3. 成功したら attached_histories に詰める
 *   4. abort 由来の例外だけは握りつぶす
 * という共通の形をしている。エンティティごとに呼ぶAPIとレスポンスのキーが違い、
 * ここを取り違えると「履歴だけ常に空」という静かな壊れ方をする。
 *
 * 削除した「既定値が空配列であること」だけを見るテストの代わりに、
 * この実際の分岐をエンティティ横断で確認する。
 */

// GkillAPI はシングルトンで、import しただけで副作用が走るためモジュールごと差し替える
const mockApi: Record<string, ReturnType<typeof vi.fn>> = {}

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_gkill_api: () => mockApi,
  },
}))

type EntityCase = {
  name: string
  /** load_attached_histories が呼ぶ GkillAPI のメソッド名 */
  apiMethod: string
  /** レスポンス中の履歴配列のキー */
  historiesKey: string
  /**
   * load_attached_datas が load_attached_histories に委譲するか。
   * ReKyou / MiReKyou は委譲せず clear_attached_histories を呼ぶだけなので false。
   */
  delegatesFromLoadAttachedDatas?: boolean
  /** インスタンスを作る */
  create: () => Promise<{
    id: string
    attached_histories: unknown[]
    load_attached_histories(): Promise<unknown[]>
    load_attached_datas(): Promise<unknown[]>
  }>
}

const cases: EntityCase[] = [
  {
    name: 'Kmemo',
    apiMethod: 'get_kmemo',
    historiesKey: 'kmemo_histories',
    create: async () => new (await import('@/classes/datas/kmemo')).Kmemo(),
  },
  {
    name: 'Tag',
    apiMethod: 'get_tag_histories_by_tag_id',
    historiesKey: 'tag_histories',
    create: async () => new (await import('@/classes/datas/tag')).Tag(),
  },
  {
    name: 'Text',
    apiMethod: 'get_text_history_by_text_id',
    historiesKey: 'text_histories',
    create: async () => new (await import('@/classes/datas/text')).Text(),
  },
  {
    name: 'TimeIs',
    apiMethod: 'get_timeis',
    historiesKey: 'timeis_histories',
    create: async () => new (await import('@/classes/datas/time-is')).TimeIs(),
  },
  {
    name: 'Lantana',
    apiMethod: 'get_lantana',
    historiesKey: 'lantana_histories',
    create: async () => new (await import('@/classes/datas/lantana')).Lantana(),
  },
  {
    name: 'KC',
    apiMethod: 'get_kc',
    historiesKey: 'kc_histories',
    create: async () => new (await import('@/classes/datas/kc')).KC(),
  },
  {
    name: 'Nlog',
    apiMethod: 'get_nlog',
    historiesKey: 'nlog_histories',
    create: async () => new (await import('@/classes/datas/nlog')).Nlog(),
  },
  {
    name: 'URLog',
    apiMethod: 'get_urlog',
    historiesKey: 'urlog_histories',
    create: async () => new (await import('@/classes/datas/ur-log')).URLog(),
  },
  {
    name: 'ReKyou',
    delegatesFromLoadAttachedDatas: false,
    apiMethod: 'get_rekyou',
    historiesKey: 'rekyou_histories',
    create: async () => new (await import('@/classes/datas/re-kyou')).ReKyou(),
  },
  {
    name: 'MiReKyou',
    delegatesFromLoadAttachedDatas: false,
    apiMethod: 'get_mirekyou',
    historiesKey: 'mirekyou_histories',
    create: async () => new (await import('@/classes/datas/mi-re-kyou')).MiReKyou(),
  },
  {
    name: 'GitCommitLog',
    apiMethod: 'get_git_commit_log',
    historiesKey: 'git_commit_log_histories',
    create: async () => new (await import('@/classes/datas/git-commit-log')).GitCommitLog(),
  },
]

describe('load_attached_histories', () => {
  beforeEach(() => {
    for (const key of Object.keys(mockApi)) delete mockApi[key]
  })

  describe.each(cases)('$name', ({ apiMethod, historiesKey, create }) => {
    test('自分のIDで履歴APIを呼び、結果を attached_histories に入れる', async () => {
      const history = { id: 'entity-1', marker: 'history-entry' }
      mockApi[apiMethod] = vi.fn().mockResolvedValue({ errors: [], [historiesKey]: [history] })

      const entity = await create()
      entity.id = 'entity-1'
      const errors = await entity.load_attached_histories()

      expect(errors).toEqual([])
      expect(mockApi[apiMethod]).toHaveBeenCalledTimes(1)
      expect(mockApi[apiMethod].mock.calls[0][0]).toMatchObject({ id: 'entity-1' })
      expect(entity.attached_histories).toEqual([history])
    })

    test('errors が返ったら attached_histories を書き換えずにエラーを返す', async () => {
      const apiErrors = [{ error_code: 'ERR000001', error_message: '失敗' }]
      mockApi[apiMethod] = vi.fn().mockResolvedValue({ errors: apiErrors, [historiesKey]: [{ id: 'leaked' }] })

      const entity = await create()
      const errors = await entity.load_attached_histories()

      expect(errors).toEqual(apiErrors)
      expect(entity.attached_histories).toEqual([])
    })

  })

  // load_attached_datas は abort 由来の例外だけを握りつぶす。
  //
  // 画面遷移や検索のやり直しで AbortController が発火したときに
  // 呼び出し側へ例外を投げないための処理で、
  // `try { return await this.load_attached_histories() } catch { ... }`
  // の await が抜けていると catch 節に入らず素通りしてしまう
  // （async 関数で `return promise` は try/catch に捕まらない）。
  describe.each(cases.filter((c) => c.delegatesFromLoadAttachedDatas !== false))(
    '$name load_attached_datas',
    ({ apiMethod, create }) => {
      test('abort 例外を握りつぶす', async () => {
        mockApi[apiMethod] = vi.fn().mockRejectedValue(new Error('signal is aborted without reason'))

        const entity = await create()
        const errors = await entity.load_attached_datas()

        expect(errors).toEqual([])
      })

      test('abort 以外の例外も呼び出し側へ投げない', async () => {
        mockApi[apiMethod] = vi.fn().mockRejectedValue(new Error('network down'))

        const entity = await create()
        const errors = await entity.load_attached_datas()

        expect(errors).toEqual([])
      })
    },
  )
})
