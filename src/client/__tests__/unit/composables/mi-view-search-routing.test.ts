/**
 * useMiView の「列(板)×検索」ルーティングのテスト。
 * rykv-view-search-routing.test.ts と対になる(実装がコピー由来の対称実装のため、
 * 修正が片側にしか入っていないとここで落ちる)。
 * mi固有: カレンダー(focused_kyous_list)の汚染防止と open_or_focus_board の一致判定。
 */
import { describe, test, expect, vi } from 'vitest'
import { watch } from 'vue'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/router', () => ({ default: { replace: vi.fn(), push: vi.fn() } }))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_tags_cache: vi.fn().mockResolvedValue(undefined),
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
// 本番同様に gkill-api を先に評価させる(mi-re-kyou-view.test.ts と同じ事情)
import '@/classes/api/gkill-api'
import { useMiView } from '@/classes/use-mi-view'
import type { MiViewProps } from '@/pages/views/mi-view-props'
import type { MiViewEmits } from '@/pages/views/mi-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import {
  createColumnViewMockApi,
  makeColumnQuery,
  makeViewApplicationConfig,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => { }) as unknown as MiViewEmits

function kyous(ids: string[]): Kyou[] {
  return ids.map((id) => ({ id })) as unknown as Kyou[]
}

function kyouIds(list: unknown): string[] {
  return (list as Array<{ id: string }>).map((kyou) => kyou.id)
}

function makeBoardColumnQuery(query_id: string, board_name: string): FindKyouQuery {
  const query = makeColumnQuery(query_id)
  // 非null=板絞り込みあり（null=「すべて」。use_mi_board_name フラグは全廃済み）
  query.mi_board_name = board_name
  return query
}

function createView(config_overrides: Record<string, unknown> = {}) {
  const { api, pending_get_kyous } = createColumnViewMockApi()
  const props = {
    gkill_api: api,
    application_config: makeViewApplicationConfig(config_overrides),
    app_title_bar_height: 50,
    app_content_height: 600,
    app_content_width: 800,
  } as unknown as MiViewProps
  const view = useMiView({ props, emits: noop_emits })
  return { api, pending_get_kyous, view }
}

describe('useMiView 列(板)×検索ルーティング', () => {
  test('別列の検索は自列にだけ書き、focused_queryを乗っ取らない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    const list_a = kyous(['a1'])
    setupColumns(view, [query_a, query_b], [list_a, []])

    view.reload_list(1)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)

    pending_get_kyous[0].resolve({ kyous: kyous(['b1', 'b2']), messages: [], errors: [] })
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['a1'])
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual(['b1', 'b2'])
    expect(view.focused_column_index.value).toBe(0)
    expect(view.focused_query.value.query_id).toBe('col-a')
  })

  test('同じ列の連続検索は、応答が逆順に届いても最後の検索が勝つ', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const fakes = setupColumns(view, [query_a], [[]])

    view.reload_list(0)
    view.reload_list(0)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(2)

    pending_get_kyous[1].resolve({ kyous: kyous(['new']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new'])

    pending_get_kyous[0].resolve({ kyous: kyous(['old']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new'])
    expect(fakes.get('col-a')!.get_is_loading()).toBe(false)
  })

  test('検索中に列を閉じると、遅れて届いた結果はどこにも書かれない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    const list_b = kyous(['b1'])
    setupColumns(view, [query_a, query_b], [kyous(['a1']), list_b])

    view.reload_list(0)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)

    await view.close_list_view(0)
    await flushAsync()

    pending_get_kyous[0].resolve({ kyous: kyous(['late']), messages: [], errors: [] })
    await flushAsync()

    expect(view.querys.value.length).toBe(1)
    expect(view.querys.value[0].query_id).toBe('col-b')
    expect(view.match_kyous_list.value.length).toBe(1)
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['b1'])
    expect(view.focused_query.value.query_id).toBe('col-b')
  })

  test('サイドバー編集はquery_idの由来列に書き戻され、query_idが重複しない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    setupColumns(view, [query_a, query_b], [[], []])

    // フォーカスは列0のまま、列1のquery_idを持つ編集が届く
    const edited = makeBoardColumnQuery('col-b', 'board-b')
    edited.keywords = 'edited'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()

    expect(view.querys.value[0].query_id).toBe('col-a')
    expect(view.querys.value[0].keywords).toBe('')
    expect(view.querys.value[1].query_id).toBe('col-b')
    expect(view.querys.value[1].keywords).toBe('edited')
    const ids = view.querys.value.map((query) => query.query_id)
    expect(new Set(ids).size).toBe(ids.length)

    pending_get_kyous[0].resolve({ kyous: kyous(['b-edited']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual(['b-edited'])
    // 列0の板名表示の元になるクエリが列1の条件で汚染されていない（null=「すべて」のまま）
    expect(view.querys.value[0].mi_board_name).toBeNull()
  })

  test('存在しない列宛てのサイドバー編集は捨てられる', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    setupColumns(view, [query_a], [[]])

    const stray = makeColumnQuery('closed-column')
    view.onSidebarUpdatedQuery(stray)
    await flushAsync()

    expect(pending_get_kyous.length).toBe(0)
    expect(view.querys.value.map((query) => query.query_id)).toEqual(['col-a'])
  })

  test('フォーカス切り替え直後の機械的なサイドバー更新は1回だけ無視され、次の編集は通る', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    setupColumns(view, [query_a, query_b], [[], []])

    view.onColumnClickedListView(1)
    view.onSidebarUpdatedQuery(makeBoardColumnQuery('col-b', 'board-b'))
    expect(pending_get_kyous.length).toBe(0)

    await flushAsync()

    const edited = makeBoardColumnQuery('col-b', 'board-b')
    edited.keywords = 'user-edit'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.keywords).toBe('user-edit')
  })

  test('init(hot reload OFF)後、focused_queryが列0の保存済み条件に同期される', async () => {
    // 同期しないと検索ボタンがサイドバーの既定値から条件を組み、保存済み条件を上書きする
    // (miでは板絞り込みの列が「すべて」列へ化ける)
    const { api, view } = createView({ rykv_hot_reload: false })
    const saved = makeBoardColumnQuery('saved-col', 'saved-board')
    api.get_saved_mi_find_kyou_querys.mockReturnValue([saved])
    view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }

    view.onSidebarInited()
    await flushAsync()

    expect(view.inited.value).toBe(true)
    expect(view.focused_query.value.query_id).toBe('saved-col')
    expect(view.focused_query.value.mi_board_name).toBe('saved-board')
  })

  test('フォーカス列以外の検索完了はカレンダー(focused_kyous_list)を差し替えない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    const list_a = kyous(['a1'])
    setupColumns(view, [query_a, query_b], [list_a, []])
    view.is_show_kyou_count_calendar.value = true
    view.focused_kyous_list.value = list_a

    view.reload_list(1)
    await flushAsync()
    pending_get_kyous[0].resolve({ kyous: kyous(['b1']), messages: [], errors: [] })
    await flushAsync()

    expect(kyouIds(view.focused_kyous_list.value)).toEqual(['a1'])
  })

  test('検索ボタンはサイドバーの現在値で検索する(hot reload OFF)', async () => {
    const { pending_get_kyous, view } = createView({ rykv_hot_reload: false })
    const query_a = makeColumnQuery('col-a')
    setupColumns(view, [query_a], [[]])

    const generate_query = vi.fn((query_id: string) => {
      const query = makeColumnQuery(query_id)
      query.keywords = 'edited-in-sidebar'
      return query
    })
    view.query_editor_sidebar.value = { generate_query }

    view.onSidebarRequestedSearch(false)
    await flushAsync()

    expect(generate_query).toHaveBeenCalledWith('col-a')
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.keywords).toBe('edited-in-sidebar')
    expect(view.querys.value[0].query_id).toBe('col-a')
  })

  test('列を閉じたら最近傍の列にフォーカスが移る', async () => {
    const { view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeBoardColumnQuery('col-b', 'board-b')
    const query_c = makeBoardColumnQuery('col-c', 'board-c')
    setupColumns(view, [query_a, query_b, query_c], [[], [], []])
    view.focused_column_index.value = 2
    view.focused_query.value = query_c

    await view.close_list_view(0)
    await flushAsync()

    expect(view.focused_column_index.value).toBe(1)
    expect(view.focused_query.value.query_id).toBe('col-c')
  })

  test('open_or_focus_board: 板列は mi_board_name の null 判定で一致する', async () => {
    const { view } = createView()
    // 「すべて」列は mi_board_name === null で表す（板名の文字列一致だけだと
    // null(すべて)列を拾えないので、null判定と板名一致の2段で判定される）
    const query_all = makeColumnQuery('col-all')
    const query_board = makeBoardColumnQuery('col-x', 'board-x')
    const list_x = kyous(['x1'])
    setupColumns(view, [query_all, query_board], [[], list_x])

    view.open_or_focus_board('board-x')
    expect(view.focused_column_index.value).toBe(1)
    // カレンダーへ渡すリストは累積push型ではなく参照差し替え
    expect(kyouIds(view.focused_kyous_list.value)).toEqual(['x1'])

    // 2回開いても累積しない
    view.open_or_focus_board('board-x')
    expect(kyouIds(view.focused_kyous_list.value)).toEqual(['x1'])

    // 「すべて」(i18n mockはキーをそのまま返す)は板絞り込みのない列へ
    view.open_or_focus_board('MI_ALL_BOARD_NAME_TITLE')
    expect(view.focused_column_index.value).toBe(0)
  })

  test('open_or_focus_board: 未知の板は新しい列として末尾に追加される', async () => {
    const { view } = createView({ rykv_hot_reload: false })
    const query_a = makeColumnQuery('col-a')
    setupColumns(view, [query_a], [[]])
    view.query_editor_sidebar.value = {
      get_default_query: () => new FindKyouQuery(),
    }

    view.open_or_focus_board('board-new')
    await flushAsync()

    expect(view.querys.value.length).toBe(2)
    const added = view.querys.value[1]
    expect(added.mi_board_name, '新列は板絞り込みあり(非null)').toBe('board-new')
    // 新列のquery_idは新規採番され、既存列と重複しない
    const ids = view.querys.value.map((query) => query.query_id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(view.focused_column_index.value).toBe(1)
  })

  test('フォーカス切替のflush中に届く機械的updated_queryは検索にならない(抑止のnextTick登録順回帰)', async () => {
    // rykv-view-search-routing.test.ts の同名テストと対。実物のサイドバーは
    // focused_query の props 同期(=ウォッチャflush中)から機械的な updated_query を
    // 発火するため、同期呼び出しでは「抑止解除がflushより先に走る」回帰を検出できない
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
    setupColumns(view, [query_a, query_b], [[], []])

    const stop = watch(() => view.focused_query.value, (new_query) => {
      // 最悪ケース(残響がドリフトしていて deep_equals の安全網も破れている)を模す
      const echo = makeColumnQuery(new_query.query_id)
      echo.keywords = 'mechanical-echo-drift'
      view.onSidebarUpdatedQuery(echo)
    })

    view.onColumnClickedListView(1)
    await flushAsync()
    expect(pending_get_kyous.length, '列クリックの残響が検索になってはいけない').toBe(0)

    // 次のユーザ編集は1回だけ検索になり、search()内のfocused_query差し替えの残響も検索にならない
    const edited = makeColumnQuery('col-b')
    edited.keywords = 'user-edit'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.keywords).toBe('user-edit')
    stop()
  })
})
