'use strict'

import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

// フォルダ選択ダイアログに表示する移動先候補。id が null のときはルート直下を表す。
export interface MoveTargetFolderCandidate {
    id: string | null
    name: string
    depth: number
}

// root の子孫から id を持つ要素の親要素と children 内 index を探す。見つからなければ null。
export function find_parent_and_index(root: FoldableStructModel, id: string): { parent: FoldableStructModel, index: number } | null {
    if (!root.children) {
        return null
    }
    for (let i = 0; i < root.children.length; i++) {
        const child = root.children[i]
        if (child.id === id) {
            return { parent: root, index: i }
        }
        const found = find_parent_and_index(child, id)
        if (found) {
            return found
        }
    }
    return null
}

// parent の子孫(parent 自身は含まない)に id を持つ要素が含まれるか。循環防止判定用。
export function contains_struct(parent: FoldableStructModel, id: string): boolean {
    if (!parent.children) {
        return false
    }
    for (const child of parent.children) {
        if (child.id === id) {
            return true
        }
        if (contains_struct(child, id)) {
            return true
        }
    }
    return false
}

// 同一親内で1つ上の要素と入れ替える。先頭または見つからなければ何もせず false。
export function move_struct_up(root: FoldableStructModel, id: string): boolean {
    const found = find_parent_and_index(root, id)
    if (!found || found.index <= 0 || !found.parent.children) {
        return false
    }
    const children = found.parent.children
    children.splice(found.index - 1, 2, children[found.index], children[found.index - 1])
    return true
}

// 同一親内で1つ下の要素と入れ替える。末尾または見つからなければ何もせず false。
export function move_struct_down(root: FoldableStructModel, id: string): boolean {
    const found = find_parent_and_index(root, id)
    if (!found || !found.parent.children || found.index >= found.parent.children.length - 1) {
        return false
    }
    const children = found.parent.children
    children.splice(found.index, 2, children[found.index + 1], children[found.index])
    return true
}

// id の要素を target_folder_id(null はルート)の children 末尾へ移動する。
// 移動できない場合はツリーを変更せず false を返す。
export function move_struct_to_folder(root: FoldableStructModel, id: string, target_folder_id: string | null): boolean {
    if (id === target_folder_id) {
        return false
    }
    const found = find_parent_and_index(root, id)
    if (!found || !found.parent.children) {
        return false
    }
    const moving_struct = found.parent.children[found.index]

    let target: FoldableStructModel | null = null
    if (target_folder_id === null) {
        target = root
    } else {
        const target_found = find_parent_and_index(root, target_folder_id)
        if (!target_found || !target_found.parent.children) {
            return false
        }
        target = target_found.parent.children[target_found.index]
        if (!target.is_dir) {
            return false
        }
        // 自分の子孫フォルダへは移動できない(循環防止)
        if (contains_struct(moving_struct, target_folder_id)) {
            return false
        }
    }

    found.parent.children.splice(found.index, 1)
    if (!target.children) {
        target.children = []
    }
    target.children.push(moving_struct)
    return true
}

// 移動先候補フォルダをルート + is_dir の全フォルダから DFS 順・depth 付きで列挙する。
// 移動対象自身とその子孫フォルダは除外する。
export function list_move_target_folders(root: FoldableStructModel, moving_struct_id: string): Array<MoveTargetFolderCandidate> {
    const candidates: Array<MoveTargetFolderCandidate> = []
    const walk = (struct: FoldableStructModel, depth: number): void => {
        if (!struct.children) {
            return
        }
        for (const child of struct.children) {
            if (child.id === moving_struct_id) {
                continue
            }
            if (child.is_dir) {
                candidates.push({ id: child.id, name: child.name, depth: depth })
                walk(child, depth + 1)
            }
        }
    }
    walk(root, 0)
    return candidates
}
