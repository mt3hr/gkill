/**
 * useFoldableStruct の端末種別ゲートの検証。
 *
 * ドラッグ&ドロップの可否は `is_pc`、ロングプレスでのコンテキストメニュー補完は
 * `has_touch && !is_pc` と、見る値が違う。以前は「タッチできるか」の2値を
 * 両方に兼用していたため、タッチパネル搭載の Windows ノート（is_pc かつ has_touch）で
 * D&D が丸ごと死んでいた。その回帰をマトリクスで固定する。
 *
 * `useDeviceKind` はモジュールレベルのシングルトンなので、resetModules で作り直すより
 * mock で ref を注入するほうが決定論的（jsdom は matchMedia 未定義なので、
 * 実物を使うと必ず 'pc' + has_touch=false に倒れて2ケースしか書けない）。
 */
import { describe, expect, test, vi, beforeEach } from 'vitest'
import type { Ref } from 'vue'

// vi.mock はホイストされるので、注入する ref の入れ物も vi.hoisted で先に用意する
const device_mock = vi.hoisted(() => ({
    is_pc: null as Ref<boolean> | null,
    has_touch: null as Ref<boolean> | null,
}))

vi.mock('@/classes/use-device-kind', async () => {
    const { shallowRef, computed } = await import('vue')
    const is_pc = shallowRef(true)
    const has_touch = shallowRef(false)
    device_mock.is_pc = is_pc
    device_mock.has_touch = has_touch
    return {
        useDeviceKind: () => ({
            device_kind: computed(() => (is_pc.value ? 'pc' : 'tablet')),
            is_pc: is_pc,
            is_tablet: computed(() => !is_pc.value),
            is_smart_phone: computed(() => false),
            has_touch: has_touch,
        }),
    }
})

import { useFoldableStruct } from '@/classes/use-foldable-struct'
import type { FoldableStructProps } from '@/pages/views/foldable-struct-props'
import type { FoldableStructEmits } from '@/pages/views/foldable-struct-emits'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'

function set_device(kind: { is_pc: boolean, has_touch: boolean }): void {
    device_mock.is_pc!.value = kind.is_pc
    device_mock.has_touch!.value = kind.has_touch
}

function make_struct_obj(): FoldableStructModel {
    return {
        name: 'item',
        id: 'item-id',
        children: null,
        key: 'item',
        is_checked: false,
        indeterminate: false,
        is_dir: false,
    }
}

interface EmittedEvent {
    event: string
    args: Array<unknown>
}

function create_view(options: { is_editable: boolean }) {
    const emitted = new Array<EmittedEvent>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }) as unknown as FoldableStructEmits
    const props = {
        struct_obj: make_struct_obj(),
        folder_name: '',
        is_open: false,
        is_editable: options.is_editable,
        is_show_checkbox: true,
        is_root: true,
    } as unknown as FoldableStructProps
    const view = useFoldableStruct({ props: props, emits: emits })
    return { view: view, emitted: emitted, props: props }
}

// dragover / drag_start に渡す最低限の DragEvent 相当
function make_drag_event() {
    const data_transfer = { dropEffect: 'none', effectAllowed: 'none', setData: vi.fn(), getData: vi.fn(() => '') }
    const event = {
        dataTransfer: data_transfer,
        preventDefault: vi.fn(),
        stopPropagation: vi.fn(),
        currentTarget: null,
        offsetY: 0,
    }
    return { event: event as unknown as DragEvent, raw: event, data_transfer: data_transfer }
}

beforeEach(() => {
    set_device({ is_pc: true, has_touch: false })
})

describe('端末種別ごとのドラッグ可否', () => {
    test.each([
        { name: 'PC + 編集可', is_pc: true, has_touch: false, is_editable: true, expected: true },
        { name: 'PC + 編集不可', is_pc: true, has_touch: false, is_editable: false, expected: false },
        { name: 'タブレット(タッチのみ) + 編集可', is_pc: false, has_touch: true, is_editable: true, expected: false },
        // タッチパネル搭載Windowsノート。ここが false に倒れていたのが直した回帰
        { name: 'タッチPC + 編集可', is_pc: true, has_touch: true, is_editable: true, expected: true },
    ])('$name → effective_draggable=$expected', ({ is_pc, has_touch, is_editable, expected }) => {
        set_device({ is_pc: is_pc, has_touch: has_touch })
        const { view } = create_view({ is_editable: is_editable })
        expect(view.effective_draggable.value).toBe(expected)
    })

    test('ドラッグ不可の端末では drag_start が dataTransfer に何も積まない', () => {
        set_device({ is_pc: false, has_touch: true })
        const { view } = create_view({ is_editable: true })
        const { event, data_transfer } = make_drag_event()

        view.drag_start(event)

        expect(data_transfer.setData, 'ドラッグ不可なのにドラッグを開始している').not.toHaveBeenCalled()
    })

    test('タッチPCでは drag_start が struct_obj を積む', () => {
        set_device({ is_pc: true, has_touch: true })
        const { view } = create_view({ is_editable: true })
        const { event, data_transfer } = make_drag_event()

        view.drag_start(event)

        expect(data_transfer.setData, 'タッチPCでD&Dが死んでいる').toHaveBeenCalledTimes(1)
        expect(data_transfer.setData.mock.calls[0][0]).toBe('gkill_struct_obj_json')
    })
})

describe('ロングプレスでのコンテキストメニュー補完', () => {
    test('タブレット(has_touch かつ !is_pc)では有効', () => {
        set_device({ is_pc: false, has_touch: true })
        const { view, emitted } = create_view({ is_editable: true })

        view.onLongPressItem({} as unknown as PointerEvent)

        expect(emitted.map((e) => e.event), 'ロングプレスでメニューが出ない端末が生まれている').toContain('contextmenu_item')
    })

    // タッチPCで有効にすると、ネイティブcontextmenu(約500ms)とv-long-press(600ms)が
    // 二重発火してメニューが2回開く。右クリックがあるので無効でも機能は欠けない
    test('タッチPCでは無効', () => {
        set_device({ is_pc: true, has_touch: true })
        const { view, emitted } = create_view({ is_editable: true })

        view.onLongPressItem({} as unknown as PointerEvent)

        expect(emitted, 'タッチPCでロングプレスまで拾うとメニューが二重に開く').toHaveLength(0)
    })

    test('編集不可なら端末を問わず無効', () => {
        set_device({ is_pc: false, has_touch: true })
        const { view, emitted } = create_view({ is_editable: false })

        view.onLongPressItem({} as unknown as PointerEvent)

        expect(emitted).toHaveLength(0)
    })

    test('右クリック由来の contextmenu_item は端末に関係なく出る', () => {
        set_device({ is_pc: true, has_touch: false })
        const { view, emitted } = create_view({ is_editable: true })

        view.onContextmenuItem({} as unknown as MouseEvent)

        expect(emitted.map((e) => e.event)).toContain('contextmenu_item')
    })
})

describe('dragover のガード', () => {
    test('ドラッグ不可のときは preventDefault しない（全行がドロップ対象になるのを防ぐ）', () => {
        set_device({ is_pc: false, has_touch: true })
        const { view } = create_view({ is_editable: true })
        const { event, raw, data_transfer } = make_drag_event()

        view.dragover(event)

        expect(raw.preventDefault, 'ドラッグ不可なのにドロップを受け付けている').not.toHaveBeenCalled()
        expect(data_transfer.dropEffect).toBe('none')
    })

    test('ドラッグ可のときは dropEffect=move にして preventDefault する', () => {
        set_device({ is_pc: true, has_touch: true })
        const { view } = create_view({ is_editable: true })
        const { event, raw, data_transfer } = make_drag_event()

        view.dragover(event)

        expect(data_transfer.dropEffect).toBe('move')
        expect(raw.preventDefault, 'preventDefault しないとドロップできない').toHaveBeenCalledTimes(1)
    })

    test('編集不可のPCでも preventDefault しない', () => {
        set_device({ is_pc: true, has_touch: false })
        const { view } = create_view({ is_editable: false })
        const { event, raw } = make_drag_event()

        view.dragover(event)

        expect(raw.preventDefault).not.toHaveBeenCalled()
    })
})
