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
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useKyouListView } from '@/classes/use-kyou-list-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { Kyou } from '@/classes/datas/kyou'
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

/**
 * scroll_to_time は for_mi で分岐する。
 * 非mi(rykv)は related_time 降順で「time 以降で最も近い行」、
 * mi列はソート基準時刻の昇順＋未設定(末尾)なので「ソート基準時刻を持つ行のうち
 * 最初に related_time >= time を満たす行」へ寄せる。
 * ソート基準時刻の有無判定は kyou-local-insert.ts の has_mi_sort_key を共有する。
 */
describe('useKyouListView scroll_to_time', () => {
  function makeKyou(id: string, data_type: string, related_time: string): Kyou {
    return { id, data_type, related_time: new Date(related_time) } as unknown as Kyou
  }

  function buildProps(matched_kyous: Array<Kyou>, for_mi: boolean, mi_sort_type = MiSortType.estimate_start_time): KyouListViewProps {
    const query = new FindKyouQuery()
    query.query_id = 'mi-col'
    query.for_mi = for_mi
    query.mi_sort_type = mi_sort_type
    return {
      query,
      matched_kyous,
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

  /** scroll_to_time が呼ぶ scrollToIndex を差し替えて、寄せた index を観測する */
  function mountViewWith(props: KyouListViewProps): { view: ReturnType<typeof useKyouListView>, scrollToIndex: ReturnType<typeof vi.fn> } {
    const view = useKyouListView({ props, emits: noop_emits })
    const scrollToIndex = vi.fn()
    view.kyou_list_view.value = { scrollToIndex } as unknown as typeof view.kyou_list_view.value
    return { view, scrollToIndex }
  }

  test('(1) for_mi=false の降順では time 以降で最も近い行へ寄せる(現行不変)', async () => {
    const list = [
      makeKyou('a', 'kmemo', '2026-08-03T00:00:00.000Z'),
      makeKyou('b', 'kmemo', '2026-08-02T00:00:00.000Z'),
      makeKyou('c', 'kmemo', '2026-08-01T00:00:00.000Z'),
    ]
    const { view, scrollToIndex } = mountViewWith(buildProps(list, false))

    const result = await view.scroll_to_time(new Date('2026-08-02T12:00:00.000Z'))

    expect(result).toBe(true)
    // 08-03 は time より新しいので飛ばし、最初に time 以下になる 08-02(index 1)へ
    expect(scrollToIndex).toHaveBeenCalledWith(1)
  })

  test('(2) for_mi=true の昇順では対象時刻の直後の行へ寄せる', async () => {
    const list = [
      makeKyou('a', 'mi_start', '2026-08-01T00:00:00.000Z'),
      makeKyou('b', 'mi_start', '2026-08-03T00:00:00.000Z'),
      makeKyou('c', 'mi_start', '2026-08-05T00:00:00.000Z'),
    ]
    const { view, scrollToIndex } = mountViewWith(buildProps(list, true))

    const result = await view.scroll_to_time(new Date('2026-08-02T00:00:00.000Z'))

    expect(result).toBe(true)
    // 昇順なので最初に time 以上になる 08-03(index 1)へ
    expect(scrollToIndex).toHaveBeenCalledWith(1)
  })

  test('(3) for_mi=true では未設定(末尾)セグメントの行はスキップする', async () => {
    // 末尾の未設定行 c は related_time(作成日時フォールバック)が time 以上でも、
    // ソート基準時刻を持たないので寄せ先にしない
    const list = [
      makeKyou('a', 'mi_start', '2026-08-01T00:00:00.000Z'),
      makeKyou('b', 'mi_start', '2026-08-03T00:00:00.000Z'),
      makeKyou('c', 'mi_create', '2026-08-10T00:00:00.000Z'),
    ]
    const { view, scrollToIndex } = mountViewWith(buildProps(list, true))

    const result = await view.scroll_to_time(new Date('2026-08-05T00:00:00.000Z'))

    // キー付き行はどれも time 未満、唯一 time 以上の c はキー無しでスキップ → false
    expect(result).toBe(false)
    expect(scrollToIndex).not.toHaveBeenCalled()
  })

  test('(4) for_mi=true で全行が対象より新しいときは先頭のキー付き行へ寄せる', async () => {
    const list = [
      makeKyou('a', 'mi_start', '2026-08-05T00:00:00.000Z'),
      makeKyou('b', 'mi_start', '2026-08-07T00:00:00.000Z'),
    ]
    const { view, scrollToIndex } = mountViewWith(buildProps(list, true))

    const result = await view.scroll_to_time(new Date('2026-08-01T00:00:00.000Z'))

    // 全行が time 以上なので最初のキー付き行(index 0)が最も近い
    expect(result).toBe(true)
    expect(scrollToIndex).toHaveBeenCalledWith(0)
  })

  test('(5) for_mi=true でキー付き行がどれも time 未満なら false', async () => {
    const list = [
      makeKyou('a', 'mi_start', '2026-08-01T00:00:00.000Z'),
      makeKyou('b', 'mi_start', '2026-08-03T00:00:00.000Z'),
    ]
    const { view, scrollToIndex } = mountViewWith(buildProps(list, true))

    const result = await view.scroll_to_time(new Date('2026-08-10T00:00:00.000Z'))

    expect(result).toBe(false)
    expect(scrollToIndex).not.toHaveBeenCalled()
  })
})
