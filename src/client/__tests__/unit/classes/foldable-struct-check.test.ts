/**
 * apply_check_state_to_struct のテスト。
 *
 * このヘルパーは、rep/tag/timeis の各クエリコンポーザブルに複製されていた
 * 「項目1つごとにツリー全体を再帰走査する O(項目数×ノード数) のチェック適用」を
 * 単一パス O(ノード数) に置き換えたもの。リポジトリやタグが数百ある環境で
 * 列フォーカス切替のたびに数秒メインスレッドを塞いでいたフリーズの修正なので、
 * 旧実装（項目ごと全走査）と同じ結果になることを等価比較で担保する。
 */
import { describe, expect, it } from 'vitest'

import { apply_check_state_to_struct } from '@/classes/foldable-struct-check'
import { CheckState } from '@/pages/views/check-state'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

function node(key: string, options?: Partial<FoldableStructModel>): FoldableStructModel {
    return {
        name: key,
        id: null,
        children: null,
        key,
        is_checked: false,
        indeterminate: false,
        is_dir: false,
        ...options,
    }
}

function make_tree(): FoldableStructModel {
    return node('__root__', {
        is_dir: true,
        children: [
            node('kmemo_laptop_2024', { is_checked: true }),
            node('urlog_laptop_2024', { indeterminate: true }),
            node('folder', {
                is_dir: true,
                is_checked: true,
                children: [
                    node('kmemo_phone_2024'),
                    // 重複キー: 別の場所に同じキーのノードがある
                    node('kmemo_laptop_2024'),
                    node('deep_folder', {
                        is_dir: true,
                        children: [node('mi_laptop_2024', { is_checked: true })],
                    }),
                ],
            }),
        ],
    })
}

// 置き換え前の実装（項目1つごとにツリー全体を走査）。等価比較の基準。
function reference_impl(
    root: FoldableStructModel,
    items: Array<string>,
    is_checked: CheckState,
    pre_uncheck_all: boolean,
): void {
    if (pre_uncheck_all) {
        const uncheck = (struct: FoldableStructModel): void => {
            struct.is_checked = false
            struct.indeterminate = false
            struct.children?.forEach(child => uncheck(child))
        }
        uncheck(root)
    }
    for (const key_name of items) {
        const apply = (struct: FoldableStructModel): void => {
            if (struct.key === key_name) {
                switch (is_checked) {
                    case CheckState.checked:
                        struct.is_checked = true
                        struct.indeterminate = false
                        break
                    case CheckState.unchecked:
                        struct.is_checked = false
                        struct.indeterminate = false
                        break
                    case CheckState.indeterminate:
                        struct.is_checked = false
                        struct.indeterminate = true
                        break
                }
            }
            struct.children?.forEach(child => apply(child))
        }
        apply(root)
    }
}

describe('apply_check_state_to_struct', () => {
    it('重複キーを含む入れ子ツリーで旧実装と同じ結果になる', () => {
        const item_patterns = [
            [],
            ['kmemo_laptop_2024'],
            ['kmemo_laptop_2024', 'mi_laptop_2024', 'folder'],
            ['not_exist'],
            ['__root__'],
        ]
        for (const items of item_patterns) {
            for (const state of [CheckState.checked, CheckState.unchecked, CheckState.indeterminate]) {
                for (const pre_uncheck_all of [true, false]) {
                    const expected = make_tree()
                    reference_impl(expected, items, state, pre_uncheck_all)
                    const actual = make_tree()
                    apply_check_state_to_struct(actual, items, state, pre_uncheck_all)
                    expect(actual, `items=${JSON.stringify(items)} state=${state} pre_uncheck_all=${pre_uncheck_all}`).toEqual(expected)
                }
            }
        }
    })

    it('pre_uncheck_all=true で対象外ノードのチェックと indeterminate が外れる', () => {
        const tree = make_tree()
        apply_check_state_to_struct(tree, ['kmemo_phone_2024'], CheckState.checked, true)

        expect(tree.children![0].is_checked).toBe(false)
        expect(tree.children![1].indeterminate).toBe(false)
        expect(tree.children![2].is_checked).toBe(false)
        expect(tree.children![2].children![0].is_checked).toBe(true)
    })

    it('pre_uncheck_all=false は対象ノードだけを書き換える', () => {
        const tree = make_tree()
        apply_check_state_to_struct(tree, ['kmemo_phone_2024'], CheckState.checked, false)

        expect(tree.children![0].is_checked).toBe(true)
        expect(tree.children![1].indeterminate).toBe(true)
        expect(tree.children![2].children![0].is_checked).toBe(true)
    })

    it('重複キーのノードは両方に適用される', () => {
        const tree = make_tree()
        apply_check_state_to_struct(tree, ['kmemo_laptop_2024'], CheckState.checked, true)

        expect(tree.children![0].is_checked).toBe(true)
        expect(tree.children![2].children![1].is_checked).toBe(true)
    })

    it('CheckState.indeterminate は is_checked を外して indeterminate を立てる', () => {
        const tree = make_tree()
        apply_check_state_to_struct(tree, ['kmemo_laptop_2024'], CheckState.indeterminate, false)

        expect(tree.children![0].is_checked).toBe(false)
        expect(tree.children![0].indeterminate).toBe(true)
    })
})
