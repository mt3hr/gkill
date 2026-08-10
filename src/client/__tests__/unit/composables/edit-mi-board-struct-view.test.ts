/**
 * Mi の板構造の編集（use-edit-mi-board-struct-view）の検証。
 *
 * delete_mi_board_struct の walk は「子で true が返ったら親が splice して false を返す」
 * という独特な形をしている。true を返すのは対象ノード自身、splice するのはその親、
 * という役割分担なので、返り値と添字のどちらを間違えても
 * 「隣の板が消える」「深い板が消えない」という形で壊れる。
 *
 * reload_cloned_application_config は append_all_mi_board を呼んではいけない。
 * 呼ぶと「すべて」が unshift で先頭に戻り、ダイアログを開き直すたびに並び順が巻き戻る。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { reactive, type Ref } from 'vue'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { useEditMiBoardStructView } from '@/classes/use-edit-mi-board-struct-view'
import type { EditMiBoardStructViewProps } from '@/pages/views/edit-mi-board-struct-view-props'
import type { EditMiBoardStructViewEmits } from '@/pages/views/edit-mi-board-struct-view-emits'

function make_board(id: string, children?: Array<MiBoardStructElementData>): MiBoardStructElementData {
    const board = new MiBoardStructElementData()
    board.id = id
    board.name = id
    board.board_name = id
    board.key = id
    board.children = children ?? null
    return board
}

// root
//  ├ folder ─ [deep_x, deep_target, deep_y]
//  ├ top_x
//  ├ top_target
//  └ top_y
function make_tree(): MiBoardStructElementData {
    return make_board('root', [
        make_board('folder', [make_board('deep_x'), make_board('deep_target'), make_board('deep_y')]),
        make_board('top_x'),
        make_board('top_target'),
        make_board('top_y'),
    ])
}

// ApplicationConfig の実物は循環importを引き込むので、この画面が触るものだけの構造フェイクを使う。
// clone() は本物と同じく mi_board_struct をJSONで深いコピーする
function make_fake_application_config(root: MiBoardStructElementData) {
    const spies = {
        append_not_found_mi_boards: vi.fn().mockResolvedValue([]),
        append_all_mi_board: vi.fn().mockResolvedValue([]),
        clone: vi.fn(),
    }
    const build = (struct: MiBoardStructElementData): Record<string, unknown> => {
        const config: Record<string, unknown> = {
            mi_board_struct: struct,
            append_not_found_mi_boards: spies.append_not_found_mi_boards,
            append_all_mi_board: spies.append_all_mi_board,
        }
        config.clone = () => {
            spies.clone()
            return build(JSON.parse(JSON.stringify(struct)) as MiBoardStructElementData)
        }
        return config
    }
    return { config: build(root), spies }
}

function create_view(root: MiBoardStructElementData) {
    const { config, spies } = make_fake_application_config(root)
    const props = reactive({
        application_config: config,
        gkill_api: {},
        mi_board_struct: root,
    }) as unknown as EditMiBoardStructViewProps
    const emitted: Array<{ event: string, payload: unknown }> = []
    const emits = ((event: string, payload: unknown) => {
        emitted.push({ event: event, payload: payload })
    }) as unknown as EditMiBoardStructViewEmits
    const view = useEditMiBoardStructView({ props: props, emits: emits })
    return { view, spies, emitted }
}

function ids_of(board: MiBoardStructElementData | null): Array<string | null> {
    return board?.children?.map((child) => child.id) ?? []
}

function child_at(board: MiBoardStructElementData, index: number): MiBoardStructElementData {
    const children = board.children
    if (!children) {
        throw new Error(`${board.id} に子が無い`)
    }
    return children[index]
}

describe('delete_mi_board_struct', () => {
    test('深い子を消しても、その兄弟と上位の並びは動かない', () => {
        const { view } = create_view(make_tree())

        view.delete_mi_board_struct('deep_target')

        const root = view.cloned_application_config.value.mi_board_struct
        expect(ids_of(child_at(root, 0)), '深い子の削除で兄弟までずれている').toEqual(['deep_x', 'deep_y'])
        expect(ids_of(root), '深い子の削除で上位の並びが動いている').toEqual(['folder', 'top_x', 'top_target', 'top_y'])
    })

    test('ルート直下を消しても、その兄弟と孫は動かない', () => {
        const { view } = create_view(make_tree())

        view.delete_mi_board_struct('top_target')

        const root = view.cloned_application_config.value.mi_board_struct
        expect(ids_of(root), 'ルート直下の削除で隣の板まで消えている').toEqual(['folder', 'top_x', 'top_y'])
        expect(ids_of(child_at(root, 0)), '無関係な子孫が消えている').toEqual(['deep_x', 'deep_target', 'deep_y'])
    })

    test('存在しないidでは何も消えない', () => {
        const { view } = create_view(make_tree())

        view.delete_mi_board_struct('not_exist')

        const root = view.cloned_application_config.value.mi_board_struct
        expect(ids_of(root)).toEqual(['folder', 'top_x', 'top_target', 'top_y'])
        expect(ids_of(child_at(root, 0))).toEqual(['deep_x', 'deep_target', 'deep_y'])
    })

    test('編集はクローンの中だけで起き、props の板構造は書き換わらない', () => {
        const root = make_tree()
        const { view } = create_view(root)

        view.delete_mi_board_struct('top_target')

        expect(ids_of(root), '適用前なのに元の板構造が書き換わっている').toEqual(['folder', 'top_x', 'top_target', 'top_y'])
    })
})

// find_mi_board_struct はコンポーザブルの返り値に含まれていない（内部関数）ので、
// 唯一の呼び出し元である show_confirm_delete_mi_board_struct_dialog 越しに検査する。
// 削除確認ダイアログへ渡す「対象ノード」がそのまま検索結果
describe('板の検索（show_confirm_delete_mi_board_struct_dialog 経由）', () => {
    function attach_confirm_dialog_stub(view: ReturnType<typeof useEditMiBoardStructView>) {
        const show = vi.fn()
        const dialog_ref = view.confirm_delete_mi_board_struct_dialog as unknown as Ref<unknown>
        dialog_ref.value = { show: show }
        return show
    }

    test('深い子をクローン上の実体で引く', () => {
        const { view } = create_view(make_tree())
        const show = attach_confirm_dialog_stub(view)

        view.show_confirm_delete_mi_board_struct_dialog('deep_target')

        const root = view.cloned_application_config.value.mi_board_struct
        const deep = child_at(child_at(root, 0), 1)
        expect(show, 'コピーを渡すと削除確認ダイアログの対象がずれる').toHaveBeenCalledWith(deep)
    })

    test('ルート自身も引ける', () => {
        const { view } = create_view(make_tree())
        const show = attach_confirm_dialog_stub(view)

        view.show_confirm_delete_mi_board_struct_dialog('root')

        expect(show).toHaveBeenCalledWith(view.cloned_application_config.value.mi_board_struct)
    })

    test('存在しないidでは確認ダイアログを開かない', () => {
        const { view } = create_view(make_tree())
        const show = attach_confirm_dialog_stub(view)

        view.show_confirm_delete_mi_board_struct_dialog('not_exist')

        expect(show, '対象が見つからないのに確認ダイアログを開いている').not.toHaveBeenCalled()
    })
})

describe('reload_cloned_application_config', () => {
    test('実在する板の補完はするが、「すべて」の付け直しはしない', async () => {
        const { view, spies } = create_view(make_tree())
        spies.append_not_found_mi_boards.mockClear()
        spies.append_all_mi_board.mockClear()

        await view.reload_cloned_application_config()

        expect(spies.append_not_found_mi_boards, '実在する板の補完が呼ばれていない').toHaveBeenCalledTimes(1)
        expect(
            spies.append_all_mi_board,
            '「すべて」を付け直している（unshiftされて並び順が先頭へ巻き戻る）',
        ).not.toHaveBeenCalled()
    })

    test('開き直すと未適用の編集は捨てられ、元の板構造から取り直す', async () => {
        const { view, spies } = create_view(make_tree())
        view.delete_mi_board_struct('top_target')
        expect(ids_of(view.cloned_application_config.value.mi_board_struct)).toEqual(['folder', 'top_x', 'top_y'])

        await view.reload_cloned_application_config()

        expect(spies.clone, '取り直しでクローンし直していない').toHaveBeenCalledTimes(2)
        expect(
            ids_of(view.cloned_application_config.value.mi_board_struct),
            '取り直しても前回の削除が残っている',
        ).toEqual(['folder', 'top_x', 'top_target', 'top_y'])
    })
})
