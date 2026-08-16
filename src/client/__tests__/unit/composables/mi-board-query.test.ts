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
import { CheckState } from '@/pages/views/check-state'
import { MI_ALL_BOARD_KEY } from '@/classes/mi-board-names'
import type { MiBoardQueryProps } from '@/pages/views/mi-board-query-props'
import type { MiBoardQueryEmits } from '@/pages/views/mi-board-query-emits'
import type { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'

// 「すべて」の番兵キー。番兵はサイドバー（このコンポーザブル）だけが使い、
// クエリ上は mi_board_name === null が「すべて」。use-mi-query-editor-sidebar.ts の
// generate_query がこの値との比較で null へ戻すので、ここからずれてはいけない。
// **ロケール非依存**であることが要件で、i18n の訳語にすると
// 日本語以外のロケールで「すべて」が全件に戻らなくなる
const ALL_BOARD_SENTINEL = MI_ALL_BOARD_KEY

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

function makeBoardNode(board_name: string, children: Array<MiBoardStructElementData> | null = null): MiBoardStructElementData {
  return {
    name: board_name,
    id: 'id-'.concat(board_name),
    board_name: board_name,
    check_when_inited: true,
    children: children,
    key: board_name,
    is_checked: false,
    indeterminate: false,
    is_dir: children !== null,
  }
}

// 保存済みの MI_BOARD_STRUCT と同じ形。
// ルートは board_name が空・key が __root__ で、子は板のリーフだけ
function makeBoardStruct(board_names: Array<string>): MiBoardStructElementData {
  const root = makeBoardNode('', board_names.map(board_name => makeBoardNode(board_name)))
  root.name = '__root__'
  root.key = '__root__'
  root.board_name = ''
  return root
}

function createHarness(initial_query: FindKyouQuery, board_names: Array<string> = []) {
  const props = reactive({
    gkill_api: { get_session_id: vi.fn(() => 'mock-session') },
    application_config: { mi_board_struct: makeBoardStruct(board_names) },
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

/**
 * 板ツリーのクリック。
 *
 * ルート行は folder_name='' で描いているので見た目は空白だが、
 * .tree_item の min-width:200px ぶんのクリック領域が残っていて、そこを押すと
 * use-foldable-struct.ts の click_group_by_user() が**自分自身の key(__root__)を含めて**
 * サブツリー全部の key を上げてくる。素通しすると `__root__` という名前の列と
 * 全板ぶんの列が一度に開いていた。
 */
describe('useMiBoardQuery onClickedItems', () => {
  const BOARDS = ['すべて', 'Inbox', '板A', '板B']

  function openedBoards(emits: MiBoardQueryEmits): Array<string> {
    return (emits as unknown as ReturnType<typeof vi.fn>).mock.calls
      .filter(call => call[0] === 'request_open_focus_board')
      .map(call => call[1] as string)
  }

  test('ルート行のクリックでは板を1つも開かない', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    // click_group_by_user が上げてくる形。ルート自身の key が先頭に載る
    composable.onClickedItems(
      new MouseEvent('click'),
      ['__root__'].concat(BOARDS),
      CheckState.checked,
      true,
    )

    expect(openedBoards(emits)).toStrictEqual([])
  })

  test('__root__ は板として開かない', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), ['__root__'], CheckState.checked, true)

    expect(openedBoards(emits)).toStrictEqual([])
  })

  test('板のリーフをクリックするとその板だけ開く', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), ['Inbox'], CheckState.checked, true)

    expect(openedBoards(emits)).toStrictEqual(['Inbox'])
    expect(composable.get_board_name()).toBe('Inbox')
  })

  // 板ツリーはいま平坦なのでフォルダはルートだけだが、
  // フォルダを持てるようになってもフォルダ行のクリックで列が生えないこと
  test('フォルダ行のクリックでも何も開かない', async () => {
    const { props, emits, composable } = createHarness(makeAllBoardQuery(), ['Inbox'])
    const struct = (props as unknown as { application_config: { mi_board_struct: MiBoardStructElementData } })
      .application_config.mi_board_struct
    struct.children?.push(makeBoardNode('フォルダ', [makeBoardNode('板C'), makeBoardNode('板D')]))
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(
      new MouseEvent('click'),
      ['フォルダ', '板C', '板D'],
      CheckState.checked,
      true,
    )

    expect(openedBoards(emits)).toStrictEqual([])
  })

  // 板を作った直後は append_not_found_mi_boards() がまだ拾えていないことがある。
  // ここで落とすと「板をクリックしても何も起きない」(エラーも出ない)になる
  test('ツリーにまだ無い板でもリーフのクリックなら開く', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), ['作ったばかりの板'], CheckState.checked, true)

    expect(openedBoards(emits)).toStrictEqual(['作ったばかりの板'])
  })

  test('is_by_user が false なら何も開かない', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), ['Inbox'], CheckState.checked, false)

    expect(openedBoards(emits)).toStrictEqual([])
  })

  test('checked 以外のチェック状態では何も開かない', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), ['Inbox'], CheckState.unchecked, true)
    composable.onClickedItems(new MouseEvent('click'), ['Inbox'], CheckState.indeterminate, true)

    expect(openedBoards(emits)).toStrictEqual([])
  })

  test('「すべて」はツリーの実ノードなので開ける', async () => {
    const { emits, composable } = createHarness(makeAllBoardQuery(), BOARDS)
    await nextTick()
    ;(emits as unknown as ReturnType<typeof vi.fn>).mockClear()

    composable.onClickedItems(new MouseEvent('click'), [ALL_BOARD_SENTINEL], CheckState.checked, true)

    expect(openedBoards(emits)).toStrictEqual([ALL_BOARD_SENTINEL])
  })
})
