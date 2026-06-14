import { describe, test, expect, beforeEach } from 'vitest'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'

/** FindKyouQuery.parse_find_kyou_query が要求する全フィールドを持つ最小JSON */
function makeMinimalFindKyouQueryJson(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    query_id: 'test-id',
    use_reps: false,
    use_tags: false,
    update_cache: false,
    use_words: false,
    keywords: '',
    words_and: false,
    words: [],
    not_words: [],
    use_timeis: false,
    timeis_words_and: false,
    timeis_keywords: '',
    timeis_words: [],
    timeis_not_words: [],
    use_timeis_tags: false,
    timeis_tags: [],
    timeis_tags_and: false,
    tags: [],
    hide_tags: [],
    tags_and: false,
    use_map: false,
    map_latitude: 0,
    map_longitude: 0,
    map_radius: 0,
    use_calendar: false,
    calendar_start_date: null,
    calendar_end_date: null,
    use_plaing: false,
    plaing_time: null,
    reps: [],
    is_image_only: false,
    devices_in_sidebar: [],
    rep_types_in_sidebar: [],
    use_update_time: false,
    update_time: null,
    is_enable_map_circle_in_sidebar: false,
    use_rep_types: false,
    rep_types: [],
    use_mi_board_name: false,
    mi_board_name: '',
    use_mi_sort_type: false,
    mi_sort_type: 'create_time',
    use_mi_check_state: false,
    mi_check_state: 'all',
    for_mi: false,
    use_period_of_time: false,
    period_of_time_start_time_second: null,
    period_of_time_end_time_second: null,
    period_of_time_week_of_days: [],
    include_create_mi: false,
    include_check_mi: false,
    include_limit_mi: false,
    include_start_mi: false,
    include_end_mi: false,
    include_end_timeis: false,
    use_include_id: false,
    is_focus_kyou_in_list_view: false,
    ...overrides,
  }
}

describe('DashboardConfig', () => {
  test('can be instantiated', () => {
    const config = new DashboardConfig()
    expect(config).toBeInstanceOf(DashboardConfig)
  })

  describe('default field values', () => {
    let config: DashboardConfig

    beforeEach(() => {
      config = new DashboardConfig()
    })

    test('dashboard_mi_find_kyou_query defaults to null', () => {
      expect(config.dashboard_mi_find_kyou_query).toBeNull()
    })

    test('dashboard_dnote_find_kyou_query defaults to null', () => {
      expect(config.dashboard_dnote_find_kyou_query).toBeNull()
    })
  })

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

    test('backward compat: dashboard_default_find_kyou_query maps to mi query', () => {
      const config = DashboardConfig.parse({
        dashboard_default_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'legacy' }),
      })
      // 旧フィールド名はmiクエリにマイグレーション
      expect(config.dashboard_mi_find_kyou_query).not.toBeNull()
    })

    test('new mi field takes priority over legacy field', () => {
      const config = DashboardConfig.parse({
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'new-mi' }),
        dashboard_default_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'legacy' }),
      })
      // 新フィールドがある場合、旧フィールドは無視される
      expect(config.dashboard_mi_find_kyou_query).not.toBeNull()
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
        dashboard_mi_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'mi-1', use_tags: true }),
        dashboard_dnote_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'dnote-1', use_words: true }),
      }
      const parsed = DashboardConfig.parse(src)
      const reparsed = DashboardConfig.parse(parsed.to_json())
      expect(reparsed.dashboard_mi_find_kyou_query).not.toBeNull()
      expect(reparsed.dashboard_dnote_find_kyou_query).not.toBeNull()
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
