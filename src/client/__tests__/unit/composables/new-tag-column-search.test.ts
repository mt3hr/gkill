/**
 * 「利用者がその場で作ったタグを、開いている列の検索条件へ足す」配線を
 * rykv / mi の**両方のビューに対して**確かめる。
 *
 * コンポーザブル単体の試験は registered-tag-column-filter.test.ts が持っているので、
 * ここが見るのは配線そのもの ―― 判定が emit より前で走ること、
 * 引き直しが列あたり1本であること、条件が localStorage へ落ちること。
 *
 * rykv と mi はコピー由来の対称実装なので、片方だけ配線が抜けても
 * 「rykvでは残るが mi では消える」という気付きにくいズレになる。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'

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
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Tag } from '@/classes/datas/tag'
import {
  createColumnViewMockApi,
  makeViewApplicationConfig,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

/** タグツリーに「既知タグ」だけがある状態 */
function make_tag_struct(): unknown {
  return {
    key: '__root__',
    name: '',
    tag_name: '',
    is_dir: true,
    children: [
      { key: '既知タグ', name: '既知タグ', tag_name: '既知タグ', children: null },
    ],
  }
}

function make_tag(tag_name: string): Tag {
  const tag = new Tag()
  tag.tag = tag_name
  return tag
}

/** タグで絞っている列。既定クエリが「絞らない」を列挙で物質化した状態を模す */
function make_tag_filtered_query(query_id: string, tags: Array<string> | null, tags_and = false): FindKyouQuery {
  const query = new FindKyouQuery()
  query.query_id = query_id
  query.tags = tags
  query.tags_and = tags_and
  query.reps = null
  return query
}

interface ColumnViewUnderTest {
  querys: { value: Array<FindKyouQuery> }
  match_kyous_list: { value: Array<Array<unknown>> }
  crudRelayHandlers: { registered_tag: (tag: Tag) => void }
}

interface ViewCase {
  name: string
  saved_querys_key: 'set_saved_rykv_find_kyou_querys' | 'set_saved_mi_find_kyou_querys'
  create: (emits: (event: string, ...args: unknown[]) => void) => {
    api: ReturnType<typeof createColumnViewMockApi>['api']
    pending_get_kyous: ReturnType<typeof createColumnViewMockApi>['pending_get_kyous']
    view: ColumnViewUnderTest
  }
}

const view_cases: ViewCase[] = [
  {
    name: 'useRykvView',
    saved_querys_key: 'set_saved_rykv_find_kyou_querys',
    create: (emits_spy) => {
      const { api, pending_get_kyous } = createColumnViewMockApi()
      const props = {
        gkill_api: api,
        application_config: makeViewApplicationConfig({ tag_struct: make_tag_struct() }),
        app_title_bar_height: 50,
        app_content_height: 600,
        app_content_width: 800,
        is_shared_rykv_view: false,
        share_title: '',
      } as unknown as Parameters<typeof useRykvView>[0]['props']
      const emits = emits_spy as unknown as Parameters<typeof useRykvView>[0]['emits']
      return { api, pending_get_kyous, view: useRykvView({ props, emits }) as unknown as ColumnViewUnderTest }
    },
  },
  {
    name: 'useMiView',
    saved_querys_key: 'set_saved_mi_find_kyou_querys',
    create: (emits_spy) => {
      const { api, pending_get_kyous } = createColumnViewMockApi()
      const props = {
        gkill_api: api,
        application_config: makeViewApplicationConfig({ tag_struct: make_tag_struct() }),
        app_title_bar_height: 50,
        app_content_height: 600,
        app_content_width: 800,
      } as unknown as Parameters<typeof useMiView>[0]['props']
      const emits = emits_spy as unknown as Parameters<typeof useMiView>[0]['emits']
      return { api, pending_get_kyous, view: useMiView({ props, emits }) as unknown as ColumnViewUnderTest }
    },
  },
]

describe.each(view_cases)('$name 新規タグの列条件への反映', ({ create, saved_querys_key }) => {
  let emits_spy: ReturnType<typeof vi.fn>

  beforeEach(() => {
    emits_spy = vi.fn()
  })

  test('未知タグを付けると、その列の tags へ足して1本だけ引き直す', async () => {
    const { pending_get_kyous, view } = create(emits_spy)
    setupColumns(view, [make_tag_filtered_query('col-a', ['no tags'])], [[]])

    view.crudRelayHandlers.registered_tag(make_tag('新タグ'))
    await nextTick()
    await flushAsync()

    expect(view.querys.value[0].tags).toEqual(['no tags', '新タグ'])
    expect(pending_get_kyous.length, '引き直しは列あたり1本').toBe(1)
  })

  test('列の同一性(query_id)は保たれる', async () => {
    const { view } = create(emits_spy)
    setupColumns(view, [make_tag_filtered_query('col-a', ['no tags'])], [[]])

    view.crudRelayHandlers.registered_tag(make_tag('新タグ'))
    await nextTick()
    await flushAsync()

    expect(view.querys.value[0].query_id).toBe('col-a')
  })

  test('引き直しを通じて新タグ入りの条件が保存される', async () => {
    const { api, view } = create(emits_spy)
    setupColumns(view, [make_tag_filtered_query('col-a', ['no tags'])], [[]])

    view.crudRelayHandlers.registered_tag(make_tag('新タグ'))
    await nextTick()
    await flushAsync()

    // 自分で localStorage へ書かず、search() の set_saved_* に乗ることを確かめる。
    // 「条件だけ変わって引き直さない」経路を作ると、次回起動時だけ列が変わる
    const saved = api[saved_querys_key] as ReturnType<typeof vi.fn>
    expect(saved, '検索を通っていない（条件だけ書き換えている）').toHaveBeenCalled()
    const last_saved = saved.mock.calls[saved.mock.calls.length - 1][0] as Array<FindKyouQuery>
    expect(last_saved[0].tags).toEqual(['no tags', '新タグ'])
  })

  test('既知のタグでは列も引き直しも動かない（利用者が外したチェックを尊重する）', async () => {
    const { pending_get_kyous, view } = create(emits_spy)
    setupColumns(view, [make_tag_filtered_query('col-a', ['no tags'])], [[]])

    view.crudRelayHandlers.registered_tag(make_tag('既知タグ'))
    await nextTick()
    await flushAsync()

    expect(view.querys.value[0].tags).toEqual(['no tags'])
    expect(pending_get_kyous.length).toBe(0)
  })

  test('tags が null の列と tags_and の列は引き直さない', async () => {
    const { pending_get_kyous, view } = create(emits_spy)
    setupColumns(view, [
      make_tag_filtered_query('col-null', null),
      make_tag_filtered_query('col-and', ['no tags'], true),
      make_tag_filtered_query('col-or', ['no tags']),
    ], [[], [], []])

    view.crudRelayHandlers.registered_tag(make_tag('新タグ'))
    await nextTick()
    await flushAsync()

    expect(view.querys.value[0].tags).toBeNull()
    // AND 列に足すと ["no tags","新タグ"] の積が必ず空になり、列が丸ごと消える
    expect(view.querys.value[1].tags).toEqual(['no tags'])
    expect(view.querys.value[2].tags).toEqual(['no tags', '新タグ'])
    expect(pending_get_kyous.length).toBe(1)
  })

  test('親への registered_tag の中継は止めない（タグツリーの同期はそちらが行う）', async () => {
    const { view } = create(emits_spy)
    setupColumns(view, [make_tag_filtered_query('col-a', ['no tags'])], [[]])

    view.crudRelayHandlers.registered_tag(make_tag('新タグ'))
    await nextTick()
    await flushAsync()

    expect(emits_spy).toHaveBeenCalledWith('registered_tag', expect.objectContaining({ tag: '新タグ' }))
  })
})
