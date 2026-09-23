/**
 * ドラッグ＆ドロップの挿入位置の表示（classes/drag-drop-indicator.ts）の検証。
 *
 * 判定（decide_drop_position）は線の位置と実際の挿入位置の両方に使うので、境界を表で固定する。
 * 線は常に1本で、ドロップ・取り消しのどちらでも残らないこと。
 */
import { afterEach, describe, expect, test, vi } from 'vitest'
import {
    DRAG_SOURCE_CLASS,
    DROP_INDICATOR_CLASS,
    begin_drag_source,
    decide_drop_position,
    end_drag,
    has_drag_type,
    hide_drop_indicator,
    is_inside_drag_source,
    is_leaving_element,
    show_drop_indicator,
} from '@/classes/drag-drop-indicator'

afterEach(() => {
    end_drag()
    document.body.innerHTML = ''
    vi.useRealTimers()
})

describe('decide_drop_position', () => {
    const rect = { top: 100, height: 30 }
    test.each([
        { y: 90, inside: true, want: 'before' },
        { y: 100, inside: true, want: 'before' },
        { y: 110, inside: true, want: 'before' },
        { y: 110.5, inside: true, want: 'inside' },
        { y: 120, inside: true, want: 'inside' },
        { y: 120.5, inside: true, want: 'after' },
        { y: 130, inside: true, want: 'after' },
        { y: 200, inside: true, want: 'after' },
        { y: 115, inside: false, want: 'before' },
        { y: 115.5, inside: false, want: 'after' },
        { y: 50, inside: false, want: 'before' },
        { y: 150, inside: false, want: 'after' },
    ])('y=$y 中を受ける=$inside → $want', ({ y, inside, want }) => {
        expect(decide_drop_position(rect, y, inside)).toBe(want)
    })

    test('高さ0でも決まる', () => {
        expect(decide_drop_position({ top: 10, height: 0 }, 10, false)).toBe('before')
        expect(decide_drop_position({ top: 10, height: 0 }, 11, false)).toBe('after')
    })
})

function make_element(): HTMLElement {
    const el = document.createElement('div')
    document.body.appendChild(el)
    return el
}

describe('線の付け外し', () => {
    test('線は常に1本（別の要素へ出すと前の線は消える）', () => {
        const first = make_element()
        const second = make_element()
        show_drop_indicator(first, 'before')
        expect(first.classList.contains(DROP_INDICATOR_CLASS.before)).toBe(true)

        show_drop_indicator(second, 'after')
        expect(first.className).toBe('')
        expect(second.classList.contains(DROP_INDICATOR_CLASS.after)).toBe(true)
    })

    test('同じ要素で位置が変わると付け替わる', () => {
        const el = make_element()
        show_drop_indicator(el, 'before')
        show_drop_indicator(el, 'inside')
        expect(el.classList.contains(DROP_INDICATOR_CLASS.before)).toBe(false)
        expect(el.classList.contains(DROP_INDICATOR_CLASS.inside)).toBe(true)
    })

    test('hide に別の要素を渡しても、いまの線は消さない（子へ移ったあとの親の dragleave）', () => {
        const parent = make_element()
        const child = make_element()
        show_drop_indicator(child, 'after')
        hide_drop_indicator(parent)
        expect(child.classList.contains(DROP_INDICATOR_CLASS.after)).toBe(true)
        hide_drop_indicator(child)
        expect(child.className).toBe('')
    })

    test('掴んだ元は次のタスクで半透明になり、end_drag で線ごと戻る', () => {
        vi.useFakeTimers()
        const source = make_element()
        const target = make_element()
        begin_drag_source(source)
        // ドラッグ中の像は dragstart の見た目から作られるので、すぐには付けない
        expect(source.classList.contains(DRAG_SOURCE_CLASS)).toBe(false)
        vi.runAllTimers()
        expect(source.classList.contains(DRAG_SOURCE_CLASS)).toBe(true)
        show_drop_indicator(target, 'before')

        end_drag()

        expect(source.classList.contains(DRAG_SOURCE_CLASS)).toBe(false)
        expect(target.className).toBe('')
    })

    test('掴んだ元とその中には線を出さない判定', () => {
        const source = make_element()
        const inner = document.createElement('div')
        source.appendChild(inner)
        const other = make_element()
        begin_drag_source(source)
        expect(is_inside_drag_source(source)).toBe(true)
        expect(is_inside_drag_source(inner)).toBe(true)
        expect(is_inside_drag_source(other)).toBe(false)
        end_drag()
        expect(is_inside_drag_source(source)).toBe(false)
    })

    test('window に dragend / drop が届くと線が消える（取り消し・仕組みの外へのドロップ）', () => {
        const el = make_element()
        show_drop_indicator(el, 'after')
        window.dispatchEvent(new Event('dragend'))
        expect(el.className).toBe('')

        show_drop_indicator(el, 'before')
        window.dispatchEvent(new Event('drop'))
        expect(el.className).toBe('')
    })
})

describe('イベントの判定', () => {
    test('is_leaving_element は子へ移っただけなら偽', () => {
        const parent = make_element()
        const child = document.createElement('span')
        parent.appendChild(child)
        const sibling = make_element()
        const event = (related: EventTarget | null) => ({ currentTarget: parent, relatedTarget: related }) as unknown as DragEvent
        expect(is_leaving_element(event(child))).toBe(false)
        expect(is_leaving_element(event(sibling))).toBe(true)
        expect(is_leaving_element(event(null))).toBe(true)
    })

    test('has_drag_type は dataTransfer.types で見る', () => {
        const event = (types: Array<string> | undefined) => ({ dataTransfer: types ? { types } : null }) as unknown as DragEvent
        expect(has_drag_type(event(['gkill_struct_obj_json']), 'gkill_struct_obj_json')).toBe(true)
        expect(has_drag_type(event(['text/plain']), 'gkill_struct_obj_json')).toBe(false)
        expect(has_drag_type(event(undefined), 'gkill_struct_obj_json')).toBe(false)
    })
})
