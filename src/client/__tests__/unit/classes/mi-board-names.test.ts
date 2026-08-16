/**
 * mi-board-names.ts のテスト。
 *
 * 板名プルダウンの並び順を ApplicationConfig の板ツリーへ揃える純関数を固定する。
 * サーバの get_mi_board_list は Go の map を回して集めているので順序を保証せず
 * (dao/reps/mi_repositories.go / mi_re_kyou_repositories.go。doc コメントにも明記)、
 * 素で :items に渡すと読み込むたびにプルダウンの並びが入れ替わる。
 */
import { MI_ALL_BOARD_KEY, collect_mi_board_names_in_config_order, resolve_clicked_mi_board_names, sort_mi_board_names_by_config_order } from '@/classes/mi-board-names'
import type { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'

function make_node(board_name: string, children: Array<MiBoardStructElementData> | null = null): MiBoardStructElementData {
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
// ルートは board_name が空で key が __root__、子は板のリーフだけのフラット構造
function make_root(board_names: Array<string>): MiBoardStructElementData {
    const root = make_node('', board_names.map(board_name => make_node(board_name)))
    root.name = '__root__'
    root.key = '__root__'
    root.board_name = ''
    return root
}

// 「利用者が設定ダイアログで並べた順」を表す板の列。Inbox は追加ビューの既定値
const CONFIGURED_BOARDS = ['Inbox', '板A', '板B', '板C', '板D', '板E', '板F', '板G', '板H']

describe('collect_mi_board_names_in_config_order', () => {
    test('children の順で board_name を返す', () => {
        expect(collect_mi_board_names_in_config_order(make_root(['b', 'a', 'c'])))
            .toStrictEqual(['b', 'a', 'c'])
    })

    test('board_name が空のルートは含めない', () => {
        const names = collect_mi_board_names_in_config_order(make_root(['Inbox']))
        expect(names).toStrictEqual(['Inbox'])
        expect(names).not.toContain('')
    })

    test('入れ子のフォルダも DFS 順に辿る', () => {
        const root = make_root([])
        root.children = [
            make_node('Inbox'),
            make_node('folder', [make_node('板A'), make_node('板B')]),
            make_node('板C'),
        ]
        // フォルダ自身も board_name を持っていれば拾う(判定はあくまで board_name の有無)
        expect(collect_mi_board_names_in_config_order(root))
            .toStrictEqual(['Inbox', 'folder', '板A', '板B', '板C'])
    })

    test('children が null / struct が null でも落ちない', () => {
        expect(collect_mi_board_names_in_config_order(make_node('x'))).toStrictEqual(['x'])
        expect(collect_mi_board_names_in_config_order(null)).toStrictEqual([])
    })
})

describe('sort_mi_board_names_by_config_order', () => {
    test('APIの順がばらばらでも設定順に並ぶ', () => {
        const struct = make_root(CONFIGURED_BOARDS)
        // マップ反復順を模したシャッフル
        const from_api = ['板E', '板H', 'Inbox', '板F', '板C', '板G', '板B', '板D', '板A']
        expect(sort_mi_board_names_by_config_order(from_api, struct)).toStrictEqual(CONFIGURED_BOARDS)
    })

    test('呼ぶたびにAPIの順が変わっても結果は同じ', () => {
        const struct = make_root(CONFIGURED_BOARDS)
        const first = sort_mi_board_names_by_config_order(['板F', 'Inbox', '板C'], struct)
        const second = sort_mi_board_names_by_config_order(['板C', '板F', 'Inbox'], struct)
        expect(first).toStrictEqual(second)
    })

    test('設定に無い板はAPIの順のまま末尾へ', () => {
        const struct = make_root(['Inbox', '板A'])
        expect(sort_mi_board_names_by_config_order(['新板B', '板A', '新板A', 'Inbox'], struct))
            .toStrictEqual(['Inbox', '板A', '新板B', '新板A'])
    })

    test('設定にしか無い板名は足さない(「すべて」を候補へ混ぜない)', () => {
        // append_all_mi_board() が入れる仮想ノード「すべて」はツリー先頭にいる
        const struct = make_root([MI_ALL_BOARD_KEY].concat(['Inbox', '板A']))
        const sorted = sort_mi_board_names_by_config_order(['板A', 'Inbox'], struct)
        expect(sorted).toStrictEqual(['Inbox', '板A'])
        expect(sorted).not.toContain(MI_ALL_BOARD_KEY)
    })

    test('「すべて」という名前の板が実在すればそれは残る', () => {
        const struct = make_root([MI_ALL_BOARD_KEY, 'Inbox'])
        expect(sort_mi_board_names_by_config_order(['Inbox', MI_ALL_BOARD_KEY], struct))
            .toStrictEqual([MI_ALL_BOARD_KEY, 'Inbox'])
    })

    test('重複は畳む', () => {
        const struct = make_root(['Inbox', '板A'])
        expect(sort_mi_board_names_by_config_order(['板A', 'Inbox', '板A', '新板', '新板'], struct))
            .toStrictEqual(['Inbox', '板A', '新板'])
    })

    test('ルートの空文字は結果に出ない', () => {
        const struct = make_root(['Inbox'])
        expect(sort_mi_board_names_by_config_order(['Inbox'], struct)).toStrictEqual(['Inbox'])
    })

    test('設定の読み込み前(struct が null / children が null)はAPIの順のまま', () => {
        expect(sort_mi_board_names_by_config_order(['b', 'a'], null)).toStrictEqual(['b', 'a'])
        expect(sort_mi_board_names_by_config_order(['b', 'a'], make_root([]))).toStrictEqual(['b', 'a'])
    })

    test('空の入力は空のまま', () => {
        expect(sort_mi_board_names_by_config_order([], make_root(['Inbox']))).toStrictEqual([])
    })

    test('入力配列を破壊しない', () => {
        const from_api = ['板A', 'Inbox']
        sort_mi_board_names_by_config_order(from_api, make_root(['Inbox', '板A']))
        expect(from_api).toStrictEqual(['板A', 'Inbox'])
    })
})

describe('resolve_clicked_mi_board_names', () => {
    const struct = make_root([MI_ALL_BOARD_KEY].concat(CONFIGURED_BOARDS))

    test('ルート行のクリックでは何も開かない', () => {
        // click_group_by_user が上げてくる形。ルート自身の key が先頭に載る
        const items = ['__root__', MI_ALL_BOARD_KEY].concat(CONFIGURED_BOARDS)
        expect(resolve_clicked_mi_board_names(items, struct)).toStrictEqual([])
    })

    test('__root__ 単体でも何も開かない', () => {
        expect(resolve_clicked_mi_board_names(['__root__'], struct)).toStrictEqual([])
    })

    test('リーフのクリックはその板を開く', () => {
        expect(resolve_clicked_mi_board_names(['Inbox'], struct)).toStrictEqual(['Inbox'])
    })

    test('「すべて」はツリーの実ノードなので開ける', () => {
        expect(resolve_clicked_mi_board_names([MI_ALL_BOARD_KEY], struct)).toStrictEqual([MI_ALL_BOARD_KEY])
    })

    test('フォルダ行のクリックでも何も開かない', () => {
        const nested = make_root(['Inbox'])
        nested.children?.push(make_node('フォルダ', [make_node('板A'), make_node('板B')]))
        expect(resolve_clicked_mi_board_names(['フォルダ', '板A', '板B'], nested)).toStrictEqual([])
    })

    test('ツリーにまだ無い板でもリーフのクリックなら開く', () => {
        // append_not_found_mi_boards() が拾う前の板を黙って握り潰さない
        expect(resolve_clicked_mi_board_names(['作ったばかりの板'], struct))
            .toStrictEqual(['作ったばかりの板'])
    })

    test('空文字は開かない', () => {
        expect(resolve_clicked_mi_board_names([''], struct)).toStrictEqual([])
    })

    test('struct が null でも __root__ は開かない', () => {
        // 設定の読み込み前。ルートキーはツリーを見なくても板ではないと分かる
        expect(resolve_clicked_mi_board_names(['__root__'], null)).toStrictEqual([])
        expect(resolve_clicked_mi_board_names(['Inbox'], null)).toStrictEqual(['Inbox'])
    })

    test('空の入力は空のまま', () => {
        expect(resolve_clicked_mi_board_names([], struct)).toStrictEqual([])
    })
})

describe('MI_ALL_BOARD_KEY', () => {
    test('ロケール非依存のハードコード値であること', () => {
        // append_all_mi_board() が作るノードの key/board_name と同じでなければならない。
        // i18n の訳語にすると日本語以外のロケールで「すべて」が効かなくなる
        expect(MI_ALL_BOARD_KEY).toBe('すべて')
    })
})
