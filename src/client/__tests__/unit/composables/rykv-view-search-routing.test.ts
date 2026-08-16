/**
 * useRykvView の「列×検索」ルーティングのテスト。
 * 期待仕様「検索時の列に検索時の結果が表示される」
 * 「同じ列で検索途中に条件が変更されたら、最後の検索条件の結果を表示する」を固定する。
 * get_kyous を deferred 化し、解決順を制御してレースを決定論的に再現する。
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
// 本番同様に gkill-api を先に評価させる(mi-re-kyou-view.test.ts と同じ事情)
import '@/classes/api/gkill-api'
import { useRykvView } from '@/classes/use-rykv-view'
import { refresh_kyou_in_list } from '@/classes/kyou-reload'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { RykvViewProps } from '@/pages/views/rykv-view-props'
import type { RykvViewEmits } from '@/pages/views/rykv-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import {
  createColumnViewMockApi,
  makeColumnQuery,
  makeColumnViewProps,
  finish_application_config_load,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => { }) as unknown as RykvViewEmits

function kyous(ids: string[]): Kyou[] {
  return ids.map((id) => ({ id })) as unknown as Kyou[]
}

function kyouIds(list: unknown): string[] {
  return (list as Array<{ id: string }>).map((kyou) => kyou.id)
}

function createView(config_overrides: Record<string, unknown> = {}) {
  const { api, pending_get_kyous } = createColumnViewMockApi()
  const raw_props = makeColumnViewProps(api, config_overrides, {
    is_shared_rykv_view: false,
    share_title: '',
  })
  const props = raw_props as unknown as RykvViewProps
  const view = useRykvView({ props, emits: noop_emits })
  // init() は application_config.is_loaded の watch で起動する
  const start_init = () => finish_application_config_load(raw_props)
  return { api, pending_get_kyous, view, start_init }
}

describe('useRykvView 列×検索ルーティング', () => {
  test('別列の検索は自列にだけ書き、focused_queryを乗っ取らない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
    const list_a = kyous(['a1'])
    setupColumns(view, [query_a, query_b], [list_a, []])

    view.reload_list(1)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)

    pending_get_kyous[0].resolve({ kyous: kyous(['b1', 'b2']), messages: [], errors: [] })
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['a1'])
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual(['b1', 'b2'])
    // フォーカスは列0のまま、サイドバーの表示対象も列0のまま
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

    // 新しい検索(2発目)が先に完了する
    pending_get_kyous[1].resolve({ kyous: kyous(['new']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new'])

    // 古い検索(1発目)が遅れて完了しても、結果は捨てられ上書きされない
    pending_get_kyous[0].resolve({ kyous: kyous(['old']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new'])
    // スピナーは最後の検索の完了で消えたまま(古い応答が触らない)
    expect(fakes.get('col-a')!.get_is_loading()).toBe(false)
  })

  test('検索中に列を閉じると、遅れて届いた結果はどこにも書かれない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
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

  test('サイドバー編集はquery_idの由来列に書き戻される(フォーカスがずれていても)', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
    setupColumns(view, [query_a, query_b], [[], []])

    // フォーカスは列0のまま、列1のquery_idを持つ編集が届く
    // (全列リロード等でサイドバーが列1の条件を表示していた状況)
    const edited = makeColumnQuery('col-b')
    edited.keywords = 'edited'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()

    expect(view.querys.value[0].query_id).toBe('col-a')
    expect(view.querys.value[0].keywords).toBe('')
    expect(view.querys.value[1].query_id).toBe('col-b')
    expect(view.querys.value[1].keywords).toBe('edited')
    // query_idが重複しない(重複すると結果の誤配送とVueのkey衝突が起きる)
    const ids = view.querys.value.map((query) => query.query_id)
    expect(new Set(ids).size).toBe(ids.length)

    pending_get_kyous[0].resolve({ kyous: kyous(['b-edited']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual(['b-edited'])
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
    const query_b = makeColumnQuery('col-b')
    setupColumns(view, [query_a, query_b], [[], []])

    view.onColumnClickedListView(1)
    // 同tickの機械的なupdated_queryは無視される
    view.onSidebarUpdatedQuery(makeColumnQuery('col-b'))
    expect(pending_get_kyous.length).toBe(0)

    await flushAsync()

    // 次tick以降のユーザ編集は1回だけ検索になる
    const edited = makeColumnQuery('col-b')
    edited.keywords = 'user-edit'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.keywords).toBe('user-edit')
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
    // query_idは列のものを引き継ぐ(列のidentityは変えない)
    expect(view.querys.value[0].query_id).toBe('col-a')
  })

  test('列を閉じたら最近傍の列にフォーカスが移る', async () => {
    // 別の列を閉じてもフォーカス中の列を追い続ける
    {
      const { view } = createView()
      const query_a = makeColumnQuery('col-a')
      const query_b = makeColumnQuery('col-b')
      const query_c = makeColumnQuery('col-c')
      setupColumns(view, [query_a, query_b, query_c], [[], [], []])
      view.focused_column_index.value = 2
      view.focused_query.value = query_c

      await view.close_list_view(0)
      await flushAsync()

      expect(view.focused_column_index.value).toBe(1)
      expect(view.focused_query.value.query_id).toBe('col-c')
    }
    // フォーカス中の列自身を閉じたら右隣(同じ位置に来た列)へ
    {
      const { view } = createView()
      const query_a = makeColumnQuery('col-a2')
      const query_b = makeColumnQuery('col-b2')
      const query_c = makeColumnQuery('col-c2')
      setupColumns(view, [query_a, query_b, query_c], [[], [], []])
      view.focused_column_index.value = 1
      view.focused_query.value = query_b

      await view.close_list_view(1)
      await flushAsync()

      expect(view.focused_column_index.value).toBe(1)
      expect(view.focused_query.value.query_id).toBe('col-c2')
    }
  })

  test('reload_kyouは列の再検索と交錯しても新しい結果を古いリストで潰さない', async () => {
    const { view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
    const list_a = kyous(['a1'])
    const list_b = kyous(['b1'])
    setupColumns(view, [query_a, query_b], [list_a, list_b])

    type RefreshOptions = { replace: (next_list: unknown[]) => void }
    const pending_refreshes: Array<{ options: RefreshOptions; resolve: () => void }> = []
    vi.mocked(refresh_kyou_in_list).mockImplementation(((
      _list: unknown[],
      _kyou: unknown,
      options: RefreshOptions,
    ) =>
      new Promise<void>((resolve) => {
        pending_refreshes.push({ options, resolve })
      })) as never)

    view.reload_kyou({ id: 'a1' } as unknown as Kyou)
    await flushAsync()
    expect(pending_refreshes.length).toBe(1)

    // 列0の検索が完了してリストが差し替わった状況
    view.match_kyous_list.value[0] = kyous(['fresh'])

    // 遅れて届いたreload結果(検索前のスナップショット由来のa1)は現在のリストを潰さない
    pending_refreshes[0].options.replace(kyous(['a1']))
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['fresh'])
    pending_refreshes[0].resolve()
    await flushAsync()

    // a1を含まない列1のrefreshも完了させる(実実装ではno-opになる)
    expect(pending_refreshes.length).toBe(2)
    pending_refreshes[1].resolve()
    await flushAsync()

    // 交錯しなければ、該当行だけが新しいインスタンスに差し替わる
    view.reload_kyou({ id: 'b1' } as unknown as Kyou)
    await flushAsync()
    expect(pending_refreshes.length).toBe(3)
    pending_refreshes[2].resolve()
    await flushAsync()
    expect(pending_refreshes.length).toBe(4)
    const refreshed_b1 = { id: 'b1', refreshed: true }
    pending_refreshes[3].options.replace([refreshed_b1])
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual(['b1'])
    expect((view.match_kyous_list.value[1][0] as { refreshed?: boolean }).refreshed).toBe(true)
    pending_refreshes[3].resolve()
  })

  test('フォーカス列以外の検索完了はカレンダー(focused_kyous_list)を差し替えない', async () => {
    const { pending_get_kyous, view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
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

  test('init(hot reload OFF)後、focused_queryが列0の保存済み条件に同期される', async () => {
    // 同期しないと検索ボタンがサイドバーの既定値から条件を組み、保存済み条件を上書きする
    const { api, view, start_init } = createView({ rykv_hot_reload: false })
    const saved = makeColumnQuery('saved-col')
    saved.keywords = 'saved-keyword'
    api.get_saved_rykv_find_kyou_querys.mockReturnValue([saved])
    view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }

    start_init()
    await flushAsync()

    expect(view.inited.value).toBe(true)
    expect(view.focused_query.value.query_id).toBe('saved-col')
    expect(view.focused_query.value.keywords).toBe('saved-keyword')
  })

  test('reload_kyou中に列を閉じても、残った列に別列の内容が書き込まれない', async () => {
    const { view } = createView()
    const query_a = makeColumnQuery('col-a')
    const query_b = makeColumnQuery('col-b')
    const list_b = kyous(['b1'])
    setupColumns(view, [query_a, query_b], [kyous(['a1']), list_b])

    type RefreshOptions = { replace: (next_list: unknown[]) => void }
    const pending_refreshes: Array<{ options: RefreshOptions; resolve: () => void }> = []
    vi.mocked(refresh_kyou_in_list).mockImplementation(((
      _list: unknown[],
      _kyou: unknown,
      options: RefreshOptions,
    ) =>
      new Promise<void>((resolve) => {
        pending_refreshes.push({ options, resolve })
      })) as never)

    view.reload_kyou({ id: 'a1' } as unknown as Kyou)
    await flushAsync()
    expect(pending_refreshes.length).toBe(1)

    // 列0(col-a)がreload中に閉じられ、indexがずれた状況
    await view.close_list_view(0)
    await flushAsync()

    pending_refreshes[0].options.replace(kyous(['a1']))
    expect(view.match_kyous_list.value.length).toBe(1)
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['b1'])
    pending_refreshes[0].resolve()
  })

  test('フォーカス切替のflush中に届く機械的updated_queryは検索にならない(抑止のnextTick登録順回帰)', async () => {
    // 実物のサイドバーは focused_query の props 同期(=ウォッチャflush中)から
    // 機械的な updated_query を発火する。同期呼び出しでの再現はこのタイミングを
    // 通らないため、以前のテストは「抑止解除がflushより先に走る」回帰を見逃した。
    // (抑止はリアクティブ書き込みでflushが予約された後に解除を登録しないと、
    //  マイクロタスクFIFOで解除が先行し一度も効かない)
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
