/**
 * useRykvView の地図の日付ピッカー（requested_change_map_date の受け口）の検証。
 *
 * 地図の日付表示をクリックして開くピッカーで選んだ日は、ライフログビューでは
 * 「地図に映す日」だけを切り替える。検索条件と一覧は動かさない
 * （ダッシュボードが表示日ごと移すのと対照的。dashboard-view-date-navigation.test.ts）。
 *
 * 受け口 onGpsLogMapRequestedChangeDate は地図の 3 つの ref だけを書く。同じ地図から出る
 * requested_focus_time の受け口 onGpsLogMapRequestedFocusTime と取り違えて focused_time を書くと、
 * 型検査は通ったまま「地図で別の日を選んだだけでフォーカス列がその時刻へスクロールし、
 * 地図に映す日は変わらない」がエラーも警告も出ずに起きる。2つの受け口の違いをここで固定する。
 *
 * focused_time はコンポーザブルの外へ出ていないので、その watch がフォーカス列へ撃つ
 * scroll_to_time（引数が focused_time）を観測点にする。
 */
import { describe, test, expect, vi } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/router', () => ({ default: { replace: vi.fn(), push: vi.fn() } }))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
  reset_dialog_history: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-scoped-enter-for-kftl', () => ({ useScopedEnterForKFTL: vi.fn() }))
vi.mock('@/classes/use-scoped-ctrl-v-for-clipboard', () => ({ useScopedCtrlVForClipboard: vi.fn() }))
vi.mock('@/classes/kyou-reload', () => ({
  build_mi_reload_query: vi.fn((query: unknown) => query),
  new_reload_batch: vi.fn(() => 0),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useRykvView } from '@/classes/use-rykv-view'
import type { RykvViewProps } from '@/pages/views/rykv-view-props'
import type { RykvViewEmits } from '@/pages/views/rykv-view-emits'
import {
  createColumnViewMockApi,
  makeColumnQuery,
  makeColumnViewProps,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => { }) as unknown as RykvViewEmits

function createView() {
  const { api, pending_get_kyous } = createColumnViewMockApi()
  const raw_props = makeColumnViewProps(api, {}, {
    is_shared_rykv_view: false,
    share_title: '',
  })
  const props = raw_props as unknown as RykvViewProps
  const view = useRykvView({ props, emits: noop_emits })
  return { api, pending_get_kyous, view }
}

// 列を1本組み、地図の3つの ref を既知の値に揃えてから試験する。
// setupColumns は inited=true にするので、focused_time の watch はフォーカス列の scroll_to_time へ届く
async function createViewWithColumn() {
  const { pending_get_kyous, view } = createView()
  const query_a = makeColumnQuery('col-a')
  query_a.calendar_start_date = new Date(2026, 8, 1)
  query_a.calendar_end_date = new Date(2026, 8, 30)
  const fakes = setupColumns(view, [query_a], [[]])

  const initial_map_time = new Date(2026, 8, 20, 12, 0, 0)
  view.gps_log_map_start_time.value = initial_map_time
  view.gps_log_map_end_time.value = initial_map_time
  view.gps_log_map_marker_time.value = initial_map_time
  await flushAsync()
  fakes.get('col-a')!.scroll_to_time.mockClear()
  const searches_before = pending_get_kyous.length

  return { pending_get_kyous, view, fakes, query_a, initial_map_time, searches_before }
}

describe('useRykvView 地図の日付ピッカー（onGpsLogMapRequestedChangeDate）', () => {
  test('地図に映す日（開始・終了・マーカー）をその日へ移す', async () => {
    const { view } = await createViewWithColumn()
    const picked = new Date(2016, 4, 15)

    view.onGpsLogMapRequestedChangeDate(picked)
    await flushAsync()

    expect(view.gps_log_map_start_time.value.getTime(), '地図の開始日が移っていない').toBe(picked.getTime())
    expect(view.gps_log_map_end_time.value.getTime(), '地図の終了日が移っていない').toBe(picked.getTime())
    expect(view.gps_log_map_marker_time.value.getTime(), '地図のマーカー時刻が移っていない').toBe(picked.getTime())
  })

  test('focused_time は触らず、フォーカス列をスクロールさせない', async () => {
    // onGpsLogMapRequestedFocusTime と取り違えて focused_time を書くと、
    // その watch がフォーカス列へ scroll_to_time(選んだ日) を撃つので、ここが赤くなる
    const { view, fakes } = await createViewWithColumn()

    view.onGpsLogMapRequestedChangeDate(new Date(2016, 4, 15))
    await flushAsync()

    expect(fakes.get('col-a')!.scroll_to_time, '地図の日を変えただけで focused_time が動き、フォーカス列がスクロールした').not.toHaveBeenCalled()
  })

  test('検索条件と一覧は動かさない（検索を飛ばさない）', async () => {
    const { pending_get_kyous, view, query_a, searches_before } = await createViewWithColumn()
    const start_before = query_a.calendar_start_date!.getTime()
    const end_before = query_a.calendar_end_date!.getTime()

    view.onGpsLogMapRequestedChangeDate(new Date(2016, 4, 15))
    await flushAsync()

    expect(view.querys.value[0].calendar_start_date!.getTime(), '列の検索期間（開始）が動いた').toBe(start_before)
    expect(view.querys.value[0].calendar_end_date!.getTime(), '列の検索期間（終了）が動いた').toBe(end_before)
    expect(pending_get_kyous.length, '地図の日を変えただけで検索が飛んだ').toBe(searches_before)
  })

  test('対照: requested_focus_time の受け口は focused_time を移して列をスクロールし、地図の日は動かさない', async () => {
    // 2つの受け口が同じ振る舞いなら、上の3本は取り違えを見つけられない。違いをここで固定する
    const { view, fakes, initial_map_time } = await createViewWithColumn()
    const time = new Date(2026, 8, 20, 15, 45, 0)

    view.onGpsLogMapRequestedFocusTime(time)
    await flushAsync()

    expect(fakes.get('col-a')!.scroll_to_time, 'focused_time が動いていない').toHaveBeenCalledTimes(1)
    expect((fakes.get('col-a')!.scroll_to_time.mock.calls[0][0] as Date).getTime()).toBe(time.getTime())
    expect(view.gps_log_map_start_time.value.getTime(), '地図のスライダー操作で地図の開始日が動いた').toBe(initial_map_time.getTime())
    expect(view.gps_log_map_end_time.value.getTime()).toBe(initial_map_time.getTime())
    expect(view.gps_log_map_marker_time.value.getTime()).toBe(initial_map_time.getTime())
  })
})

// import.meta.url は vitest の変換後は file スキームにならないので使えない。
// package.json を目印に上へ辿ってリポジトリルートを決める（column-view-init-source-scan.test.ts と同じ）
function find_repo_root(): string {
  let dir = process.cwd()
  for (let i = 0; i < 10; i++) {
    if (existsSync(join(dir, 'package.json')) && existsSync(join(dir, 'src', 'client'))) {
      return dir
    }
    const parent = dirname(dir)
    if (parent === dir) {
      break
    }
    dir = parent
  }
  throw new Error(`リポジトリルートが見つからない: cwd=${process.cwd()}`)
}

describe('rykv-view.vue の地図の配線', () => {
  // テンプレートの配線を取り違えても型検査は通る（どちらの受け口も Date を1つ取る）。
  // 2つのイベントがそれぞれ自分の受け口へ届いていることをソースで固定する
  function read_gps_log_map_tag(): string {
    const source = readFileSync(join(find_repo_root(), 'src/client/pages/views/rykv-view.vue'), 'utf8')
    const start = source.indexOf('<GPSLogMap')
    expect(start, 'GPSLogMap が見つからない').toBeGreaterThan(-1)
    const end = source.indexOf('/>', start)
    expect(end, 'GPSLogMap の閉じが見つからない').toBeGreaterThan(start)
    return source.slice(start, end)
  }

  test('requested_change_map_date は onGpsLogMapRequestedChangeDate へ配線されている', () => {
    const tag = read_gps_log_map_tag()
    const attr = /@requested_change_map_date="([^"]*)"/.exec(tag)
    expect(attr, '地図の日付ピッカーのイベントが受けられていない').not.toBeNull()
    expect(attr![1]).toMatch(/\bonGpsLogMapRequestedChangeDate\(/)
    expect(attr![1], '日付ピッカーがフォーカス時刻の受け口へ配線されている').not.toMatch(/\bonGpsLogMapRequestedFocusTime\b/)
  })

  test('requested_focus_time は onGpsLogMapRequestedFocusTime へ配線されている', () => {
    const tag = read_gps_log_map_tag()
    const attr = /@requested_focus_time="([^"]*)"/.exec(tag)
    expect(attr, '地図のスライダーのイベントが受けられていない').not.toBeNull()
    expect(attr![1]).toMatch(/\bonGpsLogMapRequestedFocusTime\(/)
    expect(attr![1], 'スライダーが地図の日付変更の受け口へ配線されている').not.toMatch(/\bonGpsLogMapRequestedChangeDate\b/)
  })
})
