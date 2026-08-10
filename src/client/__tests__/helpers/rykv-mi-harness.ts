/**
 * useRykvView / useMiView の「列×検索」テスト用ハーネス。
 *
 * - get_kyous を deferred 化し、解決順を試験側が制御してレースを決定論的に再現する
 * - v-for テンプレートref (kyou_list_views) には query_id を返す fake を手動注入する
 * - saved querys / scroll indexs は in-memory に置き換える
 *
 * vi.mock はモジュールごとにホイストされるため、ここでは行わない。
 * 各テストファイル側で @/i18n・@/router・@/classes/delete-gkill-cache・
 * @/classes/kyou-reload 等を mock してからこのヘルパーを使うこと。
 */
import { vi } from 'vitest'
import { createMockGkillAPI } from './mock-api'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export interface PendingGetKyous {
  req: { query: FindKyouQuery }
  resolve: (res: { kyous: unknown[]; messages: unknown[]; errors: unknown[] }) => void
}

/** 列ビュー(rykv/mi)用のモックAPI。get_kyous は resolve するまで完了しない */
export function createColumnViewMockApi() {
  let uuid_count = 0
  const pending_get_kyous: PendingGetKyous[] = []
  const saved = {
    rykv_querys: [] as unknown[],
    mi_querys: [] as unknown[],
    rykv_scrolls: [] as number[],
    mi_scrolls: [] as number[],
  }
  const api = {
    ...createMockGkillAPI(),
    // 既定モックは固定UUIDを返すため、query_idの一意性を試験できるよう連番にする
    generate_uuid: vi.fn(() => `generated-uuid-${++uuid_count}`),
    delete_updated_gkill_caches: vi.fn().mockResolvedValue(undefined),
    get_kyous: vi.fn(
      (req: { query: FindKyouQuery }) =>
        new Promise((resolve) => {
          pending_get_kyous.push({ req, resolve: resolve as PendingGetKyous['resolve'] })
        }),
    ),
    get_saved_rykv_find_kyou_querys: vi.fn(() => saved.rykv_querys),
    set_saved_rykv_find_kyou_querys: vi.fn((querys: unknown[]) => {
      saved.rykv_querys = querys.concat()
    }),
    get_saved_rykv_scroll_indexs: vi.fn(() => saved.rykv_scrolls),
    set_saved_rykv_scroll_indexs: vi.fn((indexs: number[]) => {
      saved.rykv_scrolls = indexs.concat()
    }),
    get_saved_mi_find_kyou_querys: vi.fn(() => saved.mi_querys),
    set_saved_mi_find_kyou_querys: vi.fn((querys: unknown[]) => {
      saved.mi_querys = querys.concat()
    }),
    get_saved_mi_scroll_indexs: vi.fn(() => saved.mi_scrolls),
    set_saved_mi_scroll_indexs: vi.fn((indexs: number[]) => {
      saved.mi_scrolls = indexs.concat()
    }),
  }
  return { api, pending_get_kyous }
}

export interface FakeKyouListView {
  get_query_id: () => string
  set_loading: ReturnType<typeof vi.fn>
  get_is_loading: () => boolean
  scroll_to: ReturnType<typeof vi.fn>
  scroll_to_time: ReturnType<typeof vi.fn>
  scroll_to_kyou: ReturnType<typeof vi.fn>
}

/** KyouListView の defineExpose 相当の fake */
export function makeFakeKyouListView(query_id: string): FakeKyouListView {
  let is_loading = false
  return {
    get_query_id: () => query_id,
    set_loading: vi.fn((loading: boolean) => {
      is_loading = loading
    }),
    get_is_loading: () => is_loading,
    scroll_to: vi.fn(),
    scroll_to_time: vi.fn(),
    scroll_to_kyou: vi.fn(),
  }
}

export function makeColumnQuery(query_id: string): FindKyouQuery {
  const query = new FindKyouQuery()
  query.query_id = query_id
  return query
}

export function makeViewApplicationConfig(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    rykv_hot_reload: true,
    rykv_image_list_column_number: 3,
    device: 'test-device',
    user_id: 'testuser',
    is_show_share_footer: false,
    ...overrides,
  }
}

/** composable の返り値のうちハーネスが触る部分 */
export interface ColumnViewLike {
  inited: { value: boolean }
  querys: { value: FindKyouQuery[] }
  match_kyous_list: { value: unknown[][] }
  focused_column_index: { value: number }
  focused_query: { value: FindKyouQuery }
  kyou_list_views: { value: FakeKyouListView[] }
  query_editor_sidebar: { value: unknown }
}

/**
 * init() を経由せず、列の状態を直接組み立てる。
 * 返り値は query_id → fake list view のマップ(スパイの検証用)
 */
export function setupColumns(
  view: ColumnViewLike,
  querys: FindKyouQuery[],
  lists: unknown[][],
): Map<string, FakeKyouListView> {
  const fakes = new Map<string, FakeKyouListView>()
  view.querys.value = querys
  view.match_kyous_list.value = lists
  view.kyou_list_views.value = querys.map((query) => {
    const fake = makeFakeKyouListView(query.query_id)
    fakes.set(query.query_id, fake)
    return fake
  })
  view.focused_column_index.value = 0
  view.focused_query.value = querys[0]
  view.inited.value = true
  return fakes
}

/** microtask と setTimeout(0) を消化して nextTick / await を進める */
export async function flushAsync(times = 20): Promise<void> {
  for (let i = 0; i < times; i++) {
    await Promise.resolve()
    await new Promise((resolve) => setTimeout(resolve, 0))
  }
}
