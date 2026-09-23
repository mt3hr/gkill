/**
 * 集計ビューの編集画面で「列」（集計項目を縦に並べる箱）を足し引きできることと、
 * 編集画面ではダブルクリックで記録の一覧を開かないことの検証。
 *
 * 列の関数が無かったころは、2列目以降を作るには設定の JSON を直接書くしかなかった。
 * 編集画面は記録を0件で読み込むので、ダブルクリックで開く一覧は常に空だった。
 */
import { describe, expect, test, vi } from 'vitest'
import { ref, toRaw } from 'vue'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { useDnoteItemTableView } from '@/classes/use-dnote-item-table-view'
import { useDnoteItemView } from '@/classes/use-dnote-item-view'
import { useAggregatedListItem } from '@/classes/use-aggregated-list-item'
import type DnoteItem from '@/classes/dnote/dnote-item'
import type DnoteItemTableViewProps from '@/pages/views/dnote-item-table-view-props'
import type DnoteItemTableViewEmits from '@/pages/views/dnote-item-table-view-emits'
import type DnoteItemProps from '@/pages/views/dnote-item-props'
import type DnoteItemViewEmits from '@/pages/views/dnote-item-view-emits'
import type AggregatedListItemProps from '@/pages/views/aggregated-list-item-props'
import type AggregatedListItemViewEmits from '@/pages/views/aggregated-list-item-view-emits'

function item(id: string): DnoteItem {
    return { id } as unknown as DnoteItem
}

function make_table(editable: boolean, columns: Array<Array<DnoteItem>>) {
    const model_value = ref(columns)
    const view = useDnoteItemTableView({
        props: { editable } as unknown as DnoteItemTableViewProps,
        emits: vi.fn() as unknown as DnoteItemTableViewEmits,
        model_value,
    })
    return { view, model_value }
}

describe('集計ビューの列の追加・削除', () => {
    test('add_column で末尾に空の列が増える（配列はその場で書き換える）', () => {
        const columns = [[item('a')]]
        const { view, model_value } = make_table(true, columns)
        view.add_column()
        expect(model_value.value.length).toBe(2)
        expect(model_value.value[1]).toEqual([])
        // 定義の items そのものを書き換える（「適用」がそのまま拾う）
        expect(toRaw(model_value.value)).toBe(columns)
    })

    test('delete_column は列を中の項目ごと消す', () => {
        const { view, model_value } = make_table(true, [[item('a')], [item('b'), item('c')], []])
        view.delete_column(1)
        expect(model_value.value.map(column => column.map(x => x.id))).toEqual([['a'], []])
    })

    test('最後の1列は消えない', () => {
        const { view, model_value } = make_table(true, [[item('a')]])
        expect(view.can_delete_column.value).toBe(false)
        view.delete_column(0)
        expect(model_value.value.length).toBe(1)
    })

    test('範囲外の添字は無視する', () => {
        const { view, model_value } = make_table(true, [[item('a')], [item('b')]])
        view.delete_column(2)
        view.delete_column(-1)
        expect(model_value.value.length).toBe(2)
    })

    test('閲覧画面（editable=false）では何もしない', () => {
        const { view, model_value } = make_table(false, [[item('a')], [item('b')]])
        view.add_column()
        view.delete_column(0)
        expect(model_value.value.map(column => column.map(x => x.id))).toEqual([['a'], ['b']])
    })
})

describe('集計項目のダブルクリック', () => {
    function make_item_view(editable: boolean) {
        const view = useDnoteItemView({
            props: { editable, dnd_list_index: 0 } as unknown as DnoteItemProps,
            emits: vi.fn() as unknown as DnoteItemViewEmits,
            model_value: ref(item('a')),
        })
        const list_dialog = { show: vi.fn() }
        const edit_dialog = { show: vi.fn() }
        view.kyou_list_view_dialog.value = list_dialog
        view.edit_dnote_item_dialog.value = edit_dialog
        return { view, list_dialog, edit_dialog }
    }

    test('閲覧画面では集計に使った記録の一覧を開く', () => {
        const { view, list_dialog, edit_dialog } = make_item_view(false)
        view.onDblclick()
        expect(list_dialog.show).toHaveBeenCalledTimes(1)
        expect(edit_dialog.show).not.toHaveBeenCalled()
    })

    test('編集画面では一覧を開かず、項目の編集ダイアログを開く', () => {
        const { view, list_dialog, edit_dialog } = make_item_view(true)
        view.onDblclick()
        expect(list_dialog.show).not.toHaveBeenCalled()
        expect(edit_dialog.show).toHaveBeenCalledTimes(1)
    })
})

describe('集計リストの行のダブルクリック', () => {
    function make_row(editable: boolean) {
        const view = useAggregatedListItem({
            props: {
                editable,
                aggregated_item: { value: '1' },
                dnote_list_query: { aggregate_target: { to_json: () => ({ type: 'AgregateCountKyou' }) } },
            } as unknown as AggregatedListItemProps,
            emits: vi.fn() as unknown as AggregatedListItemViewEmits,
        })
        const list_dialog = { show: vi.fn() }
        view.kyou_list_view_dialog.value = list_dialog
        return { view, list_dialog }
    }

    test('閲覧画面では一覧を開く', () => {
        const { view, list_dialog } = make_row(false)
        view.onDblclick()
        expect(list_dialog.show).toHaveBeenCalledTimes(1)
    })

    test('編集画面では開かない', () => {
        const { view, list_dialog } = make_row(true)
        view.onDblclick()
        expect(list_dialog.show).not.toHaveBeenCalled()
    })
})

// 編集画面の並べ替え。線（挿入位置の表示）と実際に入る位置は同じ判定で決まる
describe('集計項目の並べ替えの挿入位置', () => {
    function make_element(): HTMLElement {
        const el = document.createElement('div')
        el.getBoundingClientRect = () => ({ top: 0, height: 40 }) as DOMRect
        document.body.appendChild(el)
        return el
    }
    function make_event(target: HTMLElement, client_y: number): DragEvent {
        return {
            currentTarget: target,
            clientY: client_y,
            dataTransfer: {
                types: ['gkill_dnote_item_id', 'gkill_dnote_item_src_list_index'],
                dropEffect: 'none',
                getData: (type: string) => (type === 'gkill_dnote_item_id' ? 'b' : type === 'gkill_dnote_item_src_list_index' ? '0' : ''),
            },
            preventDefault: vi.fn(),
            stopPropagation: vi.fn(),
        } as unknown as DragEvent
    }

    test.each([
        { y: 10, css: 'gkill-drop-before', want: 'up' },
        { y: 30, css: 'gkill-drop-after', want: 'down' },
    ])('項目の上 clientY=$y → 線 $css・移動 $want', ({ y, css, want }) => {
        const emits = vi.fn()
        const view = useDnoteItemView({
            props: { editable: true, dnd_list_index: 1 } as unknown as DnoteItemProps,
            emits: emits as unknown as DnoteItemViewEmits,
            model_value: ref(item('a')),
        })
        const el = make_element()

        view.dragover(make_event(el, y))
        expect(el.classList.contains(css)).toBe(true)

        view.drop(make_event(el, y))
        const moved = emits.mock.calls.filter(call => call[0] === 'requested_move_dnote_item')
        expect(moved.map(call => call[5])).toEqual([want])
        expect(el.className).toBe('')
        el.remove()
    })

    test('列の空き（td）に落とすと上半分は先頭・下半分は末尾へ入る', () => {
        const { view, model_value } = make_table(true, [[item('b')], [item('x'), item('y')]])
        const el = make_element()

        view.onCellDragover(make_event(el, 30))
        expect(el.classList.contains('gkill-drop-after')).toBe(true)
        view.onCellDrop(make_event(el, 30), 1)

        expect(model_value.value.map(column => column.map(x => x.id))).toEqual([[], ['x', 'y', 'b']])
        expect(el.className).toBe('')
        el.remove()
    })
})
