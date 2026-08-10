/**
 * 「新しい板です」確認ゲートの検証。
 *
 * 板はサーバ側に実体が無く、板名の打ち間違いは無言で新しい板を生やす
 * （`usecase/mi.go` の GetMiBoardList が Mi と MiReKyou の board_name を集めるだけ）。
 * 唯一の防波堤がこのクライアント側の確認なので、
 * 「既存板なのに聞かれる」「未知の板なのに聞かれない」のどちらも致命的になる。
 *
 * 確認を通した板をその場で板ツリーへ足す remember_confirmed_mi_boards() も、
 * 落ちると同じ板へ2件目を入れるたびに確認が出続ける（体感では不具合そのもの）。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// append_mi_board_to_struct が追加ノードのidに generate_uuid() を使う
vi.mock('@/classes/api/gkill-api', () => {
    let uuid_serial = 0
    return {
        GkillAPI: {
            get_gkill_api: vi.fn(() => ({
                generate_uuid: vi.fn(() => `generated-uuid-${++uuid_serial}`),
            })),
            get_instance: vi.fn(() => ({
                get_session_id: vi.fn(() => 'mock-session'),
                generate_uuid: vi.fn(() => `generated-uuid-${++uuid_serial}`),
            })),
        },
    }
})

import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { useConfirmUnknownMiBoard } from '@/classes/use-confirm-unknown-mi-board'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { ComponentRef } from '@/classes/component-ref'

function make_node(board_name: string, children: Array<MiBoardStructElementData> | null = null): MiBoardStructElementData {
    const node = new MiBoardStructElementData()
    node.key = board_name
    node.name = board_name
    node.board_name = board_name
    node.children = children
    node.is_dir = children !== null
    return node
}

// ApplicationConfig の実物は req_res との循環importを引き込むため、
// コンポーザブルが触るフィールド（板ツリー）だけ持つ fake を使う
function create_view() {
    const mi_board_struct = make_node('__root__', [
        make_node('すべて'),
        make_node('フォルダ', [
            make_node('仕事'),
        ]),
    ])
    const application_config = { mi_board_struct: mi_board_struct }
    const view = useConfirmUnknownMiBoard({
        application_config: () => application_config as unknown as ApplicationConfig,
    })
    const show = vi.fn()
    view.confirm_unknown_mi_board_dialog.value = { show: show } as unknown as ComponentRef
    return { view, mi_board_struct, show }
}

describe('collect_unknown_mi_boards', () => {
    test('未知の板だけを入力順のまま返す', () => {
        const { view } = create_view()
        expect(view.collect_unknown_mi_boards(['板Z', '板A'])).toEqual(['板Z', '板A'])
    })

    test('同じ板名が何度出てきても1回だけ返す（確認を2回出さない）', () => {
        const { view } = create_view()
        expect(view.collect_unknown_mi_boards(['板A', '板A', '板B', '板A'])).toEqual(['板A', '板B'])
    })

    test('空文字は未知の板として数えない（既定の板へのフォールバックなので）', () => {
        const { view } = create_view()
        expect(view.collect_unknown_mi_boards(['', '板A', ''])).toEqual(['板A'])
    })

    test('板ツリーにある板は深い階層にあってもスキップする', () => {
        const { view } = create_view()
        expect(view.collect_unknown_mi_boards(['すべて', '仕事', '板A']), '既存板を未知扱いすると毎回確認が出る').toEqual(['板A'])
    })

    test('すべて既存なら空配列（確認を出さない）', () => {
        const { view } = create_view()
        expect(view.collect_unknown_mi_boards(['すべて', '仕事', ''])).toEqual([])
    })
})

describe('確認ダイアログの開閉', () => {
    test('open_confirm で対象が入り、ダイアログの show() が呼ばれる', () => {
        const { view, show } = create_view()
        expect(view.unknown_mi_boards.value).toEqual([])

        view.open_confirm(['板A', '板B'])

        expect(view.unknown_mi_boards.value).toEqual(['板A', '板B'])
        expect(show, 'show() を呼んでいないと確認が画面に出ない').toHaveBeenCalledTimes(1)
    })

    test('close_confirm で対象が空に戻る', () => {
        const { view } = create_view()
        view.open_confirm(['板A'])
        view.close_confirm()
        expect(view.unknown_mi_boards.value).toEqual([])
    })

    test('ダイアログrefが未設定でも open_confirm は例外を投げない', () => {
        const { view } = create_view()
        view.confirm_unknown_mi_board_dialog.value = null
        expect(() => view.open_confirm(['板A'])).not.toThrow()
        expect(view.unknown_mi_boards.value).toEqual(['板A'])
    })
})

describe('remember_confirmed_mi_boards', () => {
    test('確認を通した板は板ツリーへ入り、2件目では確認が出ない', () => {
        const { view, mi_board_struct } = create_view()
        const unknown = view.collect_unknown_mi_boards(['板A', '板B'])
        view.open_confirm(unknown)

        view.remember_confirmed_mi_boards()
        view.close_confirm()

        expect(mi_board_struct.children).toHaveLength(4)
        expect(view.collect_unknown_mi_boards(['板A', '板B']), '確認済みの板でまた確認が出る').toEqual([])
    })

    test('close_confirm より後に呼ぶと何も覚えない（呼ぶ順序が仕様）', () => {
        const { view, mi_board_struct } = create_view()
        view.open_confirm(view.collect_unknown_mi_boards(['板A']))

        view.close_confirm()
        view.remember_confirmed_mi_boards()

        expect(mi_board_struct.children).toHaveLength(2)
        expect(view.collect_unknown_mi_boards(['板A'])).toEqual(['板A'])
    })

    test('既存板が混じっていてもツリーは増えない（冪等）', () => {
        const { view, mi_board_struct } = create_view()
        view.unknown_mi_boards.value = ['すべて', '仕事', '板A', '板A']

        view.remember_confirmed_mi_boards()

        expect(mi_board_struct.children).toHaveLength(3)
        expect(mi_board_struct.children?.[2].board_name).toBe('板A')
    })
})
