/**
 * Query Composable tests.
 * Tests query builders that manage search/filter state.
 *
 * FindKyouQuery は use_* フラグ全廃後の null 判定意味論で動く:
 *   - null = フィルタ未使用 / 非nullの空配列 [] = フィルタ有効だが空指定
 *   - チェックボックスのUI状態は各コンポーザブルのローカルref
 *     （クエリ上の null 判定から導出。query_id が同じ間はオフ(null)着信でも
 *       ローカルの入力値を保持し、query_id が変われば着信値でリセットする）
 */
import { vi } from 'vitest'
import { nextTick, reactive } from 'vue'
import moment from 'moment'

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

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { useKeywordQuery } from '@/classes/use-keyword-query'
import { useTimeIsQuery } from '@/classes/use-time-is-query'
import { useCalendarQuery } from '@/classes/use-calendar-query'
import { useMapQuery } from '@/classes/use-map-query'
import { usePeriodOfTimeQuery } from '@/classes/use-period-of-time-query'

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
  const mockAppConfig: Record<string, unknown> = {
    tag_struct: { is_checked: false, indeterminate: false, children: [], key: '', tag_name: '', check_when_inited: false },
    rep_struct: { is_checked: false, indeterminate: false, children: [], key: '', rep_name: '' },
    device_struct: { is_checked: false, indeterminate: false, children: [], key: '', device_name: '' },
    rep_type_struct: { is_checked: false, indeterminate: false, children: [], key: '', rep_type_name: '' },
    google_map_api_key: '',
    clone() { return { ...this, clone: this.clone } },
  }
  return {
    gkill_api: {
      get_google_map_api_key: vi.fn(() => ''),
      get_session_id: vi.fn(() => 'mock-session'),
    },
    find_kyou_query: new FindKyouQuery(),
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
        const result = (factory as (options: unknown) => unknown)({ props: createMockQueryProps(), emits: vi.fn() })
        expect(result).toBeDefined()
      })
    })
  }
})

// ── 挙動テスト用の小道具 ──

// ApplicationConfig の実物は req_res との循環importを引き込むため、
// コンポーザブルが触るフィールドだけ持つ fake を使う
function make_fake_application_config(): Record<string, unknown> {
  const config: Record<string, unknown> = {
    tag_struct: { key: '', tag_name: '', check_when_inited: false, is_checked: false, indeterminate: false, is_force_hide: false, children: [] },
    google_map_api_key: '',
  }
  config.clone = () => ({ ...config })
  return config
}

// props watcher（async含む）を消化する
async function flush_watchers(times = 4): Promise<void> {
  for (let i = 0; i < times; i++) {
    await nextTick()
  }
}

describe('useKeywordQuery: words の null 判定', () => {
  function create_view(initial: FindKyouQuery) {
    const props = reactive({ find_kyou_query: initial })
    const view = useKeywordQuery({ props: props as never, emits: vi.fn() as never })
    return { props, view }
  }

  test('チェックオン(words非null)着信で立ち、オフ(null)着信で倒れる。keywords文字列は失われない', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_words(), '既定クエリは words=null(未使用)').toBe(false)

    const q_on = new FindKyouQuery()
    q_on.query_id = 'q1'
    q_on.words = []
    q_on.not_words = []
    q_on.keywords = '写真'
    props.find_kyou_query = q_on
    await flush_watchers()
    expect(view.get_use_words()).toBe(true)
    expect(view.get_keywords()).toBe('写真')

    const q_off = q_on.clone()
    q_off.words = null
    q_off.not_words = null
    props.find_kyou_query = q_off
    await flush_watchers()
    expect(view.get_use_words()).toBe(false)
    // keywords はUI専用の非nullフィールドとしてクエリ上に残る（オフでも入力は消えない）
    expect(view.get_keywords()).toBe('写真')

    // 即時トグルで再び有効化しても値が復活している
    const q_re_on = q_off.clone()
    q_re_on.words = []
    q_re_on.not_words = []
    props.find_kyou_query = q_re_on
    await flush_watchers()
    expect(view.get_use_words()).toBe(true)
    expect(view.get_keywords()).toBe('写真')
  })
})

describe('useTimeIsQuery: timeis_words / timeis_tags の null 判定とローカル保持', () => {
  function create_view(initial: FindKyouQuery) {
    const props = reactive({
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
      inited: true,
    })
    const view = useTimeIsQuery({ props: props as never, emits: vi.fn() as never })
    return { props, view }
  }

  test('null判定からチェック状態が導出される（グループ・タグ絞りの2段）', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_timeis()).toBe(false)
    expect(view.get_use_timeis_tags()).toBe(false)

    const q_on = new FindKyouQuery()
    q_on.query_id = 'q1'
    q_on.timeis_words = []
    q_on.timeis_not_words = []
    q_on.timeis_tags = ['タグA']
    props.find_kyou_query = q_on
    await flush_watchers()
    expect(view.get_use_timeis()).toBe(true)
    expect(view.get_use_timeis_tags()).toBe(true)
  })

  test('同一query_idのオフ(null)着信ではローカルのキーワードを保持し、query_idが変わればリセットする', async () => {
    const { props, view } = create_view(new FindKyouQuery())

    const q_on = new FindKyouQuery()
    q_on.query_id = 'q1'
    q_on.timeis_words = []
    q_on.timeis_not_words = []
    q_on.timeis_keywords = '作業'
    props.find_kyou_query = q_on
    await flush_watchers()
    expect(view.get_use_timeis()).toBe(true)
    expect(view.get_timeis_keywords()).toBe('作業')

    // チェックオフ: クエリ上は null（キーワード文字列も親からは空で来る想定の最悪ケース）
    const q_off = new FindKyouQuery()
    q_off.query_id = 'q1'
    props.find_kyou_query = q_off
    await flush_watchers()
    expect(view.get_use_timeis()).toBe(false)
    expect(view.get_timeis_keywords(), '同一query_id内の即時トグルで値が復活する').toBe('作業')

    // 別列（query_id変化）へは着信値でリセット
    const q2 = new FindKyouQuery()
    q2.query_id = 'q2'
    props.find_kyou_query = q2
    await flush_watchers()
    expect(view.get_timeis_keywords()).toBe('')
  })
})

describe('useCalendarQuery: calendar日付の null 判定とローカル保持', () => {
  function create_view(initial: FindKyouQuery) {
    const props = reactive({
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
      inited: true,
    })
    const view = useCalendarQuery({ props: props as never, emits: vi.fn() as never })
    return { props, view }
  }

  function make_dated_query(query_id: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = query_id
    query.calendar_start_date = new Date(2026, 6, 1)
    query.calendar_end_date = new Date(2026, 6, 3, 23, 59, 59, 999)
    return query
  }

  test('日付非null着信で立ち、日付レンジが構築される', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_calendar(), '既定クエリは両方null(未使用)').toBe(false)

    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(true)
    expect(moment(view.get_start_date()).format('YYYY-MM-DD')).toBe('2026-07-01')
    expect(moment(view.get_end_date()).format('YYYY-MM-DD')).toBe('2026-07-03')
  })

  test('同一query_idの両null着信では日付選択もチェック状態も触らず、query_idが変わればリセットする', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    // 同一列での両null着信（チェックオフ / ピッカーでの選択解除）
    const q_off = new FindKyouQuery()
    q_off.query_id = 'q1'
    props.find_kyou_query = q_off
    await flush_watchers()
    expect(view.get_use_calendar(), '着信クエリでチェックを外すと同じ日付の2回クリックでチェックが勝手に外れる').toBe(true)
    expect(moment(view.get_start_date()).format('YYYY-MM-DD'), '同一query_id内の即時トグルで値が復活する').toBe('2026-07-01')

    // 別列（query_id変化）へは着信値（null=選択なし）でリセット
    const q2 = new FindKyouQuery()
    q2.query_id = 'q2'
    props.find_kyou_query = q2
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(false)
    expect(view.get_start_date()).toBeNull()
    expect(view.get_end_date()).toBeNull()
  })

  test('同じ日付の連続クリック（Vuetifyのレンジ解除）でチェックは外れない', async () => {
    const { props, view } = create_view(new FindKyouQuery())

    // 単一日選択の状態。同日なので dates は1要素で保持される
    const single_day = new FindKyouQuery()
    single_day.query_id = 'q1'
    single_day.calendar_start_date = new Date(2026, 7, 1)
    single_day.calendar_end_date = new Date(2026, 7, 1, 23, 59, 59, 999)
    props.find_kyou_query = single_day
    await flush_watchers()
    expect(view.dates.value).toHaveLength(1)
    expect(view.get_use_calendar()).toBe(true)

    // 同じ日付をもう一度クリックすると Vuetify が空配列を返す
    view.clicked_date([])
    expect(view.get_start_date()).toBeNull()
    expect(view.get_end_date()).toBeNull()
    expect(view.get_use_calendar(), '選択が空になってもカレンダー条件は有効なまま').toBe(true)

    // 列のクエリが null/null で返ってきてもチェックは外れない
    const echo = new FindKyouQuery()
    echo.query_id = 'q1'
    props.find_kyou_query = echo
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(true)
    expect(view.get_start_date()).toBeNull()
  })

  test('クリアはローカルの日付選択を捨てる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    view.clicked_clear_calendar_button()
    expect(view.get_start_date(), '既定期間が未設定だと着信はnull/nullなので、ここで落とさないと古い選択が残る').toBeNull()
    expect(view.get_end_date()).toBeNull()
  })
})

describe('useMapQuery: 地図3値の null 判定とローカル保持', () => {
  function create_view(initial: FindKyouQuery) {
    const props = reactive({
      gkill_api: { get_google_map_api_key: () => '' },
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
    })
    const view = useMapQuery({ props: props as never, emits: vi.fn() as never })
    return { props, view }
  }

  test('非null着信で立って座標が同期され、null着信では座標を保持してチェックだけ倒れる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_map()).toBe(false)

    const q_on = new FindKyouQuery()
    q_on.query_id = 'q1'
    q_on.map_latitude = 10.5
    q_on.map_longitude = 20.25
    q_on.map_radius = 300
    props.find_kyou_query = q_on
    await flush_watchers()
    expect(view.get_use_map()).toBe(true)
    expect(view.get_latitude()).toBe(10.5)
    expect(view.get_longitude()).toBe(20.25)
    expect(view.get_radius()).toBe(300)

    const q_off = new FindKyouQuery()
    q_off.query_id = 'q1'
    props.find_kyou_query = q_off
    await flush_watchers()
    expect(view.get_use_map()).toBe(false)
    expect(view.get_latitude(), 'null着信(未使用)ではローカル座標を保持する').toBe(10.5)
    expect(view.get_longitude()).toBe(20.25)
    expect(view.get_radius()).toBe(300)
  })

  test('3値そろわない半端な着信ではチェックが立たない', async () => {
    // サーバのゲート(HasMapFilter)は緯度・経度・半径の3値そろって初めて有効。
    // 緯度だけで立ててしまうと「チェックは入っているのに絞り込まれない」表示になる
    const { props, view } = create_view(new FindKyouQuery())

    const q_partial = new FindKyouQuery()
    q_partial.query_id = 'q1'
    q_partial.map_latitude = 10.5
    props.find_kyou_query = q_partial
    await flush_watchers()
    expect(view.get_use_map(), '緯度だけではサーバが地図フィルタを適用しない').toBe(false)

    const q_no_radius = new FindKyouQuery()
    q_no_radius.query_id = 'q2'
    q_no_radius.map_latitude = 10.5
    q_no_radius.map_longitude = 20.25
    props.find_kyou_query = q_no_radius
    await flush_watchers()
    expect(view.get_use_map(), '半径が無ければ地図フィルタは成立しない').toBe(false)
  })
})

describe('usePeriodOfTimeQuery: week_of_days の null 判定', () => {
  function create_view(initial: FindKyouQuery) {
    const props = reactive({
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
    })
    const view = usePeriodOfTimeQuery({ props: props as never, emits: vi.fn() as never })
    return { props, view }
  }

  test('week_of_days 非null着信で立ち、null着信で倒れる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_period_of_time()).toBe(false)

    const q_on = new FindKyouQuery()
    q_on.query_id = 'q1'
    q_on.period_of_time_week_of_days = [1, 2]
    props.find_kyou_query = q_on
    await flush_watchers()
    expect(view.get_use_period_of_time()).toBe(true)
    expect(view.get_period_of_time_week_of_days()).toEqual([1, 2])

    const q_off = new FindKyouQuery()
    q_off.query_id = 'q1'
    props.find_kyou_query = q_off
    await flush_watchers()
    expect(view.get_use_period_of_time()).toBe(false)
    expect(view.get_period_of_time_week_of_days()).toEqual([])
  })
})

// FindKyouQuery 本体の null 判定意味論（全クエリコンポーザブルの前提）
describe('FindKyouQuery null判定の意味論', () => {
  test('コンストラクタ既定: nullable群はnull、tags/repsは[](有効・チェック0個=0件)', () => {
    const query = new FindKyouQuery()
    expect(query.words).toBeNull()
    expect(query.not_words).toBeNull()
    expect(query.timeis_words).toBeNull()
    expect(query.timeis_not_words).toBeNull()
    expect(query.timeis_tags).toBeNull()
    expect(query.rep_types).toBeNull()
    expect(query.map_latitude).toBeNull()
    expect(query.map_longitude).toBeNull()
    expect(query.map_radius).toBeNull()
    expect(query.calendar_start_date).toBeNull()
    expect(query.calendar_end_date).toBeNull()
    expect(query.plaing_time).toBeNull()
    expect(query.period_of_time_start_time_second).toBeNull()
    expect(query.period_of_time_end_time_second).toBeNull()
    expect(query.period_of_time_week_of_days).toBeNull()
    expect(query.mi_board_name, 'null=「すべて」').toBeNull()
    // 旧 use_tags=true+tags=[] / use_reps=true+reps=[] の厳密等価
    // （ダッシュボード等がコンストラクタ既定のままwireへ乗せるため）
    expect(query.tags).toEqual([])
    expect(query.reps).toEqual([])
  })

  test('clone は null と [] を相互変換しない', () => {
    const query = new FindKyouQuery()
    query.words = []
    query.timeis_tags = null
    query.tags = null
    query.reps = ['rep1']
    const cloned = query.clone()
    expect(cloned.words).toEqual([])
    expect(cloned.words, '配列はコピーされ参照を共有しない').not.toBe(query.words)
    expect(cloned.timeis_tags).toBeNull()
    expect(cloned.tags).toBeNull()
    expect(cloned.reps).toEqual(['rep1'])
  })

  test('parse_find_kyou_query は旧形式(use_*入り)を正規化し、旧キーを残さない', () => {
    const legacy = {
      query_id: 'q-legacy',
      use_words: true, keywords: '写真 -除外', words: null, not_words: null,
      use_tags: false, tags: ['旅行'],
      use_reps: true, reps: null,
      use_calendar: false, calendar_start_date: '2026-07-01T00:00:00.000Z',
      use_map: false, map_latitude: 35, map_longitude: 139, map_radius: 500,
      use_timeis: false, timeis_words: ['作業'],
      use_mi_board_name: false, mi_board_name: '板',
      use_update_time: false, update_time: '2020-01-01T00:00:00.000Z',
    }
    const parsed = FindKyouQuery.parse_find_kyou_query(legacy)
    expect(parsed.words, 'use=trueで値がnullなら[]を物質化(空指定の旧挙動保存)').toEqual([])
    expect(parsed.not_words).toEqual([])
    expect(parsed.reps).toEqual([])
    expect(parsed.tags, 'use=falseならnull(未使用)').toBeNull()
    expect(parsed.calendar_start_date).toBeNull()
    expect(parsed.map_latitude).toBeNull()
    expect(parsed.timeis_words).toBeNull()
    expect(parsed.mi_board_name).toBeNull()
    const keys = Object.keys(parsed)
    expect(keys.filter((key) => key.startsWith('use_')), '旧キーがインスタンスへ混入するとdeep_equalsのキー数比較が崩れる').toEqual([])
    expect(keys).not.toContain('update_time')
  })

  test('parse_find_kyou_query は新形式JSONを恒等で復元する(localStorage往復)', () => {
    const query = new FindKyouQuery()
    query.query_id = 'q-roundtrip'
    query.keywords = '写真'
    query.words = ['写真']
    query.not_words = []
    query.tags = ['旅行']
    query.reps = null
    query.rep_types = ['mi']
    query.timeis_tags = []
    query.calendar_start_date = new Date(2026, 6, 1)
    query.calendar_end_date = new Date(2026, 6, 3, 23, 59, 59, 999)
    query.map_latitude = 35.65
    query.map_longitude = 139.74
    query.map_radius = 500
    query.plaing_time = new Date(2026, 6, 2, 12, 0)
    query.period_of_time_week_of_days = []
    query.mi_board_name = '板A'

    const restored = FindKyouQuery.parse_find_kyou_query(JSON.parse(JSON.stringify(query)))
    expect(deep_equals(restored, query), 'JSON往復で null/[]/日付が変質してはいけない').toBe(true)
  })
})
