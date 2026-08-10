/**
 * useKyouListView.scroll_to の永久リトライ打ち切りの検証。
 *
 * 以前は世代も上限も無い自己再帰で、0件のまま終わった列(scrollHeight=0)や
 * 別条件の再検索で件数が減った列(保存済みスクロール位置>コンテンツ高)に対して
 * 50ms間隔の強制レイアウト(scrollHeight読み)が永久に残り、列を開く/クリックする
 * たびにチェーンが増殖してレンダラを飽和させていた(2026-08-10 rykvタブフリーズの
 * 回復不能化要因)。
 */
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_tags_cache: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useKyouListView } from '@/classes/use-kyou-list-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { KyouListViewProps } from '@/pages/views/kyou-list-view-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'

const noop_emits = (() => { }) as unknown as KyouListViewEmits

function createProps(query_id = 'col-a'): KyouListViewProps {
  const query = new FindKyouQuery()
  query.query_id = query_id
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

/** 列のリスト要素とv-virtual-scrollコンテナのfake DOM。scrollHeightは実測できないので固定値を与える */
function mountListElement(query_id: string, scroll_height: number): HTMLElement {
  const element = document.createElement('div')
  element.id = `${query_id}_kyou_list_view`
  const container = document.createElement('div')
  container.className = 'v-virtual-scroll__container'
  Object.defineProperty(container, 'scrollHeight', { value: scroll_height, configurable: true })
  element.appendChild(container)
  document.body.appendChild(element)
  return element
}

async function flushMicrotasks(times = 10): Promise<void> {
  for (let i = 0; i < times; i++) {
    await Promise.resolve()
  }
}

describe('useKyouListView scroll_to', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  test('対象要素が現れなくてもリトライは上限で止まる(タイマーが残らない)', async () => {
    const view = useKyouListView({ props: createProps('ghost-col'), emits: noop_emits })

    view.scroll_to(100)
    await flushMicrotasks()

    // 上限(40回)ぶん+余裕まで時間を進め、タイマーが完全に絶えることを確認する。
    // 旧実装は打ち切りが無いので、いくら進めても常に次のタイマーが残っていた
    await vi.advanceTimersByTimeAsync(50 * 60)
    expect(vi.getTimerCount(), '打ち切り後にリトライタイマーが残ってはいけない').toBe(0)
  })

  test('コンテンツ高が目標に届かないまま上限に達したらクランプ代入で打ち切る', async () => {
    const element = mountListElement('short-col', 300)
    const view = useKyouListView({ props: createProps('short-col'), emits: noop_emits })

    // 別条件の再検索で件数が減った列に、以前の(今より大きい)スクロール位置を復元するケース
    view.scroll_to(1000)
    await flushMicrotasks()
    await vi.advanceTimersByTimeAsync(50 * 60)

    expect(vi.getTimerCount()).toBe(0)
    // jsdomはscrollTopをクランプしないので代入値がそのまま残る。
    // 実ブラウザでは最大スクロール量へクランプされる
    expect(element.scrollTop, '諦めるときはそのまま代入して打ち切る(ブラウザがクランプする)').toBe(1000)
  })

  test('新しいscroll_to呼び出しが古いリトライチェーンを破棄する', async () => {
    const element = mountListElement('col-a', 300)
    const view = useKyouListView({ props: createProps('col-a'), emits: noop_emits })

    // 届かない位置への呼び出しでリトライ中…
    view.scroll_to(1000)
    await flushMicrotasks()
    await vi.advanceTimersByTimeAsync(200)

    // …のうちに届く位置への新しい呼び出しが来たら、そちらが即座に勝つ
    view.scroll_to(100)
    await flushMicrotasks()
    expect(element.scrollTop).toBe(100)

    // 古いチェーンが上限まで生き延びてクランプ代入(1000)で上書きしない
    await vi.advanceTimersByTimeAsync(50 * 60)
    expect(element.scrollTop, '破棄されたチェーンがスクロール位置を上書きしてはいけない').toBe(100)
    expect(vi.getTimerCount()).toBe(0)
  })
})
