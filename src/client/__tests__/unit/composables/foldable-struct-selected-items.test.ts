/**
 * FoldableStruct.get_selected_items() が「入れ物」を返さないことの検証。
 *
 * ここが返す key は、そのまま検索条件（tags / reps / devices_in_sidebar /
 * rep_types_in_sidebar / timeis_tags）として流れる。ルートは folder_name='' の
 * 空白帯としてクリックでき（`.tree_item { min-width: 200px }`）、踏むと
 * click_group_by_user が **自分自身の key（`__root__`）を含めて** 全部の key を
 * 上げてくるので、素通しすると実在しない名前が条件に紛れ込む。
 * OR検索では無害だが、AND検索（tags_and 等）では必ず0件になる。
 *
 * フォルダも同じで、実運用の TAG_STRUCT ではフォルダの大半が同名のタグを持たない
 * 純粋な入れ物になっている。
 */
import { describe, expect, test } from 'vitest'

import { useFoldableStruct } from '@/classes/use-foldable-struct'
import { CheckState } from '@/pages/views/check-state'
import { FOLDABLE_STRUCT_ROOT_KEY, is_struct_container_node, type FoldableStructModel } from '@/pages/views/foldable-struct-model'
import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'
import type { FoldableStructProps } from '@/pages/views/foldable-struct-props'
import type { FoldableStructEmits } from '@/pages/views/foldable-struct-emits'

function make_leaf(key: string, is_checked = false): FoldableStructModel {
    return {
        name: key,
        id: 'id-'.concat(key),
        children: null,
        key: key,
        is_checked: is_checked,
        indeterminate: false,
        is_dir: false,
    }
}

function make_folder(key: string, children: Array<FoldableStructModel>, is_checked = false): FoldableStructModel {
    return {
        name: key,
        id: 'id-'.concat(key),
        children: children,
        key: key,
        is_checked: is_checked,
        indeterminate: false,
        is_dir: true,
    }
}

// 保存済みの TAG_STRUCT と同じ形のルート
function make_root(children: Array<FoldableStructModel>, is_checked = false): FoldableStructModel {
    const root = make_folder(FOLDABLE_STRUCT_ROOT_KEY, children, is_checked)
    root.name = FOLDABLE_STRUCT_ROOT_KEY
    return root
}

function create_view(struct_obj: FoldableStructModel) {
    const emits = (() => { }) as unknown as FoldableStructEmits
    const props = {
        struct_obj: struct_obj,
        folder_name: '',
        is_open: true,
        is_editable: false,
        is_show_checkbox: true,
        is_root: true,
    } as unknown as FoldableStructProps
    return useFoldableStruct({ props: props, emits: emits })
}

describe('is_struct_container_node', () => {
    test('ルートは入れ物', () => {
        expect(is_struct_container_node(make_root([]))).toBe(true)
    })

    test('フォルダは入れ物', () => {
        expect(is_struct_container_node(make_folder('フォルダ', []))).toBe(true)
    })

    test('葉は入れ物ではない', () => {
        expect(is_struct_container_node(make_leaf('タグA'))).toBe(false)
    })

    test('is_dir が付いていなくてもルートキーなら入れ物', () => {
        // 保存済みJSONのルートに is_dir が無い実例があり、そのときルートは
        // 葉として描かれて name/key の __root__ がそのまま条件に入る
        const legacy_root = make_leaf(FOLDABLE_STRUCT_ROOT_KEY)
        expect(is_struct_container_node(legacy_root)).toBe(true)
    })
})

describe('useFoldableStruct get_selected_items', () => {
    test('チェックの入った葉だけを返す', () => {
        const view = create_view(make_root([make_leaf('a', true), make_leaf('b'), make_leaf('c', true)]))
        expect(view.get_selected_items()).toStrictEqual(['a', 'c'])
    })

    test('ルートがチェックされていても __root__ は返さない', () => {
        // ルート行の空白帯をクリックした状態
        const view = create_view(make_root([make_leaf('a', true)], true))
        const items = view.get_selected_items()
        expect(items).not.toContain(FOLDABLE_STRUCT_ROOT_KEY)
        expect(items).toStrictEqual(['a'])
    })

    test('フォルダがチェックされていてもフォルダ名は返さない', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA', true), make_leaf('タグB', true)], true),
            make_leaf('タグC', true),
        ]))
        const items = view.get_selected_items()
        expect(items).not.toContain('フォルダ')
        expect(items).toStrictEqual(['タグA', 'タグB', 'タグC'])
    })

    test('入れ物しかチェックされていなければ空を返す', () => {
        const view = create_view(make_root([make_folder('フォルダ', [make_leaf('タグA')], true)], true))
        expect(view.get_selected_items()).toStrictEqual([])
    })

    // フォルダ行のチェックは click/change_group_by_user が自分の key も載せてくるが、
    // apply_check_state_to_struct は key 一致でツリー全体を走査するので、
    // 同名の葉が別の場所にあってもそちらにチェックが入る。条件は落ちない
    test('フォルダ名と同名のタグが実在する場合、そのタグは条件に残る', () => {
        const root = make_root([
            make_folder('フォルダ', [make_leaf('タグA')]),
            make_leaf('フォルダ'),
        ])
        const view = create_view(root)

        // フォルダ行のクリック相当: 自分の key + 配下の key
        apply_check_state_to_struct(root, ['フォルダ', 'タグA'], CheckState.checked, true)

        const items = view.get_selected_items()
        // 入れ物としての「フォルダ」は落ちるが、葉の「フォルダ」は残る（重複もしない）
        expect(items).toStrictEqual(['タグA', 'フォルダ'])
    })

    test('入れ子のフォルダでも葉だけを拾う', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [
                make_folder('入れ子フォルダ', [make_leaf('タグA', true)], true),
            ], true),
        ]))
        expect(view.get_selected_items()).toStrictEqual(['タグA'])
    })
})
