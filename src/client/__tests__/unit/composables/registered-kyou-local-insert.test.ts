/**
 * 追加されたKyouを、列を再検索せずに差し込む経路のテスト。
 *
 * rykv と mi はコピー由来の対称実装なので、同じ表を両方に流す。
 * 片方だけ直すと「rykvでは差し込まれるがmiでは再検索のまま」のようなズレが残る。
 * 列の同一性は query_id で、await をまたいで index を持たないことが不変条件。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'

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
  new_reload_batch: vi.fn(() => 12345),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useRykvView } from '@/classes/use-rykv-view'
import { useMiView } from '@/classes/use-mi-view'
import { refresh_kyou } from '@/classes/kyou-reload'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import {
  createColumnViewMockApi,
  makeViewApplicationConfig,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const refresh_kyou_mock = vi.mocked(refresh_kyou)

/** 差し込み対象になれる、絞り込みを掛けていない列の条件 */
function make_open_query(query_id: string): FindKyouQuery {
  const query = new FindKyouQuery()
  query.query_id = query_id
  // 既定は tags/reps とも [] = 「有効かつ0件」なので、絞り込み無しにするには null にする
  query.tags = null
  query.reps = null
  return query
}

interface FakeKyouOptions {
  id?: string
  related_time?: Date
  rep_name?: string
  is_deleted?: boolean
}

function make_kyou(options?: FakeKyouOptions): Kyou {
  const related_time = options?.related_time ?? new Date('2026-08-02T00:00:00.000Z')
  const kyou = {
    id: options?.id ?? 'new-1',
    data_type: 'kmemo',
    rep_name: options?.rep_name ?? 'rep_a',
    related_time: related_time,
    create_time: related_time,
    update_time: related_time,
    is_deleted: options?.is_deleted ?? false,
    attached_tags: [],
    typed_mi: null,
    typed_mirekyou: null,
    clone: () => make_kyou(options),
  }
  return kyou as unknown as Kyou
}

/** 列に最初から入っている行。related_time 降順で並べておく */
function make_row(id: string, iso: string): Kyou {
  return make_kyou({ id: id, related_time: new Date(iso) })
}

function kyouIds(list: unknown): string[] {
  return (list as Array<{ id: string }>).map((kyou) => kyou.id)
}

interface ViewCase {
  name: string
  create: () => {
    api: ReturnType<typeof createColumnViewMockApi>['api']
    pending_get_kyous: ReturnType<typeof createColumnViewMockApi>['pending_get_kyous']
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    view: any
  }
}

const view_cases: ViewCase[] = [
  {
    name: 'useRykvView',
    create: () => {
      const { api, pending_get_kyous } = createColumnViewMockApi()
      const props = {
        gkill_api: api,
        application_config: makeViewApplicationConfig(),
        app_title_bar_height: 50,
        app_content_height: 600,
        app_content_width: 800,
        is_shared_rykv_view: false,
        share_title: '',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return { api, pending_get_kyous, view: useRykvView({ props, emits: (() => { }) as any }) }
    },
  },
  {
    name: 'useMiView',
    create: () => {
      const { api, pending_get_kyous } = createColumnViewMockApi()
      const props = {
        gkill_api: api,
        application_config: makeViewApplicationConfig(),
        app_title_bar_height: 50,
        app_content_height: 600,
        app_content_width: 800,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return { api, pending_get_kyous, view: useMiView({ props, emits: (() => { }) as any }) }
    },
  },
]

describe.each(view_cases)('$name 追加Kyouのローカル挿入', ({ create }) => {
  beforeEach(() => {
    refresh_kyou_mock.mockReset()
    refresh_kyou_mock.mockResolvedValue(null)
  })

  test('一致する列へ並び順を保って差し込み、検索は1回も投げない', async () => {
    const { pending_get_kyous, view } = create()
    const query_a = make_open_query('col-a')
    const list_a = [make_row('old-新', '2026-08-03T00:00:00.000Z'), make_row('old-古', '2026-08-01T00:00:00.000Z')]
    setupColumns(view, [query_a], [list_a])

    const added = make_kyou({ id: 'new-1', related_time: new Date('2026-08-02T00:00:00.000Z') })
    refresh_kyou_mock.mockResolvedValue(added)

    await view.insert_registered_kyou(added)
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['old-新', 'new-1', 'old-古'])
    expect(pending_get_kyous.length).toBe(0)
  })

  test('一致しない列は触らない', async () => {
    const { pending_get_kyous, view } = create()
    const query_a = make_open_query('col-a')
    // rep_b だけを見る列。追加された記録は rep_a なので入らない
    const query_b = make_open_query('col-b')
    query_b.reps = ['rep_b']
    const list_a: Kyou[] = []
    const list_b: Kyou[] = []
    setupColumns(view, [query_a, query_b], [list_a, list_b])

    const added = make_kyou({ id: 'new-1', rep_name: 'rep_a' })
    refresh_kyou_mock.mockResolvedValue(added)

    await view.insert_registered_kyou(added)
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new-1'])
    expect(kyouIds(view.match_kyous_list.value[1])).toEqual([])
    expect(pending_get_kyous.length).toBe(0)
  })

  test('判定できない条件の列だけ再検索し、他の列は差し込みで済ませる', async () => {
    const { pending_get_kyous, view } = create()
    const query_a = make_open_query('col-a')
    const query_b = make_open_query('col-b')
    // 本文検索はクライアントで判定できない
    query_b.words = ['会議']
    setupColumns(view, [query_a, query_b], [[], []])

    const added = make_kyou({ id: 'new-1' })
    refresh_kyou_mock.mockResolvedValue(added)

    await view.insert_registered_kyou(added)
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new-1'])
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.query_id).toBe('col-b')
  })

  test('引き直せなかった列は従来どおり再検索へ落とす', async () => {
    const { pending_get_kyous, view } = create()
    setupColumns(view, [make_open_query('col-a')], [[]])
    refresh_kyou_mock.mockResolvedValue(null)

    await view.insert_registered_kyou(make_kyou())
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual([])
    expect(pending_get_kyous.length).toBe(1)
    expect(pending_get_kyous[0].req.query.query_id).toBe('col-a')
  })

  test('削除済みで返ってきたら差し込みも再検索もしない', async () => {
    const { pending_get_kyous, view } = create()
    setupColumns(view, [make_open_query('col-a')], [[]])
    refresh_kyou_mock.mockResolvedValue(make_kyou({ is_deleted: true }))

    await view.insert_registered_kyou(make_kyou())
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual([])
    expect(pending_get_kyous.length).toBe(0)
  })

  test('二重に発火しても1行しか増えない', async () => {
    const { view } = create()
    setupColumns(view, [make_open_query('col-a')], [[]])
    const added = make_kyou({ id: 'new-1' })
    refresh_kyou_mock.mockResolvedValue(added)

    await view.insert_registered_kyou(added)
    await view.insert_registered_kyou(added)
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new-1'])
  })

  test('引き直しの最中に列を閉じても、残った列を壊さない', async () => {
    const { view } = create()
    const query_a = make_open_query('col-a')
    const query_b = make_open_query('col-b')
    setupColumns(view, [query_a, query_b], [[], []])

    const added = make_kyou({ id: 'new-1' })
    let release_refresh: (() => void) | null = null
    refresh_kyou_mock.mockImplementation(async () => {
      await new Promise<void>((resolve) => { release_refresh = resolve })
      return added
    })

    const inserting = view.insert_registered_kyou(added)
    await flushAsync(2)
    // 飛行中に先頭列を閉じる(querys と match_kyous_list が同時に詰められる)
    view.querys.value.splice(0, 1)
    view.match_kyous_list.value.splice(0, 1)
    release_refresh?.()
    await inserting
    await flushAsync()

    expect(view.querys.value.length).toBe(1)
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new-1'])
  })

  test('引き直しの最中に列の条件が差し替わったら書き戻さない', async () => {
    const { view } = create()
    const query_a = make_open_query('col-a')
    setupColumns(view, [query_a], [[]])

    const added = make_kyou({ id: 'new-1' })
    let release_refresh: (() => void) | null = null
    refresh_kyou_mock.mockImplementation(async () => {
      await new Promise<void>((resolve) => { release_refresh = resolve })
      return added
    })

    const inserting = view.insert_registered_kyou(added)
    await flushAsync(2)
    // 別のquery_idの列に差し替わった = もとの列はもう無い
    view.querys.value[0] = make_open_query('col-z')
    release_refresh?.()
    await inserting
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual([])
  })

  test('全列に同じ requested_at を渡して1往復に合流させる', async () => {
    const { view } = create()
    const query_a = make_open_query('col-a')
    const query_b = make_open_query('col-b')
    setupColumns(view, [query_a, query_b], [[], []])
    refresh_kyou_mock.mockResolvedValue(make_kyou())

    await view.insert_registered_kyou(make_kyou())
    await flushAsync()

    // 同じ引き直しキーの列はまとめられるので往復は1回
    expect(refresh_kyou_mock).toHaveBeenCalledTimes(1)
    expect(refresh_kyou_mock.mock.calls[0][2]).toBe(12345)
  })

  test('crudRelayHandlers.registered_kyou が差し込みと再emitの両方を行う', async () => {
    const { view } = create()
    setupColumns(view, [make_open_query('col-a')], [[]])
    const added = make_kyou({ id: 'new-1' })
    refresh_kyou_mock.mockResolvedValue(added)

    view.crudRelayHandlers.registered_kyou(added)
    await flushAsync()

    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['new-1'])
  })

  test('registered_kyou を伴わない requested_reload_list は従来どおり全列を再検索する', async () => {
    const { pending_get_kyous, view } = create()
    setupColumns(view, [make_open_query('col-a'), make_open_query('col-b')], [[], []])

    view.allColumnsRequestHandlers.requested_reload_list()
    await flushAsync()

    expect(pending_get_kyous.length).toBe(2)
  })
})
