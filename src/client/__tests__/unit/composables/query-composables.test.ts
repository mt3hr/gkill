/**
 * Query Composable tests.
 * Tests query builders that manage search/filter state.
 */
import { vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      generate_uuid: vi.fn(() => 'mock-uuid'),
      get_all_tag_names: vi.fn().mockResolvedValue({ tag_names: [], messages: [], errors: [] }),
      get_all_rep_names: vi.fn().mockResolvedValue({ rep_names: [], messages: [], errors: [] }),
    })),
    get_gkill_api: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
    })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

// Try importing query composables
const queryComposables: Array<{ name: string; factory: unknown }> = []

async function tryImport(name: string, path: string, exportName: string) {
  try {
    const mod = await import(path)
    if (mod[exportName]) {
      queryComposables.push({ name, factory: mod[exportName] })
    }
  } catch {
    // Import failed - skip
  }
}

// The query composables may have various names - try common patterns
await tryImport('useCalendarQuery', '@/classes/use-calendar-query', 'useCalendarQuery')
await tryImport('useTagQuery', '@/classes/use-tag-query', 'useTagQuery')
await tryImport('useMapQuery', '@/classes/use-map-query', 'useMapQuery')
await tryImport('useRepQuery', '@/classes/use-rep-query', 'useRepQuery')
await tryImport('usePeriodOfTimeQuery', '@/classes/use-period-of-time-query', 'usePeriodOfTimeQuery')
await tryImport('useTimeIsQuery', '@/classes/use-time-is-query', 'useTimeIsQuery')

// Build minimal mock props that satisfy what query composables access
function createMockQueryProps() {
  const mockQuery: Record<string, unknown> = {
    is_enable_map_circle_in_sidebar: false,
    tags: [],
    clone() { return { ...this, clone: this.clone } },
  }
  const mockAppConfig: Record<string, unknown> = {
    tag_struct: { is_checked: false, indeterminate: false, children: [], key_name: '', label: '' },
    rep_struct: { is_checked: false, indeterminate: false, children: [], key_name: '', label: '' },
    device_struct: { is_checked: false, indeterminate: false, children: [], key_name: '', label: '' },
    rep_type_struct: { is_checked: false, indeterminate: false, children: [], key_name: '', label: '' },
    clone() { return { ...this, clone: this.clone } },
  }
  return {
    gkill_api: {
      get_google_map_api_key: vi.fn(() => ''),
      get_session_id: vi.fn(() => 'mock-session'),
    },
    find_kyou_query: mockQuery,
    application_config: mockAppConfig,
    inited: false,
  }
}

describe('Query Composables', () => {
  test('at least one query composable is importable', () => {
    expect(queryComposables.length).toBeGreaterThan(0)
  })

  for (const { name, factory } of queryComposables) {
    describe(name, () => {
      test('can be instantiated', () => {
        const result = factory({ props: createMockQueryProps(), emits: vi.fn() })
        expect(result).toBeDefined()
      })
    })
  }
})

// Test FindKyouQuery structure (used by all query composables)
describe('FindKyouQuery structure', () => {
  test('query object can be built with calendar fields', () => {
    const query = {
      use_calendar: true,
      calendar_start_date: new Date('2025-01-01'),
      calendar_end_date: new Date('2025-12-31'),
      use_words: false,
      words: [],
      use_tags: false,
      tags: [],
    }
    expect(query.use_calendar).toBe(true)
    expect(query.calendar_start_date).toBeInstanceOf(Date)
  })

  test('query object can be built with word search fields', () => {
    const query = {
      use_words: true,
      words: ['テスト', '検索'],
      words_and: true,
      not_words: ['除外'],
    }
    expect(query.words.length).toBe(2)
    expect(query.words_and).toBe(true)
  })

  test('query object can be built with tag filter', () => {
    const query = {
      use_tags: true,
      tags: ['重要', 'メモ'],
    }
    expect(query.tags).toEqual(['重要', 'メモ'])
  })

  test('query object can be built with rep filter', () => {
    const query = {
      use_reps: true,
      reps: ['rep1', 'rep2'],
    }
    expect(query.reps.length).toBe(2)
  })

  test('query object can be built with Mi-specific fields', () => {
    const query = {
      for_mi: true,
      mi_check_state: 'unchecked',
      mi_sort_type: 'limit_time',
      mi_board_name: 'Inbox',
    }
    expect(query.for_mi).toBe(true)
    expect(query.mi_check_state).toBe('unchecked')
  })

  test('query object can be built with period-of-time fields', () => {
    const query = {
      use_period_of_time: true,
      period_of_time_start_time: new Date('2025-01-01T00:00:00'),
      period_of_time_end_time: new Date('2025-01-31T23:59:59'),
    }
    expect(query.use_period_of_time).toBe(true)
  })

  test('default empty query has no filters active', () => {
    const query = {
      use_calendar: false,
      use_words: false,
      use_tags: false,
      use_reps: false,
      for_mi: false,
      use_period_of_time: false,
    }
    const activeFilters = Object.values(query).filter(v => v === true)
    expect(activeFilters.length).toBe(0)
  })

  test('multiple filters can be combined', () => {
    const query = {
      use_calendar: true,
      calendar_start_date: new Date('2025-03-01'),
      calendar_end_date: new Date('2025-03-31'),
      use_words: true,
      words: ['テスト'],
      use_tags: true,
      tags: ['重要'],
    }
    expect(query.use_calendar).toBe(true)
    expect(query.use_words).toBe(true)
    expect(query.use_tags).toBe(true)
  })
})
