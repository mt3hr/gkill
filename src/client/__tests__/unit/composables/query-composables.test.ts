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

import { useTagQuery } from '@/classes/use-tag-query'
import { useRepQuery } from '@/classes/use-rep-query'

// 生成テストの対象。以前は動的 import を try/catch で包んで失敗を黙ってスキップしていた
// （import が壊れても「1本でも通れば緑」）ので、静的 import に置き換えて全本を固定する
const query_composables: Array<[string, unknown]> = [
  ['useCalendarQuery', useCalendarQuery],
  ['useTagQuery', useTagQuery],
  ['useMapQuery', useMapQuery],
  ['useRepQuery', useRepQuery],
  ['usePeriodOfTimeQuery', usePeriodOfTimeQuery],
  ['useTimeIsQuery', useTimeIsQuery],
]

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
  test.each(query_composables)('%s は生成できる', (_name, factory) => {
    const result = (factory as (options: unknown) => unknown)({ props: createMockQueryProps(), emits: vi.fn() })
    expect(result).toBeDefined()
    expect(Object.keys(result as object).length).toBeGreaterThan(0)
  })
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
    query.calendar_start_time = new Date(2026, 6, 1)
    query.calendar_end_time = new Date(2026, 6, 3, 23, 59, 59, 999)
    return query
  }

  test('日付非null着信で立ち、日付レンジが構築される', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    expect(view.get_use_calendar(), '既定クエリは両方null(未使用)').toBe(false)

    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(true)
    expect(moment(view.get_start_time()).format('YYYY-MM-DD')).toBe('2026-07-01')
    expect(moment(view.get_end_time()).format('YYYY-MM-DD')).toBe('2026-07-03')
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
    expect(moment(view.get_start_time()).format('YYYY-MM-DD'), '同一query_id内の即時トグルで値が復活する').toBe('2026-07-01')

    // 別列（query_id変化）へは着信値（null=選択なし）でリセット
    const q2 = new FindKyouQuery()
    q2.query_id = 'q2'
    props.find_kyou_query = q2
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(false)
    expect(view.get_start_time()).toBeNull()
    expect(view.get_end_time()).toBeNull()
  })

  test('同じ日付の連続クリック（Vuetifyのレンジ解除）でチェックは外れない', async () => {
    const { props, view } = create_view(new FindKyouQuery())

    // 単一日選択の状態。同日なので dates は1要素で保持される
    const single_day = new FindKyouQuery()
    single_day.query_id = 'q1'
    single_day.calendar_start_time = new Date(2026, 7, 1)
    single_day.calendar_end_time = new Date(2026, 7, 1, 23, 59, 59, 999)
    props.find_kyou_query = single_day
    await flush_watchers()
    expect(view.dates.value).toHaveLength(1)
    expect(view.get_use_calendar()).toBe(true)

    // 同じ日付をもう一度クリックすると Vuetify が空配列を返す
    view.clicked_date([])
    expect(view.get_start_time()).toBeNull()
    expect(view.get_end_time()).toBeNull()
    expect(view.get_use_calendar(), '選択が空になってもカレンダー条件は有効なまま').toBe(true)

    // 列のクエリが null/null で返ってきてもチェックは外れない
    const echo = new FindKyouQuery()
    echo.query_id = 'q1'
    props.find_kyou_query = echo
    await flush_watchers()
    expect(view.get_use_calendar()).toBe(true)
    expect(view.get_start_time()).toBeNull()
  })

  test('クリアはローカルの日付選択を捨てる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    view.clicked_clear_calendar_button()
    expect(view.get_start_time(), '既定期間が未設定だと着信はnull/nullなので、ここで落とさないと古い選択が残る').toBeNull()
    expect(view.get_end_time()).toBeNull()
  })
})

describe('useCalendarQuery: 期間の開始時刻・終了時刻', () => {
  function create_view(initial: FindKyouQuery) {
    const emits = vi.fn()
    const props = reactive({
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
      inited: true,
    })
    const view = useCalendarQuery({ props: props as never, emits: emits as never })
    return { props, view, emits }
  }

  function make_dated_query(query_id: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = query_id
    query.calendar_start_time = new Date(2026, 6, 1)
    query.calendar_end_time = new Date(2026, 6, 3, 23, 59, 59, 999)
    return query
  }

  // ここが後方互換の正本。既定が変わると、保存済みの全列の検索結果が黙ってずれる
  test('時刻を指定しなければ従来どおり 00:00:00.000 / 23:59:59.999', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    expect(view.start_time_string.value, '日の境界ちょうどは未指定なので欄は空のまま').toBeNull()
    expect(view.end_time_string.value).toBeNull()
    expect(moment(view.get_start_time()).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-01 00:00:00.000')
    expect(moment(view.get_end_time()).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-03 23:59:59.999')
  })

  test('開始時刻を入れるとその時刻ちょうどになる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    view.start_time_string.value = '09:00'
    await flush_watchers()
    expect(moment(view.get_start_time()).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-01 09:00:00.000')
  })

  // VTimePicker に秒の UI が無いので、HH:mm は「その分の末尾まで」と読む。
  // 18:30:00.000 にすると、ちょうど 18:30 の記録1件しか入らない
  test('終了時刻は指定された分の末尾まで', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()

    view.end_time_string.value = '18:30'
    await flush_watchers()
    expect(moment(view.get_end_time()).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-03 18:30:59.999')
  })

  test('時刻付きで着信した列を開き直すと時刻欄が埋まる', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    const query = new FindKyouQuery()
    query.query_id = 'q1'
    query.calendar_start_time = new Date(2026, 6, 1, 9, 0, 0, 0)
    query.calendar_end_time = new Date(2026, 6, 3, 18, 30, 59, 999)
    props.find_kyou_query = query
    await flush_watchers()

    expect(view.start_time_string.value).toBe('09:00')
    expect(view.end_time_string.value).toBe('18:30')
  })

  test('時刻だけ動かしても検索をやり直す', async () => {
    const { props, view, emits } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    emits.mockClear()

    view.start_time_string.value = '09:00'
    await flush_watchers()
    const call = emits.mock.calls.find((args) => args[0] === 'request_update_dates')
    expect(call, '時刻だけの変更でも request_update_dates が出る').toBeTruthy()
    expect(moment(call?.[1]).format('HH:mm:ss.SSS')).toBe('09:00:00.000')
  })

  test('日付が選ばれていなければ時刻を動かしても何も出さない', async () => {
    const { view, emits } = create_view(new FindKyouQuery())
    await flush_watchers()
    emits.mockClear()

    view.start_time_string.value = '09:00'
    await flush_watchers()
    expect(emits.mock.calls.filter((args) => args[0] === 'request_update_dates')).toHaveLength(0)
  })

  test('props からの逆同期では emit しない（往復が終わらなくなる）', async () => {
    const { props, view, emits } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    emits.mockClear()

    const query = new FindKyouQuery()
    query.query_id = 'q2'
    query.calendar_start_time = new Date(2026, 6, 1, 9, 0, 0, 0)
    query.calendar_end_time = new Date(2026, 6, 3, 18, 30, 59, 999)
    props.find_kyou_query = query
    await flush_watchers()

    expect(view.start_time_string.value).toBe('09:00')
    expect(emits.mock.calls.filter((args) => args[0] === 'request_update_dates')).toHaveLength(0)
  })

  test('クリアは時刻欄も落とす', async () => {
    const { props, view } = create_view(new FindKyouQuery())
    props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    view.start_time_string.value = '09:00'
    view.end_time_string.value = '18:30'
    await flush_watchers()

    view.clicked_clear_calendar_button()
    expect(view.start_time_string.value).toBeNull()
    expect(view.end_time_string.value).toBeNull()
  })
})

describe('useCalendarQuery: 期間の開始日・終了日の欄', () => {
  function create_view(initial: FindKyouQuery) {
    const emits = vi.fn()
    const props = reactive({
      application_config: make_fake_application_config(),
      find_kyou_query: initial,
      inited: true,
    })
    const view = useCalendarQuery({ props: props as never, emits: emits as never })
    return { props, view, emits }
  }

  function make_dated_query(query_id: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = query_id
    query.calendar_start_time = new Date(2026, 6, 1)
    query.calendar_end_time = new Date(2026, 6, 3, 23, 59, 59, 999)
    return query
  }

  async function create_dated_view() {
    const created = create_view(new FindKyouQuery())
    created.props.find_kyou_query = make_dated_query('q1')
    await flush_watchers()
    created.emits.mockClear()
    return created
  }

  function find_update_dates(emits: ReturnType<typeof vi.fn>) {
    return emits.mock.calls.filter((args) => args[0] === 'request_update_dates')
  }

  test('カレンダーの選択が日付欄へ写る', async () => {
    const { view } = await create_dated_view()
    expect(view.start_date_string.value).toBe('2026-07-01')
    expect(view.end_date_string.value).toBe('2026-07-03')

    view.clicked_date([new Date(2026, 7, 10), new Date(2026, 7, 20)])
    expect(view.start_date_string.value, 'カレンダーをクリックしても欄が追従する').toBe('2026-08-10')
    expect(view.end_date_string.value).toBe('2026-08-20')
  })

  test('開始日を打ち直すと選択も検索も動く', async () => {
    const { view, emits } = await create_dated_view()

    view.updated_start_date('2026-06-20')

    expect(view.dates.value).toHaveLength(2)
    expect(view.start_date_string.value).toBe('2026-06-20')
    expect(view.end_date_string.value, '直していない側は動かない').toBe('2026-07-03')
    const calls = find_update_dates(emits)
    expect(calls).toHaveLength(1)
    expect(moment(calls[0][1]).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-06-20 00:00:00.000')
    expect(moment(calls[0][2]).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-03 23:59:59.999')
  })

  test('終了日を打ち直すと選択も検索も動く', async () => {
    const { view, emits } = await create_dated_view()

    view.updated_end_date('2026-07-31')

    expect(view.start_date_string.value, '直していない側は動かない').toBe('2026-07-01')
    expect(view.end_date_string.value).toBe('2026-07-31')
    const calls = find_update_dates(emits)
    expect(calls).toHaveLength(1)
    expect(moment(calls[0][2]).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-07-31 23:59:59.999')
  })

  test('打った日の月がカレンダーに出る', async () => {
    const { view } = await create_dated_view()

    view.updated_start_date('2024-03-05')

    expect(view.calendar_year.value, '月が動かないと欄とカレンダーが連動して見えない').toBe(2024)
    expect(view.calendar_month.value, 'VDatePicker の month は0始まり').toBe(2)
  })

  test('時刻を入れてあれば日付欄からの指定にも効く', async () => {
    const { view, emits } = await create_dated_view()
    view.start_time_string.value = '09:00'
    await flush_watchers()
    emits.mockClear()

    view.updated_start_date('2026-06-20')

    const calls = find_update_dates(emits)
    expect(moment(calls[0][1]).format('YYYY-MM-DD HH:mm:ss.SSS')).toBe('2026-06-20 09:00:00.000')
  })

  test('開始日が終了日より後なら、打った側の1日へ畳む', async () => {
    const { view } = await create_dated_view()

    view.updated_start_date('2026-07-10')

    expect(view.dates.value, '同じ日は1要素。2要素だとVDatePickerがレンジ完了と判断する').toHaveLength(1)
    expect(view.start_date_string.value).toBe('2026-07-10')
    expect(view.end_date_string.value, '畳んだことが欄にも出る').toBe('2026-07-10')
  })

  test('日付欄を空にすると範囲ごと落ちる', async () => {
    const { view, emits } = await create_dated_view()

    view.updated_start_date(null)

    expect(view.dates.value).toHaveLength(0)
    expect(view.get_start_time()).toBeNull()
    expect(view.get_end_time()).toBeNull()
    const calls = find_update_dates(emits)
    expect(calls).toHaveLength(1)
    expect(calls[0][1]).toBeNull()
    expect(calls[0][2]).toBeNull()
  })

  // type="date" は年を4桁0埋めで返すので、年を打っている途中は "0002-07-01" が流れてくる
  test('打ちかけの年では検索をやり直さない', async () => {
    const { view, emits } = await create_dated_view()

    view.updated_start_date('0002-07-01')

    expect(view.dates.value[0], '打ちかけで選択を動かすと、打ち終わる前に検索が何度も走る')
      .toStrictEqual(new Date(2026, 6, 1))
    expect(find_update_dates(emits)).toHaveLength(0)
  })

  test('同じ日を打ち直しても検索をやり直さない', async () => {
    const { view, emits } = await create_dated_view()

    view.updated_start_date('2026-07-01')

    expect(find_update_dates(emits)).toHaveLength(0)
  })

  test('props からの逆同期では emit しない（往復が終わらなくなる）', async () => {
    const { props, view, emits } = await create_dated_view()

    const query = new FindKyouQuery()
    query.query_id = 'q2'
    query.calendar_start_time = new Date(2026, 8, 1)
    query.calendar_end_time = new Date(2026, 8, 5, 23, 59, 59, 999)
    props.find_kyou_query = query
    await flush_watchers()

    expect(view.start_date_string.value).toBe('2026-09-01')
    expect(view.end_date_string.value).toBe('2026-09-05')
    expect(find_update_dates(emits)).toHaveLength(0)
  })

  test('クリアは日付欄も落とす', async () => {
    const { view } = await create_dated_view()

    view.clicked_clear_calendar_button()

    expect(view.start_date_string.value).toBeNull()
    expect(view.end_date_string.value).toBeNull()
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
    expect(view.week_of_days.value, 'null着信でローカルの選択は空に戻る').toEqual([])
  })

  // 時間帯にチェックを入れた直後は曜日が未選択で、そのまま [] を送ると「0件指定」になる
  // （曜日を押すまで対象なし）。画面の未選択は全曜日(=曜日制限なし)として送り、
  // 1つ押せばその曜日だけになる。「水を1回押しても出ない」の調査で見つけた
  test('曜日が未選択なら全曜日(制限なし)を送り、1つ押せばその曜日だけになる', async () => {
    const emits = vi.fn()
    const props = reactive({ application_config: make_fake_application_config(), find_kyou_query: new FindKyouQuery() })
    const view = usePeriodOfTimeQuery({ props: props as never, emits: emits as never })

    expect(view.get_period_of_time_week_of_days(), '未選択は [] ではなく全7曜日').toEqual([0, 1, 2, 3, 4, 5, 6])

    view.week_of_days.value = [3]
    await flush_watchers()
    expect(view.get_period_of_time_week_of_days(), '1つ押せばその曜日だけ').toEqual([3])
    expect(emits).toHaveBeenLastCalledWith('request_update_period_of_time', null, null, [3])

    view.week_of_days.value = []
    await flush_watchers()
    expect(view.get_period_of_time_week_of_days(), '全部外せばまた全曜日').toEqual([0, 1, 2, 3, 4, 5, 6])
    expect(emits).toHaveBeenLastCalledWith('request_update_period_of_time', null, null, [0, 1, 2, 3, 4, 5, 6])
  })

  // 時刻欄を空に戻す手段は欄の中の ×（clearable）だけ（ADR-0419）。
  // × が入れるのは "" で、null ではない ―― null を入れると
  // get_period_of_time_*_time_second の === "" 判定をすり抜け、NaN の秒が条件に乗る
  test('時刻の × で未指定（null）へ戻る', async () => {
    const emits = vi.fn()
    const props = reactive({ application_config: make_fake_application_config(), find_kyou_query: new FindKyouQuery() })
    const view = usePeriodOfTimeQuery({ props: props as never, emits: emits as never })

    view.period_of_time_start_time_string.value = '09:00'
    view.period_of_time_end_time_string.value = '18:00'
    await flush_watchers()
    expect(view.get_period_of_time_start_time_second()).not.toBeNull()
    expect(view.get_period_of_time_end_time_second()).not.toBeNull()

    view.period_of_time_start_time_string.value = ''
    view.period_of_time_end_time_string.value = ''
    await flush_watchers()
    expect(view.get_period_of_time_start_time_second(), '× で開始時刻が未指定へ戻らない').toBeNull()
    expect(view.get_period_of_time_end_time_second(), '× で終了時刻が未指定へ戻らない').toBeNull()
    expect(emits).toHaveBeenLastCalledWith('request_update_period_of_time', null, null, [0, 1, 2, 3, 4, 5, 6])
  })

  test('全7曜日が戻ってきてもローカルの未選択は保つ（次の1押しが「その曜日だけ」になる）', async () => {
    const props = reactive({ application_config: make_fake_application_config(), find_kyou_query: new FindKyouQuery() })
    const view = usePeriodOfTimeQuery({ props: props as never, emits: vi.fn() as never })

    // 時間帯チェック直後の往復: 全曜日で検索した結果が props に戻る
    const q_all = new FindKyouQuery()
    q_all.query_id = 'q1'
    q_all.period_of_time_week_of_days = [0, 1, 2, 3, 4, 5, 6]
    props.find_kyou_query = q_all
    await flush_watchers()
    expect(view.get_use_period_of_time()).toBe(true)
    expect(view.week_of_days.value, '全点灯へ書き換えない').toEqual([])
    expect(view.get_period_of_time_week_of_days()).toEqual([0, 1, 2, 3, 4, 5, 6])

    // 利用者が明示的に全部押した状態も、全曜日の往復で崩さない
    view.week_of_days.value = [6, 5, 4, 3, 2, 1, 0]
    await flush_watchers()
    const q_all2 = new FindKyouQuery()
    q_all2.query_id = 'q1'
    q_all2.period_of_time_week_of_days = [6, 5, 4, 3, 2, 1, 0]
    props.find_kyou_query = q_all2
    await flush_watchers()
    expect(view.week_of_days.value).toEqual([6, 5, 4, 3, 2, 1, 0])

    // 全曜日でない着信（保存条件の復元・別の列へのフォーカス切替）は従来どおり上書きする
    const q_some = new FindKyouQuery()
    q_some.query_id = 'q2'
    q_some.period_of_time_week_of_days = [1, 2]
    props.find_kyou_query = q_some
    await flush_watchers()
    expect(view.week_of_days.value).toEqual([1, 2])
    expect(view.get_period_of_time_week_of_days()).toEqual([1, 2])

    // 旧UIが保存した [] (0件指定) は、そのまま復元すると未選択=全曜日として次回送られる
    const q_empty = new FindKyouQuery()
    q_empty.query_id = 'q3'
    q_empty.period_of_time_week_of_days = []
    props.find_kyou_query = q_empty
    await flush_watchers()
    expect(view.week_of_days.value).toEqual([])
    expect(view.get_period_of_time_week_of_days()).toEqual([0, 1, 2, 3, 4, 5, 6])
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
    expect(query.calendar_start_time).toBeNull()
    expect(query.calendar_end_time).toBeNull()
    expect(query.playing_time).toBeNull()
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
      use_calendar: false, calendar_start_time: '2026-07-01T00:00:00.000Z',
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
    expect(parsed.calendar_start_time).toBeNull()
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
    query.calendar_start_time = new Date(2026, 6, 1)
    query.calendar_end_time = new Date(2026, 6, 3, 23, 59, 59, 999)
    query.map_latitude = 35.65
    query.map_longitude = 139.74
    query.map_radius = 500
    query.playing_time = new Date(2026, 6, 2, 12, 0)
    query.period_of_time_week_of_days = []
    query.mi_board_name = '板A'

    const restored = FindKyouQuery.parse_find_kyou_query(JSON.parse(JSON.stringify(query)))
    expect(deep_equals(restored, query), 'JSON往復で null/[]/日付が変質してはいけない').toBe(true)
  })
})
