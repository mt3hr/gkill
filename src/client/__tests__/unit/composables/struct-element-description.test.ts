/**
 * 設定ツリー（タグ / 記録保管場所 / 記録種別 / プロファイル / 板 / メモ帳テンプレート）の要素に
 * 利用者が書く「説明」（description。MCP へ渡す運用メモ）が、ダイアログの往復で落ちないことの検証。
 *
 * 要素編集ダイアログの apply() は既存ノードを in-place で直さず、`new XxxStructElementData()` に
 * 既知のフィールドだけを詰め直して id 一致で splice 差し替えする。ここに description を写し忘れると、
 * 「適用」のたびに説明がエラーも警告も出ずに消える（ADR-0632）。読み込み・clone・D&D・保存は
 * JSON を丸ごと往復させるので落とさない ―― 落ちる唯一の箇所がこの詰め直しで、型では検出できない。
 *
 * 板（MiBoard）は要素編集ダイアログ自体が新設で、update_mi_board_struct の splice と
 * 「ルートは開かない」（walk は子しか差し替えないので、開けても適用が消える）をここで固定する。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { reactive } from 'vue'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { useEditTagStructElementView } from '@/classes/use-edit-tag-struct-element-view'
import { useEditRepStructElementView } from '@/classes/use-edit-rep-struct-element-view'
import { useEditRepTypeStructElementView } from '@/classes/use-edit-rep-type-struct-element-view'
import { useEditDeviceStructElementView } from '@/classes/use-edit-device-struct-element-view'
import { useEditMiBoardStructElementView } from '@/classes/use-edit-mi-board-struct-element-view'
import { useEditKFTLTemplateStructElementView } from '@/classes/use-edit-kftl-template-struct-element-view'
import { useAddNewTagStructElementView } from '@/classes/use-add-new-tag-struct-element-view'
import { useAddNewRepStructElementView } from '@/classes/use-add-new-rep-struct-element-view'
import { useAddNewRepTypeStructElementView } from '@/classes/use-add-new-rep-type-struct-element-view'
import { useAddNewDeviceStructElementView } from '@/classes/use-add-new-device-struct-element-view'
import { useAddNewKftlTemplateStructElementView } from '@/classes/use-add-new-kftl-template-struct-element-view'
import { useEditMiBoardStructView } from '@/classes/use-edit-mi-board-struct-view'
import type { EditMiBoardStructViewProps } from '@/pages/views/edit-mi-board-struct-view-props'
import type { EditMiBoardStructViewEmits } from '@/pages/views/edit-mi-board-struct-view-emits'

type Emitted = Array<{ event: string, payload: unknown }>

function make_emits(emitted: Emitted) {
    return ((event: string, payload: unknown) => {
        emitted.push({ event: event, payload: payload })
    })
}

function emitted_payload(emitted: Emitted, event: string): Record<string, unknown> {
    const found = emitted.find(e => e.event === event)
    if (!found) {
        throw new Error(`${event} が emit されていない: ${emitted.map(e => e.event).join(', ')}`)
    }
    return found.payload as Record<string, unknown>
}

// ── 要素編集: apply() が description を写すこと ──

interface EditCase {
    name: string
    identity_key: string
    make: () => Record<string, unknown>
    use: (struct_obj: unknown, emits: unknown) => { description: { value: string }, apply: () => Promise<void> }
    event: string
}

const edit_cases: Array<EditCase> = [
    {
        name: 'タグ',
        identity_key: 'tag_name',
        make: () => Object.assign(new TagStructElementData(), { id: 'id-tag', tag_name: 'tagA', name: 'tagA', check_when_inited: true, is_force_hide: true, description: '古い説明' }),
        use: (struct_obj, emits) => useEditTagStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_tag_struct',
    },
    {
        name: '記録保管場所',
        identity_key: 'rep_name',
        make: () => Object.assign(new RepStructElementData(), { id: 'id-rep', rep_name: 'Kmemo_pc', name: 'Kmemo_pc', check_when_inited: true, ignore_check_rep_rykv: true, description: '古い説明' }),
        use: (struct_obj, emits) => useEditRepStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_rep_struct',
    },
    {
        name: '記録種別',
        identity_key: 'rep_type_name',
        make: () => Object.assign(new RepTypeStructElementData(), { id: 'id-rep-type', rep_type_name: 'kmemo', name: 'kmemo', check_when_inited: true, description: '古い説明' }),
        use: (struct_obj, emits) => useEditRepTypeStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_rep_type_struct',
    },
    {
        name: 'プロファイル',
        identity_key: 'device_name',
        make: () => Object.assign(new DeviceStructElementData(), { id: 'id-device', device_name: 'pc', name: 'pc', check_when_inited: true, description: '古い説明' }),
        use: (struct_obj, emits) => useEditDeviceStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_device_struct',
    },
    {
        name: '板',
        identity_key: 'board_name',
        make: () => Object.assign(new MiBoardStructElementData(), { id: 'id-board', board_name: 'Inbox', name: 'Inbox', check_when_inited: true, description: '古い説明' }),
        use: (struct_obj, emits) => useEditMiBoardStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_mi_board_struct',
    },
    {
        name: 'メモ帳テンプレート',
        identity_key: 'title',
        make: () => Object.assign(new KFTLTemplateElementData(), { id: 'id-template', title: '日記', name: '日記', template: 'ーにっき\n', description: '古い説明' }),
        use: (struct_obj, emits) => useEditKFTLTemplateStructElementView({ props: { struct_obj } as never, emits: emits as never }),
        event: 'requested_update_kftl_template_struct',
    },
]

describe.each(edit_cases)('要素編集 $name: apply() は description を詰め直す', ({ identity_key, make, use, event }) => {
    test('編集した説明が emit されるノードに載る', async () => {
        const struct_obj = make()
        const emitted: Emitted = []
        const view = use(struct_obj, make_emits(emitted))

        expect(view.description.value, '開いた時点の値が元ノードの説明でない').toBe('古い説明')
        view.description.value = '新しい説明'
        await view.apply()

        const updated = emitted_payload(emitted, event)
        expect(updated.description, 'apply() が description を写していない').toBe('新しい説明')
        expect(updated.id).toBe(struct_obj.id)
        expect(updated[identity_key], '識別欄が失われている').toBe(struct_obj[identity_key])
    })

    test('説明欄の無い古い保存データは空文字として開く', async () => {
        const struct_obj = make()
        delete struct_obj.description
        const emitted: Emitted = []
        const view = use(struct_obj, make_emits(emitted))

        expect(view.description.value).toBe('')
        await view.apply()
        expect(emitted_payload(emitted, event).description).toBe('')
    })
})

// ── 要素追加: description を載せ、reset で空に戻すこと ──

interface AddCase {
    name: string
    use: (emits: unknown) => Record<string, unknown>
    fill: (view: Record<string, unknown>) => void
    submit: string
    reset: string
    event: string
}

const fake_api = { generate_uuid: () => 'uuid-new' }
const set_ref = (view: Record<string, unknown>, key: string, value: string) => { (view[key] as { value: string }).value = value }

const add_cases: Array<AddCase> = [
    {
        name: 'タグ',
        use: (emits) => useAddNewTagStructElementView({ props: { gkill_api: fake_api } as never, emits: emits as never }),
        fill: (view) => set_ref(view, 'tag_name', 'tagA'),
        submit: 'emits_tag_name', reset: 'reset_tag_name', event: 'requested_add_tag_struct_element',
    },
    {
        name: '記録保管場所',
        use: (emits) => useAddNewRepStructElementView({ props: { gkill_api: fake_api } as never, emits: emits as never }),
        fill: (view) => set_ref(view, 'rep_name', 'Kmemo_pc'),
        submit: 'emits_rep_name', reset: 'reset_rep_name', event: 'requested_add_rep_struct_element',
    },
    {
        name: '記録種別',
        use: (emits) => useAddNewRepTypeStructElementView({ props: { gkill_api: fake_api } as never, emits: emits as never }),
        fill: (view) => set_ref(view, 'rep_type_name', 'kmemo'),
        submit: 'emits_rep_type_name', reset: 'reset_rep_type_name', event: 'requested_add_rep_type_struct_element',
    },
    {
        name: 'プロファイル',
        use: (emits) => useAddNewDeviceStructElementView({ props: { gkill_api: fake_api } as never, emits: emits as never }),
        fill: (view) => set_ref(view, 'device_name', 'pc'),
        submit: 'emits_device_name', reset: 'reset_device_name', event: 'requested_add_device_struct_element',
    },
    {
        name: 'メモ帳テンプレート',
        use: (emits) => useAddNewKftlTemplateStructElementView({ props: { gkill_api: fake_api } as never, emits: emits as never }),
        fill: (view) => { set_ref(view, 'title', '日記'); set_ref(view, 'template', 'ーにっき\n') },
        submit: 'emits_kftl_template_name', reset: 'reset_kftl_template_name', event: 'requested_add_kftl_template_struct_element',
    },
]

describe.each(add_cases)('要素追加 $name: description を載せ、reset で空に戻す', ({ use, fill, submit, reset, event }) => {
    test('追加されるノードに説明が載る', () => {
        const emitted: Emitted = []
        const view = use(make_emits(emitted))
        fill(view)
        set_ref(view, 'description', '追加時の説明')

        ;(view[submit] as () => void)()

        expect(emitted_payload(emitted, event).description).toBe('追加時の説明')
    })

    test('reset で説明欄が空に戻る（ダイアログの show / hide で呼ばれる）', () => {
        const view = use(make_emits([]))
        set_ref(view, 'description', '残骸')

        ;(view[reset] as () => void)()

        expect((view.description as { value: string }).value).toBe('')
    })
})

// ── 板構造の編集: update_mi_board_struct の差し替えと、ルートは開かないこと ──

function make_board(id: string, children?: Array<MiBoardStructElementData>): MiBoardStructElementData {
    const board = new MiBoardStructElementData()
    board.id = id
    board.name = id
    board.board_name = id
    board.key = id
    board.children = children ?? null
    return board
}

// ApplicationConfig の実物は循環importを引き込むので、この画面が触るものだけの構造フェイクを使う
function make_fake_application_config(root: MiBoardStructElementData) {
    const build = (struct: MiBoardStructElementData): Record<string, unknown> => {
        const config: Record<string, unknown> = {
            mi_board_struct: struct,
            append_not_found_mi_boards: vi.fn().mockResolvedValue([]),
            append_all_mi_board: vi.fn().mockResolvedValue([]),
        }
        config.clone = () => build(JSON.parse(JSON.stringify(struct)) as MiBoardStructElementData)
        return config
    }
    return build(root)
}

function create_mi_board_view(root: MiBoardStructElementData) {
    const props = reactive({
        application_config: make_fake_application_config(root),
        gkill_api: {},
        mi_board_struct: root,
    }) as unknown as EditMiBoardStructViewProps
    const emitted: Emitted = []
    const view = useEditMiBoardStructView({ props: props, emits: make_emits(emitted) as unknown as EditMiBoardStructViewEmits })
    return { view, emitted }
}

describe('板構造の編集ダイアログ', () => {
    test('update_mi_board_struct は id 一致の板だけを差し替え、並びは動かない', () => {
        const { view } = create_mi_board_view(make_board('root', [make_board('Inbox'), make_board('Work'), make_board('Home')]))
        const updated = make_board('Work')
        updated.description = '仕事のタスク'

        view.update_mi_board_struct(updated)

        const children = view.cloned_application_config.value.mi_board_struct.children ?? []
        expect(children.map(c => c.id)).toEqual(['Inbox', 'Work', 'Home'])
        expect(children[1].description).toBe('仕事のタスク')
        expect(children[0].description, '無関係な板の説明が書き換わっている').toBe('')
    })

    test('show_edit_mi_board_struct_dialog は子の板ではダイアログを開き、ルートでは開かない', () => {
        const { view } = create_mi_board_view(make_board('root', [make_board('Inbox')]))
        const show = vi.fn()
        view.edit_mi_board_struct_element_dialog.value = { show } as never

        view.show_edit_mi_board_struct_dialog('root')
        expect(show, 'ルートを開くと適用しても差し替わらないので開いてはいけない').not.toHaveBeenCalled()

        view.show_edit_mi_board_struct_dialog('Inbox')
        expect(show).toHaveBeenCalledTimes(1)
        expect((show.mock.calls[0][0] as MiBoardStructElementData).id).toBe('Inbox')
    })
})
