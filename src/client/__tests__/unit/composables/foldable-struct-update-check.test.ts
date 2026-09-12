/**
 * FoldableStruct.update_check() がグループ行のチェック表示を**葉だけ**から導出することの検証。
 *
 * 入れ物（フォルダ・ルート）のモデル is_checked が true になるのは、フォルダ行を
 * 直接クリックした経路（change_group_by_user / click_group_by_user が自分の key を
 * 載せる）だけ。葉を1つずつチェックした経路や、クエリからの再同期
 * （get_selected_items() は入れ物を返さないので、pre_uncheck_all でフォルダは false に戻る）
 * では false のまま。旧実装はそれも数えていたので、フォルダ自身は葉から見て
 * チェック表示なのに、その親だけが「子が全部チェック済みなのに indeterminate」になっていた。
 */
import { describe, expect, test } from 'vitest'

import { useFoldableStruct } from '@/classes/use-foldable-struct'
import { CheckState } from '@/pages/views/check-state'
import { FOLDABLE_STRUCT_ROOT_KEY, type FoldableStructModel } from '@/pages/views/foldable-struct-model'
import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'
import type { FoldableStructProps } from '@/pages/views/foldable-struct-props'
import type { FoldableStructEmits } from '@/pages/views/foldable-struct-emits'

function make_leaf(key: string, is_checked = false, indeterminate = false): FoldableStructModel {
    return {
        name: key,
        id: 'id-'.concat(key),
        children: null,
        key: key,
        is_checked: is_checked,
        indeterminate: indeterminate,
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

describe('useFoldableStruct update_check', () => {
    test('フォルダのモデルが未チェックでも、配下の葉が全部チェック済みなら親はチェック表示（indeterminateではない）', () => {
        // 葉を1つずつチェックして全部チェックした状態。葉の key しか emit されないのでフォルダは false のまま
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA', true), make_leaf('タグB', true)]),
        ]))
        view.update_check()
        expect(view.check.value).toBe(true)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('入れ子のフォルダのモデルが全部未チェックでも、葉が全部チェック済みならルートはチェック表示', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [
                make_folder('入れ子フォルダ', [make_leaf('タグA', true), make_leaf('タグB', true)]),
                make_leaf('タグC', true),
            ]),
        ]))
        view.update_check()
        expect(view.check.value).toBe(true)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('クエリからの再同期（pre_uncheck_all）でフォルダが false に戻っても、葉が全部残ればルートはチェック表示', () => {
        // フォルダ行のクリックでフォルダ自身も true になっている状態から始める
        const root = make_root([
            make_folder('フォルダ', [make_leaf('タグA', true), make_leaf('タグB', true)], true),
        ], true)
        const view = create_view(root)

        // find_kyou_query の watch 相当: tags（= get_selected_items() の結果。入れ物を含まない）を
        // pre_uncheck_all=true で適用し直す
        apply_check_state_to_struct(root, ['タグA', 'タグB'], CheckState.checked, true)
        view.update_check()

        expect(root.children![0].is_checked).toBe(false)
        expect(view.check.value).toBe(true)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('フォルダのモデルが true でも、葉が全部未チェックなら親は未チェック（indeterminateではない）', () => {
        // フォルダ行のクリックでフォルダを true にしたあと、葉を1つずつ全部外した状態
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA'), make_leaf('タグB')], true),
        ]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('一部の葉だけチェック済みなら indeterminate', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA', true), make_leaf('タグB')]),
            make_leaf('タグC'),
        ]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(true)
    })

    test('どの葉も未チェックなら未チェック', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA'), make_leaf('タグB')]),
            make_leaf('タグC'),
        ]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('葉が0個のフォルダしか無ければ未チェック（空虚にチェック表示しない）', () => {
        const view = create_view(make_root([make_folder('空のフォルダ', [], true)]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('葉が1つも無いルートは未チェック', () => {
        const view = create_view(make_root([]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(false)
    })

    test('indeterminate な葉があれば親は indeterminate', () => {
        const view = create_view(make_root([
            make_folder('フォルダ', [make_leaf('タグA', true), make_leaf('タグB', false, true)]),
        ]))
        view.update_check()
        expect(view.check.value).toBe(false)
        expect(view.indeterminate_group.value).toBe(true)
    })
})
