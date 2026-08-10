'use strict'

import { CheckState } from '@/pages/views/check-state'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

// チェック対象キーの集合をツリーへ1回の走査で適用する。
// pre_uncheck_all が真なら、対象外ノードのチェックも同じ走査の中で外す。
// 以前は項目1つごとにツリー全体を再帰走査する O(項目数×ノード数) の実装が
// rep/tag/timeis の各クエリコンポーザブルに複製されていて、リポジトリやタグが
// 数百ある環境では列フォーカス切替のたびに数秒メインスレッドを塞いでいた。
export function apply_check_state_to_struct(
    root: FoldableStructModel,
    items: Array<string>,
    is_checked: CheckState,
    pre_uncheck_all: boolean,
): void {
    const item_keys = new Set(items)
    const walk = (struct: FoldableStructModel): void => {
        if (item_keys.has(struct.key)) {
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
        } else if (pre_uncheck_all) {
            struct.is_checked = false
            struct.indeterminate = false
        }
        if (struct.children) {
            struct.children.forEach(child => walk(child))
        }
    }
    walk(root)
}
