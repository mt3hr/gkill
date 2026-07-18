import { describe, test, expect, beforeEach } from 'vitest'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'
import {
    find_parent_and_index,
    contains_struct,
    move_struct_up,
    move_struct_down,
    move_struct_to_folder,
    list_move_target_folders,
} from '@/classes/foldable-struct-move'

function make_node(id: string, is_dir: boolean, children: Array<FoldableStructModel> | null = null): FoldableStructModel {
    return {
        name: id,
        id: id,
        children: children,
        key: id,
        is_checked: false,
        indeterminate: false,
        is_dir: is_dir,
    }
}

// ルート
// ├ item1
// ├ folder_a
// │ ├ item2
// │ └ folder_b
// │   └ item3
// └ item4
function make_tree(): FoldableStructModel {
    return make_node('root', true, [
        make_node('item1', false),
        make_node('folder_a', true, [
            make_node('item2', false),
            make_node('folder_b', true, [
                make_node('item3', false),
            ]),
        ]),
        make_node('item4', false),
    ])
}

function ids_of(node: FoldableStructModel): Array<string | null> {
    return node.children ? node.children.map(child => child.id) : []
}

describe('foldable-struct-move', () => {
    let root: FoldableStructModel

    beforeEach(() => {
        root = make_tree()
    })

    describe('find_parent_and_index', () => {
        test('ルート直下の要素を見つける', () => {
            const found = find_parent_and_index(root, 'item4')
            expect(found).not.toBeNull()
            expect(found?.parent.id).toBe('root')
            expect(found?.index).toBe(2)
        })
        test('ネストした要素を見つける', () => {
            const found = find_parent_and_index(root, 'item3')
            expect(found?.parent.id).toBe('folder_b')
            expect(found?.index).toBe(0)
        })
        test('存在しないidはnull', () => {
            expect(find_parent_and_index(root, 'nothing')).toBeNull()
        })
    })

    describe('contains_struct', () => {
        test('子孫を含む', () => {
            const folder_a = root.children![1]
            expect(contains_struct(folder_a, 'item3')).toBe(true)
        })
        test('自分自身は含まない', () => {
            const folder_a = root.children![1]
            expect(contains_struct(folder_a, 'folder_a')).toBe(false)
        })
        test('無関係な要素は含まない', () => {
            const folder_a = root.children![1]
            expect(contains_struct(folder_a, 'item4')).toBe(false)
        })
    })

    describe('move_struct_up', () => {
        test('中間要素は1つ上と入れ替わる', () => {
            expect(move_struct_up(root, 'folder_a')).toBe(true)
            expect(ids_of(root)).toEqual(['folder_a', 'item1', 'item4'])
        })
        test('先頭要素は何もしない', () => {
            expect(move_struct_up(root, 'item1')).toBe(false)
            expect(ids_of(root)).toEqual(['item1', 'folder_a', 'item4'])
        })
        test('ネスト内でも入れ替わる', () => {
            expect(move_struct_up(root, 'folder_b')).toBe(true)
            expect(ids_of(root.children![1])).toEqual(['folder_b', 'item2'])
        })
        test('存在しないidは何もしない', () => {
            expect(move_struct_up(root, 'nothing')).toBe(false)
        })
    })

    describe('move_struct_down', () => {
        test('中間要素は1つ下と入れ替わる', () => {
            expect(move_struct_down(root, 'folder_a')).toBe(true)
            expect(ids_of(root)).toEqual(['item1', 'item4', 'folder_a'])
        })
        test('末尾要素は何もしない', () => {
            expect(move_struct_down(root, 'item4')).toBe(false)
            expect(ids_of(root)).toEqual(['item1', 'folder_a', 'item4'])
        })
        test('ネスト内でも入れ替わる', () => {
            expect(move_struct_down(root, 'item2')).toBe(true)
            expect(ids_of(root.children![1])).toEqual(['folder_b', 'item2'])
        })
    })

    describe('move_struct_to_folder', () => {
        test('フォルダ末尾へ移動する', () => {
            expect(move_struct_to_folder(root, 'item1', 'folder_a')).toBe(true)
            expect(ids_of(root)).toEqual(['folder_a', 'item4'])
            expect(ids_of(root.children![0])).toEqual(['item2', 'folder_b', 'item1'])
        })
        test('ルート(null)へ移動する', () => {
            expect(move_struct_to_folder(root, 'item3', null)).toBe(true)
            expect(ids_of(root)).toEqual(['item1', 'folder_a', 'item4', 'item3'])
            const folder_b = root.children![1].children![1]
            expect(ids_of(folder_b)).toEqual([])
        })
        test('childrenがnullのフォルダへも移動できる', () => {
            const folder_c = make_node('folder_c', true, null)
            root.children!.push(folder_c)
            expect(move_struct_to_folder(root, 'item1', 'folder_c')).toBe(true)
            expect(ids_of(folder_c)).toEqual(['item1'])
        })
        test('自分自身への移動は拒否してツリー無傷', () => {
            const before = JSON.stringify(root)
            expect(move_struct_to_folder(root, 'folder_a', 'folder_a')).toBe(false)
            expect(JSON.stringify(root)).toBe(before)
        })
        test('自分の子孫フォルダへの移動は拒否してツリー無傷', () => {
            const before = JSON.stringify(root)
            expect(move_struct_to_folder(root, 'folder_a', 'folder_b')).toBe(false)
            expect(JSON.stringify(root)).toBe(before)
        })
        test('非フォルダへの移動は拒否してツリー無傷', () => {
            const before = JSON.stringify(root)
            expect(move_struct_to_folder(root, 'item1', 'item4')).toBe(false)
            expect(JSON.stringify(root)).toBe(before)
        })
        test('存在しない移動先は拒否してツリー無傷', () => {
            const before = JSON.stringify(root)
            expect(move_struct_to_folder(root, 'item1', 'nothing')).toBe(false)
            expect(JSON.stringify(root)).toBe(before)
        })
        test('存在しない移動対象は拒否', () => {
            expect(move_struct_to_folder(root, 'nothing', 'folder_a')).toBe(false)
        })
    })

    describe('list_move_target_folders', () => {
        test('全フォルダをDFS順・depth付きで列挙する', () => {
            const candidates = list_move_target_folders(root, 'item1')
            expect(candidates).toEqual([
                { id: 'folder_a', name: 'folder_a', depth: 0 },
                { id: 'folder_b', name: 'folder_b', depth: 1 },
            ])
        })
        test('移動対象自身とその子孫フォルダは除外する', () => {
            const candidates = list_move_target_folders(root, 'folder_a')
            expect(candidates).toEqual([])
        })
        test('非フォルダ要素は列挙しない', () => {
            const candidates = list_move_target_folders(root, 'item3')
            expect(candidates.map(candidate => candidate.id)).toEqual(['folder_a', 'folder_b'])
        })
    })
})
