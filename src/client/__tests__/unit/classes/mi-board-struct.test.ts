/**
 * 板ツリーの純関数2つ（存在判定・追加）の検証。
 *
 * 板はサーバに独立したエンティティとして存在せず、
 * 「その名前のタスクが1件でもあること」から導出される概念なので、
 * 「その板名は既知か？」を答えられるのはこのツリーだけになる。
 * board_exists_in_mi_board_struct が深い階層を見落とすと、
 * 既存の板へ入れているのに「新しい板です」の確認が出続ける。
 * append_mi_board_to_struct の冪等性が落ちると、
 * 確認を通すたびに同じ板がツリーへ何本も生える。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// append_mi_board_to_struct は追加ノードのidに generate_uuid() を使う。
// 呼び出しごとに違う値を返させて「新しいノードが作られたか」を見分けられるようにする
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
import { append_mi_board_to_struct, board_exists_in_mi_board_struct } from '@/classes/mi-board-struct'

function make_node(board_name: string, children: Array<MiBoardStructElementData> | null = null): MiBoardStructElementData {
    const node = new MiBoardStructElementData()
    node.key = board_name
    node.name = board_name
    node.board_name = board_name
    node.children = children
    node.is_dir = children !== null
    return node
}

// ルート / 直下の子 / 深い子 の3階層を持つツリー
function make_struct(): MiBoardStructElementData {
    return make_node('__root__', [
        make_node('すべて'),
        make_node('フォルダ', [
            make_node('仕事'),
            make_node('入れ子フォルダ', [
                make_node('趣味'),
            ]),
        ]),
    ])
}

describe('board_exists_in_mi_board_struct', () => {
    test('ルート自身の板名に一致すれば true', () => {
        expect(board_exists_in_mi_board_struct('__root__', make_struct())).toBe(true)
    })

    test('ルート直下の子に一致すれば true', () => {
        expect(board_exists_in_mi_board_struct('すべて', make_struct())).toBe(true)
    })

    test('深い階層の子に一致すれば true（再帰していないと取りこぼす）', () => {
        const struct = make_struct()
        expect(board_exists_in_mi_board_struct('仕事', struct), '2段目の子を見落としている').toBe(true)
        expect(board_exists_in_mi_board_struct('趣味', struct), '3段目の子を見落としている').toBe(true)
    })

    test('どこにも無ければ false', () => {
        expect(board_exists_in_mi_board_struct('知らない板', make_struct())).toBe(false)
    })

    test('children が null の葉で止まらない', () => {
        const leaf = make_node('葉')
        expect(leaf.children).toBeNull()
        expect(board_exists_in_mi_board_struct('葉', leaf)).toBe(true)
        expect(board_exists_in_mi_board_struct('知らない板', leaf)).toBe(false)
    })

    test('板名は部分一致ではなく完全一致で見る', () => {
        // 「仕事」があるからといって「仕事メモ」を既存扱いしてはいけない
        expect(board_exists_in_mi_board_struct('仕事メモ', make_struct())).toBe(false)
        expect(board_exists_in_mi_board_struct('事', make_struct())).toBe(false)
    })
})

describe('append_mi_board_to_struct', () => {
    test('未知の板はルート直下へ、初期チェックありで足される', () => {
        const struct = make_struct()
        append_mi_board_to_struct('新しい板', struct)

        expect(struct.children).toHaveLength(3)
        const appended = struct.children?.[2] as MiBoardStructElementData
        expect(appended.key).toBe('新しい板')
        expect(appended.name).toBe('新しい板')
        expect(appended.board_name).toBe('新しい板')
        expect(appended.check_when_inited, '初期チェックが無いと追加直後の板が mi 画面の列に出ない').toBe(true)
        expect(appended.is_checked).toBe(true)
        expect(appended.id, 'idに generate_uuid() が入っていない').toMatch(/^generated-uuid-\d+$/)
        expect(appended.children, '葉として足す（フォルダにはしない）').toBeNull()
        expect(board_exists_in_mi_board_struct('新しい板', struct)).toBe(true)
    })

    test('既にある板名なら何もしない（冪等）', () => {
        const struct = make_struct()
        append_mi_board_to_struct('新しい板', struct)
        const appended_id = struct.children?.[2].id

        append_mi_board_to_struct('新しい板', struct)
        append_mi_board_to_struct('新しい板', struct)

        expect(struct.children, '同じ板名が重ねて足されている').toHaveLength(3)
        expect(struct.children?.[2].id, '既存ノードが作り直されている').toBe(appended_id)
    })

    test('深い階層にある板名も既存として扱う（ルート直下に複製しない）', () => {
        const struct = make_struct()
        append_mi_board_to_struct('趣味', struct)
        expect(struct.children, '深い階層の板をルート直下へ複製している').toHaveLength(2)
    })

    test('空文字は足さない', () => {
        const struct = make_struct()
        append_mi_board_to_struct('', struct)
        expect(struct.children).toHaveLength(2)
    })

    test('children が null のノードへ足すときは配列を作ってから push する', () => {
        const leaf = make_node('葉')
        expect(leaf.children).toBeNull()

        append_mi_board_to_struct('新しい板', leaf)

        expect(leaf.children).toHaveLength(1)
        expect(leaf.children?.[0].board_name).toBe('新しい板')
    })
})
