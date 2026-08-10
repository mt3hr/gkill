import { describe, test, expect } from 'vitest'
import { SavedFindQueryConfig } from '@/classes/datas/config/saved-find-query-config'

/**
 * FindKyouQuery の新形式(null判定)全41フィールドを持つJSON。
 * null=フィルタ未使用 / 非nullの空配列=有効だが空指定。use_* フラグは存在しない
 */
function makeMinimalFindKyouQueryJson(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    query_id: 'test-id',
    update_cache: false,
    keywords: '',
    words_and: false,
    words: null,
    not_words: null,
    timeis_keywords: '',
    timeis_words_and: false,
    timeis_words: null,
    timeis_not_words: null,
    timeis_tags: null,
    timeis_tags_and: false,
    tags: [],
    hide_tags: [],
    tags_and: false,
    reps: [],
    rep_types: null,
    map_latitude: null,
    map_longitude: null,
    map_radius: null,
    calendar_start_date: null,
    calendar_end_date: null,
    plaing_time: null,
    period_of_time_start_time_second: null,
    period_of_time_end_time_second: null,
    period_of_time_week_of_days: null,
    devices_in_sidebar: [],
    rep_types_in_sidebar: [],
    is_enable_map_circle_in_sidebar: false,
    is_image_only: false,
    is_focus_kyou_in_list_view: false,
    mi_board_name: null,
    mi_sort_type: 'estimate_start_time',
    mi_check_state: 'uncheck',
    for_mi: false,
    include_create_mi: true,
    include_check_mi: false,
    include_limit_mi: false,
    include_start_mi: false,
    include_end_mi: false,
    include_end_timeis: true,
    ...overrides,
  }
}

function makeItemJson(id: string, title: string, query_overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: id,
    title: title,
    find_kyou_query: makeMinimalFindKyouQueryJson(query_overrides),
  }
}

describe('SavedFindQueryConfig', () => {
  describe('parse()', () => {
    test.each([
      { name: 'null', json: null },
      { name: 'undefined', json: undefined },
      { name: '空オブジェクト', json: {} },
      { name: '非オブジェクト(文字列)', json: 'not an object' },
      { name: '非オブジェクト(数値)', json: 42 },
    ])('parse($name) は両リスト空のインスタンスを返す（初回起動考慮）', ({ json }) => {
      const config = SavedFindQueryConfig.parse(json)
      expect(config).toBeInstanceOf(SavedFindQueryConfig)
      expect(config.saved_rykv_find_kyou_querys).toEqual([])
      expect(config.saved_mi_find_kyou_querys).toEqual([])
    })

    test('両リストのアイテムが復元される', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [makeItemJson('r1', '仕事のメモ', { words: ['メモ'] })],
        saved_mi_find_kyou_querys: [makeItemJson('m1', '今週のタスク'), makeItemJson('m2', '買い物')],
      })
      expect(config.saved_rykv_find_kyou_querys).toHaveLength(1)
      expect(config.saved_rykv_find_kyou_querys[0].id).toBe('r1')
      expect(config.saved_rykv_find_kyou_querys[0].title).toBe('仕事のメモ')
      expect(config.saved_rykv_find_kyou_querys[0].find_kyou_query.words).toEqual(['メモ'])
      expect(config.saved_mi_find_kyou_querys).toHaveLength(2)
      expect(config.saved_mi_find_kyou_querys.map((item) => item.title)).toEqual(['今週のタスク', '買い物'])
    })

    test('旧形式(use_*入り)のクエリJSONも新形式へ正規化して読む', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [
          makeItemJson('legacy', '旧形式', {
            use_words: false,
            words: ['stale'],
            use_tags: true,
            tags: null,
            use_update_time: false,
            update_time: '2026-01-01T00:00:00.000Z',
          }),
        ],
      })
      const query = config.saved_rykv_find_kyou_querys[0].find_kyou_query
      // use_* キーはインスタンスに生えない（deep_equals のキー数比較の前提）
      expect(Object.keys(query).filter((key) => key.startsWith('use_'))).toEqual([])
      expect(Object.keys(query)).not.toContain('update_time')
      // use_words=false の値は null 化、use_tags=true の null は [] に物質化
      expect(query.words).toBeNull()
      expect(query.tags).toEqual([])
    })

    test('片方のリストが欠けていても、もう片方は復元される', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [makeItemJson('r1', 'ライフログのみ')],
      })
      expect(config.saved_rykv_find_kyou_querys).toHaveLength(1)
      expect(config.saved_mi_find_kyou_querys).toEqual([])
    })

    test('リストが配列でなければ空扱い', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: 'broken',
        saved_mi_find_kyou_querys: { not: 'array' },
      })
      expect(config.saved_rykv_find_kyou_querys).toEqual([])
      expect(config.saved_mi_find_kyou_querys).toEqual([])
    })

    test('find_kyou_query を持たない不正アイテムは除外される', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [
          makeItemJson('ok', '正常'),
          { id: 'broken', title: 'クエリ無し' },
          null,
          'not an object',
        ],
      })
      expect(config.saved_rykv_find_kyou_querys).toHaveLength(1)
      expect(config.saved_rykv_find_kyou_querys[0].id).toBe('ok')
    })

    test('id/title が文字列でないアイテムは空文字にフォールバックする', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [
          { id: 123, title: null, find_kyou_query: makeMinimalFindKyouQueryJson() },
        ],
      })
      expect(config.saved_rykv_find_kyou_querys).toHaveLength(1)
      expect(config.saved_rykv_find_kyou_querys[0].id).toBe('')
      expect(config.saved_rykv_find_kyou_querys[0].title).toBe('')
    })
  })

  describe('to_json()', () => {
    test('空の設定でも両フィールドを持つ', () => {
      const json = new SavedFindQueryConfig().to_json()
      expect(json['saved_rykv_find_kyou_querys']).toEqual([])
      expect(json['saved_mi_find_kyou_querys']).toEqual([])
    })
  })

  describe('parse/to_json round-trip', () => {
    test('id/title/クエリ内容が往復で保存される', () => {
      const src = {
        saved_rykv_find_kyou_querys: [makeItemJson('r1', '旅行の写真', { words: ['写真'] })],
        saved_mi_find_kyou_querys: [makeItemJson('m1', '今週', { mi_board_name: '仕事' })],
      }
      const reparsed = SavedFindQueryConfig.parse(SavedFindQueryConfig.parse(src).to_json())
      expect(reparsed.saved_rykv_find_kyou_querys[0].id).toBe('r1')
      expect(reparsed.saved_rykv_find_kyou_querys[0].title).toBe('旅行の写真')
      expect(reparsed.saved_rykv_find_kyou_querys[0].find_kyou_query.words).toEqual(['写真'])
      expect(reparsed.saved_mi_find_kyou_querys[0].find_kyou_query.mi_board_name).toBe('仕事')
    })
  })

  describe('clone_items() / clone()', () => {
    test('clone_items はクエリの参照を切る（作業用コピーを変えても元が汚れない）', () => {
      const config = SavedFindQueryConfig.parse({
        saved_rykv_find_kyou_querys: [makeItemJson('r1', '元の名前')],
      })
      const cloned = SavedFindQueryConfig.clone_items(config.saved_rykv_find_kyou_querys)
      cloned[0].title = '書き換えた名前'
      cloned[0].find_kyou_query.words = ['書き換えた語']
      expect(config.saved_rykv_find_kyou_querys[0].title).toBe('元の名前')
      expect(config.saved_rykv_find_kyou_querys[0].find_kyou_query.words).toBeNull()
    })

    test('clone は両リストごと参照を切る', () => {
      const config = SavedFindQueryConfig.parse({
        saved_mi_find_kyou_querys: [makeItemJson('m1', 'タスク')],
      })
      const cloned = config.clone()
      cloned.saved_mi_find_kyou_querys.splice(0, 1)
      expect(config.saved_mi_find_kyou_querys).toHaveLength(1)
    })
  })
})
