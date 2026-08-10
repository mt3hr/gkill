/**
 * useFindQueryEditorView（Dnote/Ryuu などが使う検索条件エディタ）のテスト。
 *
 * ここで固定したいのは「TimeIsのタグツリーへ流し込むチェック集合は timeis_tags であって
 * tags ではない」こと。両者は別のフィルタなのに、rykvサイドバー
 * （use-rykv-query-editor-side-bar.ts）と同型のコードをコピーした際に
 * timeis 側だけ find_query.tags を渡す取り違えが起きていた。
 * 取り違えると「TimeIsのタグ絞り込みを開くと通常タグの選択が写る」ことになる。
 */
import { vi } from 'vitest'
import { nextTick, reactive } from 'vue'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({ generate_uuid: vi.fn(() => 'mock-uuid') })),
    get_gkill_api: vi.fn(() => ({ generate_uuid: vi.fn(() => 'mock-uuid') })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { useFindQueryEditorView } from '@/classes/use-find-query-editor-view'

// ApplicationConfig の実物は req_res との循環importを引き込むため、
// generate_default_query_for_rykv / apply_hide_tags が触るフィールドだけ持つ fake を使う
function make_fake_application_config(): Record<string, unknown> {
  const empty_struct = (extra: Record<string, unknown>) => ({
    key: '', check_when_inited: false, is_checked: false, indeterminate: false, is_force_hide: false, children: [], ...extra,
  })
  const config: Record<string, unknown> = {
    // check_when_inited のタグが既定クエリの tags に入る。timeis_tags には入らない
    tag_struct: empty_struct({
      tag_name: '',
      children: [empty_struct({ tag_name: 'inited-tag', check_when_inited: true })],
    }),
    rep_struct: empty_struct({ rep_name: '' }),
    device_struct: empty_struct({ device_name: '' }),
    rep_type_struct: empty_struct({ rep_type_name: '' }),
    rykv_default_period: -1,
    google_map_api_key: '',
  }
  config.clone = () => ({ ...config })
  return config
}

function create_view() {
  const props = reactive({
    gkill_api: { generate_uuid: vi.fn(() => 'generated-uuid') },
    application_config: make_fake_application_config(),
    find_kyou_query: new FindKyouQuery(),
  })
  const view = useFindQueryEditorView({ props: props as never, emits: vi.fn() as never })

  const timeis_update_check = vi.fn()
  const tag_update_check = vi.fn()

  // 子コンポーネントの代わり。generate_query が呼ぶ getter だけ生やす
  view.timeis_query.value = {
    get_use_timeis: () => true,
    get_use_timeis_tags: () => true,
    get_timeis_tags: () => ['timeis-tag'],
    get_timeis_keywords: () => '',
    get_use_and_search_timeis_words: () => false,
    get_use_and_search_timeis_tags: () => false,
    update_check: timeis_update_check,
  } as never
  view.tag_query.value = {
    get_tags: () => ['regular-tag'],
    get_is_and_search: () => false,
    update_check: tag_update_check,
  } as never

  return { props, view, timeis_update_check, tag_update_check }
}

describe('useFindQueryEditorView: TimeIsタグツリーへ渡すチェック集合', () => {
  test('TimeIs検索条件のクリアでは timeis_tags を流し込む（通常タグではない）', async () => {
    const { view, timeis_update_check } = create_view()
    await nextTick()

    view.emits_cleard_timeis_query()

    expect(timeis_update_check).toHaveBeenCalledTimes(1)
    const passed_items = timeis_update_check.mock.calls[0][0]
    expect(passed_items, 'rykv既定クエリの timeis_tags は null なので空集合になる').toEqual([])
    expect(passed_items, '通常タグの選択がTimeIsのタグツリーへ写ってはいけない').not.toContain('regular-tag')
  })

  test('既定条件へ戻すときも、通常タグとTimeIsタグをそれぞれのツリーへ渡す', async () => {
    const { view, timeis_update_check, tag_update_check } = create_view()
    await nextTick()

    await view.emits_default_query()

    expect(tag_update_check.mock.calls[0][0], '通常タグツリーには初期チェックタグが入る').toEqual(['inited-tag'])
    expect(timeis_update_check.mock.calls[0][0], 'TimeIsタグツリーは timeis_tags 由来（既定はnull=空集合）').toEqual([])
  })
})
