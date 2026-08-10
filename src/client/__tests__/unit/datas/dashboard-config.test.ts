import { describe, test, expect } from 'vitest'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'

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

describe('DashboardConfig', () => {
  describe('parse()', () => {
    test('parse(null) returns default instance', () => {
      const config = DashboardConfig.parse(null)
      expect(config).toBeInstanceOf(DashboardConfig)
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
      expect(config.dashboard_dnote_find_kyou_query).toBeNull()
    })

    test('parse(undefined) returns default instance', () => {
      const config = DashboardConfig.parse(undefined)
      expect(config).toBeInstanceOf(DashboardConfig)
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
      expect(config.dashboard_dnote_find_kyou_query).toBeNull()
    })

    test('parse({}) returns default instance with null queries', () => {
      const config = DashboardConfig.parse({})
      expect(config).toBeInstanceOf(DashboardConfig)
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
      expect(config.dashboard_dnote_find_kyou_query).toBeNull()
    })

    test('parse with dashboard_mi_find_kyou_query populates mi query', () => {
      const config = DashboardConfig.parse({
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'test-mi' }),
      })
      expect(config.dashboard_mi_find_kyou_query).not.toBeNull()
    })

    test('parse with dashboard_dnote_find_kyou_query populates dnote query', () => {
      const config = DashboardConfig.parse({
        dashboard_dnote_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'test-dnote' }),
      })
      expect(config.dashboard_dnote_find_kyou_query).not.toBeNull()
    })

    test('parse with both queries populates both', () => {
      const config = DashboardConfig.parse({
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'mi-q' }),
        dashboard_dnote_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'dnote-q' }),
      })
      expect(config.dashboard_mi_find_kyou_query).not.toBeNull()
      expect(config.dashboard_dnote_find_kyou_query).not.toBeNull()
    })

    test('parse with non-object value returns default instance', () => {
      const config = DashboardConfig.parse('not an object')
      expect(config).toBeInstanceOf(DashboardConfig)
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
    })

    test('parse with number returns default instance', () => {
      const config = DashboardConfig.parse(42)
      expect(config).toBeInstanceOf(DashboardConfig)
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
    })

    test('旧形式(use_*入り)のクエリJSONも新形式へ正規化して読む', () => {
      const config = DashboardConfig.parse({
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({
          query_id: 'legacy-mi',
          use_words: false,
          words: ['stale'],
          use_reps: true,
          reps: null,
        }),
      })
      const query = config.dashboard_mi_find_kyou_query!
      // use_* キーはインスタンスに生えない
      expect(Object.keys(query).filter((key) => key.startsWith('use_'))).toEqual([])
      // use_words=false の値は null 化、use_reps=true の null は [] に物質化
      expect(query.words).toBeNull()
      expect(query.reps).toEqual([])
    })
  })

  describe('to_json()', () => {
    test('to_json() returns Record<string, unknown>', () => {
      const config = new DashboardConfig()
      const json = config.to_json()
      expect(typeof json).toBe('object')
      expect(json).not.toBeNull()
    })

    test('to_json() with null queries returns null fields', () => {
      const config = new DashboardConfig()
      const json = config.to_json()
      expect(json['dashboard_mi_find_kyou_query']).toBeNull()
      expect(json['dashboard_dnote_find_kyou_query']).toBeNull()
    })

    test('to_json() has required fields', () => {
      const config = new DashboardConfig()
      const json = config.to_json()
      expect('dashboard_mi_find_kyou_query' in json).toBe(true)
      expect('dashboard_dnote_find_kyou_query' in json).toBe(true)
    })
  })

  describe('parse/to_json round-trip', () => {
    test('default config survives round-trip', () => {
      const original = new DashboardConfig()
      const json = original.to_json()
      const parsed = DashboardConfig.parse(json)
      expect(parsed.dashboard_mi_find_kyou_query).toBeNull()
      expect(parsed.dashboard_dnote_find_kyou_query).toBeNull()
    })

    test('config with both queries survives round-trip', () => {
      const src = {
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'mi-1', tags: ['t1'] }),
        dashboard_dnote_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'dnote-1', words: ['w1'] }),
      }
      const parsed = DashboardConfig.parse(src)
      const reparsed = DashboardConfig.parse(parsed.to_json())
      expect(reparsed.dashboard_mi_find_kyou_query).not.toBeNull()
      expect(reparsed.dashboard_dnote_find_kyou_query).not.toBeNull()
      expect(reparsed.dashboard_mi_find_kyou_query?.tags).toEqual(['t1'])
      expect(reparsed.dashboard_dnote_find_kyou_query?.words).toEqual(['w1'])
    })

    test('query_id is preserved through round-trip', () => {
      const src = {
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'my-query-id' }),
      }
      const parsed = DashboardConfig.parse(src)
      const reparsed = DashboardConfig.parse(parsed.to_json())
      expect(reparsed.dashboard_mi_find_kyou_query?.query_id).toBe('my-query-id')
    })
  })
})
