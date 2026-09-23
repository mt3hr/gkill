/**
 * 構造ツリー（useFoldableStruct）のドロップ位置の検証。
 *
 * 以前は drop だけが offsetY（ポインタの下にある一番内側の要素からの距離）と固定の行の高さ 24px で
 * 前/中/後を決めていて、ドラッグ中の見た目の変化も無かった。いまは dragover で線を出し、
 * drop でも同じ関数（見出しの行の矩形 + clientY）で決めるので、見えている線と実際の挿入位置が一致する。
 */
import { afterEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/classes/use-device-kind', async () => {
    const { shallowRef, computed } = await import('vue')
    return {
        useDeviceKind: () => ({
            device_kind: computed(() => 'pc'),
            is_pc: shallowRef(true),
            is_tablet: computed(() => false),
            is_smart_phone: computed(() => false),
            has_touch: shallowRef(false),
        }),
    }
})

import { useFoldableStruct } from '@/classes/use-foldable-struct'
import { DropTypeFoldableStruct } from '@/classes/api/drop-type-foldable-struct'
import { DROP_INDICATOR_CLASS, end_drag } from '@/classes/drag-drop-indicator'
import type { FoldableStructProps } from '@/pages/views/foldable-struct-props'
import type { FoldableStructEmits } from '@/pages/views/foldable-struct-emits'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

afterEach(() => {
    end_drag()
    document.body.innerHTML = ''
})

function make_struct_obj(id: string, is_folder: boolean): FoldableStructModel {
    return {
        name: id,
        id: id,
        children: is_folder ? [] : null,
        key: id,
        is_checked: false,
        indeterminate: false,
        is_dir: is_folder,
    }
}

function create_view(struct_obj: FoldableStructModel, is_root: boolean = false) {
    const emitted = new Array<{ event: string, args: Array<unknown> }>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event, args })
    }) as unknown as FoldableStructEmits
    const props = {
        struct_obj,
        folder_name: '',
        is_open: true,
        is_editable: true,
        is_show_checkbox: false,
        is_root,
    } as unknown as FoldableStructProps
    return { view: useFoldableStruct({ props, emits }), emitted }
}

/** 見出し（top=100, 高さ30）と、その下に子孫の行を抱えた高い tr を作る */
function make_row(): HTMLElement {
    const tr = document.createElement('tr')
    const td = document.createElement('td')
    const header = document.createElement('table')
    header.className = 'foldable_struct_header'
    header.getBoundingClientRect = () => ({ top: 100, height: 30 }) as DOMRect
    td.appendChild(header)
    tr.appendChild(td)
    // 開いたフォルダの tr は子孫まで含んだ高さになる。tr 全体で測ると境界がずれる
    tr.getBoundingClientRect = () => ({ top: 100, height: 300 }) as DOMRect
    document.body.appendChild(tr)
    return tr
}

function make_event(target: HTMLElement, client_y: number, dragged: FoldableStructModel): DragEvent {
    return {
        currentTarget: target,
        clientY: client_y,
        // わざと矛盾する値。offsetY では判定しない
        offsetY: 999,
        dataTransfer: {
            types: ['gkill_struct_obj_json'],
            dropEffect: 'none',
            getData: (type: string) => (type === 'gkill_struct_obj_json' ? JSON.stringify(dragged) : ''),
            setData: vi.fn(),
        },
        preventDefault: vi.fn(),
        stopPropagation: vi.fn(),
    } as unknown as DragEvent
}

describe('フォルダへのドロップ位置', () => {
    test.each([
        { y: 105, want: DropTypeFoldableStruct.up_element, css: DROP_INDICATOR_CLASS.before },
        { y: 115, want: DropTypeFoldableStruct.in_folder_bottom, css: DROP_INDICATOR_CLASS.inside },
        { y: 125, want: DropTypeFoldableStruct.down_element, css: DROP_INDICATOR_CLASS.after },
    ])('clientY=$y → 線と drop の判定が一致する', ({ y, want, css }) => {
        const { view, emitted } = create_view(make_struct_obj('folder', true))
        const tr = make_row()
        const dragged = make_struct_obj('dragged', false)

        view.dragover(make_event(tr, y, dragged))
        expect(tr.classList.contains(css), '線の位置').toBe(true)

        view.drop(make_event(tr, y, dragged))
        const moved = emitted.filter(e => e.event === 'requested_move_struct_obj')
        expect(moved.length).toBe(1)
        expect(moved[0].args[2]).toBe(want)
        // ドロップしたら線は残らない
        expect(tr.className).toBe('')
    })
})

describe('項目へのドロップ位置', () => {
    test.each([
        { y: 110, want: DropTypeFoldableStruct.up_element },
        { y: 120, want: DropTypeFoldableStruct.down_element },
    ])('clientY=$y → $want（項目は中へ入れない）', ({ y, want }) => {
        const { view, emitted } = create_view(make_struct_obj('item', false))
        const tr = make_row()
        view.drop(make_event(tr, y, make_struct_obj('dragged', false)))
        expect(emitted.find(e => e.event === 'requested_move_struct_obj')?.args[2]).toBe(want)
    })
})

describe('線を出さない場所', () => {
    test('ルートの行（ドロップしても受け手が無い）', () => {
        const { view } = create_view(make_struct_obj('root', true), true)
        const tr = make_row()
        view.dragover(make_event(tr, 115, make_struct_obj('dragged', false)))
        expect(tr.className).toBe('')
    })

    test('掴んでいる行そのもの', () => {
        const struct_obj = make_struct_obj('self', true)
        const { view } = create_view(struct_obj)
        const tr = make_row()
        view.drag_start(make_event(tr, 115, struct_obj))
        view.dragover(make_event(tr, 115, struct_obj))
        expect([...tr.classList].some(c => c.startsWith('gkill-drop-'))).toBe(false)
    })

    test('この並べ替えのものでないドラッグ（ファイル等）', () => {
        const { view } = create_view(make_struct_obj('folder', true))
        const tr = make_row()
        const event = make_event(tr, 115, make_struct_obj('dragged', false))
        ;(event.dataTransfer as unknown as { types: Array<string> }).types = ['Files']
        view.dragover(event)
        expect(tr.className).toBe('')
    })
})
