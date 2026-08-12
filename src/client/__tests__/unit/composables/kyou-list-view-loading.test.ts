/**
 * useKyouListView のローディング表示状態のテスト。
 * 列のkeyはquery_idで安定化されており、再検索でコンポーネントがremountされない。
 * そのため set_loading(true) が has_loaded を倒さないと、検索開始時に空にされた
 * リストが「読み込み中」ではなく「該当なし」と誤表示される。
 */
import { describe, test, expect, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useKyouListView } from '@/classes/use-kyou-list-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { KyouListViewProps } from '@/pages/views/kyou-list-view-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'

const noop_emits = (() => { }) as unknown as KyouListViewEmits

function createProps(): KyouListViewProps {
  const query = new FindKyouQuery()
  query.query_id = 'col-a'
  return {
    query,
    matched_kyous: [],
    application_config: { rykv_image_list_column_number: 3 },
    kyou_height: 180,
    width: 400,
    show_footer: true,
    is_focused_list: false,
    closable: true,
    is_readonly_mi_check: false,
    enable_context_menu: true,
    enable_dialog: true,
  } as unknown as KyouListViewProps
}

describe('useKyouListView loading state', () => {
  test('set_loading(true)でhas_loadedが倒れ、set_loading(false)で立つ', () => {
    const view = useKyouListView({ props: createProps(), emits: noop_emits })

    expect(view.has_loaded.value).toBe(false)
    expect(view.is_loading.value).toBe(false)

    view.set_loading(true)
    expect(view.is_loading.value).toBe(true)
    expect(view.has_loaded.value).toBe(false)

    view.set_loading(false)
    expect(view.is_loading.value).toBe(false)
    expect(view.has_loaded.value).toBe(true)

    // 再検索(remountなし)でも「該当なし」ではなく「読み込み中」へ戻る
    view.set_loading(true)
    expect(view.has_loaded.value).toBe(false)
    expect(view.get_is_loading()).toBe(true)

    view.set_loading(false)
    expect(view.has_loaded.value).toBe(true)
  })
})
