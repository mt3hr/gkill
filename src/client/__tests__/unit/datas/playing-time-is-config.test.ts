import { describe, test, expect } from 'vitest'
import { PlayingTimeIsConfig } from '@/classes/datas/config/playing-time-is-config'

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
    playing_time: null,
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

describe('PlayingTimeIsConfig', () => {
  describe('parse()', () => {
    test('parse(null) returns default instance', () => {
      const config = PlayingTimeIsConfig.parse(null)
      expect(config).toBeInstanceOf(PlayingTimeIsConfig)
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('parse(undefined) returns default instance', () => {
      const config = PlayingTimeIsConfig.parse(undefined)
      expect(config).toBeInstanceOf(PlayingTimeIsConfig)
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('parse({}) returns default instance with null query', () => {
      const config = PlayingTimeIsConfig.parse({})
      expect(config).toBeInstanceOf(PlayingTimeIsConfig)
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('parse with playing_timeis_find_kyou_query populates query', () => {
      const config = PlayingTimeIsConfig.parse({
        playing_timeis_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'test-playing' }),
      })
      expect(config.playing_timeis_find_kyou_query).not.toBeNull()
    })

    test('parse with explicit null query keeps null', () => {
      const config = PlayingTimeIsConfig.parse({ playing_timeis_find_kyou_query: null })
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('parse with non-object value returns default instance', () => {
      const config = PlayingTimeIsConfig.parse('not an object')
      expect(config).toBeInstanceOf(PlayingTimeIsConfig)
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('parse with number returns default instance', () => {
      const config = PlayingTimeIsConfig.parse(42)
      expect(config).toBeInstanceOf(PlayingTimeIsConfig)
      expect(config.playing_timeis_find_kyou_query).toBeNull()
    })

    test('旧形式(use_*入り)のクエリJSONも新形式へ正規化して読む', () => {
      const config = PlayingTimeIsConfig.parse({
        playing_timeis_find_kyou_query: makeMinimalFindKyouQueryJson({
          query_id: 'legacy-playing',
          use_tags: false,
          tags: ['stale'],
          use_playing: false,
          playing_time: '2026-01-01T00:00:00.000Z',
        }),
      })
      const query = config.playing_timeis_find_kyou_query!
      // use_* キーはインスタンスに生えない
      expect(Object.keys(query).filter((key) => key.startsWith('use_'))).toEqual([])
      // use_X=false の値は null 化される
      expect(query.tags).toBeNull()
      expect(query.playing_time).toBeNull()
    })
  })

  describe('to_json()', () => {
    test('to_json() returns Record<string, unknown>', () => {
      const config = new PlayingTimeIsConfig()
      const json = config.to_json()
      expect(typeof json).toBe('object')
      expect(json).not.toBeNull()
    })

    test('to_json() with null query returns null field', () => {
      const config = new PlayingTimeIsConfig()
      const json = config.to_json()
      expect(json['playing_timeis_find_kyou_query']).toBeNull()
    })

    test('to_json() has required field', () => {
      const config = new PlayingTimeIsConfig()
      const json = config.to_json()
      expect('playing_timeis_find_kyou_query' in json).toBe(true)
    })
  })

  describe('parse/to_json round-trip', () => {
    test('default config survives round-trip', () => {
      const original = new PlayingTimeIsConfig()
      const json = original.to_json()
      const parsed = PlayingTimeIsConfig.parse(json)
      expect(parsed.playing_timeis_find_kyou_query).toBeNull()
    })

    test('config with query survives round-trip', () => {
      const src = {
        playing_timeis_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'playing-1', tags: ['t1'], reps: ['rep1'] }),
      }
      const parsed = PlayingTimeIsConfig.parse(src)
      const reparsed = PlayingTimeIsConfig.parse(parsed.to_json())
      expect(reparsed.playing_timeis_find_kyou_query).not.toBeNull()
      expect(reparsed.playing_timeis_find_kyou_query?.tags).toEqual(['t1'])
      expect(reparsed.playing_timeis_find_kyou_query?.reps).toEqual(['rep1'])
    })

    test('query_id is preserved through round-trip', () => {
      const src = {
        playing_timeis_find_kyou_query: makeMinimalFindKyouQueryJson({ query_id: 'my-query-id' }),
      }
      const parsed = PlayingTimeIsConfig.parse(src)
      const reparsed = PlayingTimeIsConfig.parse(parsed.to_json())
      expect(reparsed.playing_timeis_find_kyou_query?.query_id).toBe('my-query-id')
    })
  })
})
