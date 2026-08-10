/**
 * useMiBoardQuery のテスト。
 * サイドバーの板選択がフォーカス列の検索条件(find_kyou_query)に追随することを固定する。
 * 追随しないと最後にクリックした板名がサイドバーに残り続け、generate_query経由で
 * 別列の検索条件に混入する(板クリック後に「すべて」列で検索すると板名が化けるバグ)。
 */
import { vi } from 'vitest'
import { nextTick, reactive } from 'vue'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { useMiBoardQuery } from '@/classes/use-mi-board-query'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { MiBoardQueryProps } from '@/pages/views/mi-board-query-props'
import type { MiBoardQueryEmits } from '@/pages/views/mi-board-query-emits'

// 「すべて」の番兵キー。番兵文字列は表示層（このコンポーザブル）だけが使い、
// クエリ上は mi_board_name === null が「すべて」。use-mi-query-editor-sidebar.ts の
// generate_query がこの値との比較で null へ戻すので、ここからずれてはいけない
const ALL_BOARD_SENTINEL = 'MI_ALL_BOARD_NAME_TITLE'

function makeBoardQuery(board_name: string): FindKyouQuery {
  const query = new FindKyouQuery()
  query.query_id = 'query-'.concat(board_name)
  // 非null=板絞り込みあり（use_mi_board_name フラグは全廃済み）
  query.mi_board_name = board_name
  return query
}

function makeAllBoardQuery(): FindKyouQuery {
  const query = new FindKyouQuery()
  query.query_id = 'query-all'
  // null=「すべて」（コンストラクタ既定のまま。明示は意図の表明）
  query.mi_board_name = null
  return query
}

function createHarness(initial_query: FindKyouQuery) {
  const props = reactive({
    gkill_api: { get_session_id: vi.fn(() => 'mock-session') },
    application_config: { mi_board_struct: { children: [] } },
    app_content_height: 600,
    app_content_width: 800,
    inited: true,
    find_kyou_query: initial_query,
  }) as unknown as MiBoardQueryProps
  const emits = vi.fn() as unknown as MiBoardQueryEmits
  const composable = useMiBoardQuery({ props, emits })
  return { props, emits, composable }
}

describe('useMiBoardQuery', () => {
  test('初期状態: 板絞り込みなしのクエリなら「すべて」の番兵になる', async () => {
    const { composable } = createHarness(makeAllBoardQuery())
    await nextTick()
    expect(composable.get_board_name()).toBe(ALL_BOARD_SENTINEL)
  })

  test('初期状態: 板絞り込みありのクエリなら板名になる', async () => {
    const { composable } = createHarness(makeBoardQuery('board-a'))
    await nextTick()
    expect(composable.get_board_name()).toBe('board-a')
  })

  test('フォーカス列の切り替え(クエリ差し替え)に板名が追随する', async () => {
    const { props, composable } = createHarness(makeBoardQuery('board-a'))
    await nextTick()
    expect(composable.get_board_name()).toBe('board-a')

    // 板Aの列 → 「すべて」列にフォーカスが移った状況
    ;(props as { find_kyou_query: FindKyouQuery }).find_kyou_query = makeAllBoardQuery()
    await nextTick()
    expect(composable.get_board_name()).toBe(ALL_BOARD_SENTINEL)

    // 「すべて」列 → 板Bの列
    ;(props as { find_kyou_query: FindKyouQuery }).find_kyou_query = makeBoardQuery('board-b')
    await nextTick()
    expect(composable.get_board_name()).toBe('board-b')
  })

  test('ユーザの板クリック(board_name直接代入)は次のフォーカス切り替えで上書きされる', async () => {
    const { props, composable } = createHarness(makeAllBoardQuery())
    await nextTick()

    // mi-board-query.vue のクリックハンドラ相当
    composable.board_name.value = 'board-clicked'
    expect(composable.get_board_name()).toBe('board-clicked')

    // クリックを受けて親がフォーカス列を切り替えると、その列の条件に揃う
    ;(props as { find_kyou_query: FindKyouQuery }).find_kyou_query = makeBoardQuery('board-clicked')
    await nextTick()
    expect(composable.get_board_name()).toBe('board-clicked')

    // 別の列(すべて)へフォーカスすると板名は残らない
    ;(props as { find_kyou_query: FindKyouQuery }).find_kyou_query = makeAllBoardQuery()
    await nextTick()
    expect(composable.get_board_name()).toBe(ALL_BOARD_SENTINEL)
  })

  test('inited を emit する', async () => {
    const { emits } = createHarness(makeAllBoardQuery())
    await nextTick()
    expect(emits).toHaveBeenCalledWith('inited')
  })
})
