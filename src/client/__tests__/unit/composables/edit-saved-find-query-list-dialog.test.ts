/**
 * 保存済み検索条件の一覧編集ダイアログ（use-edit-saved-find-query-list-dialog）の検証。
 *
 * このダイアログは show() で受け取ったリストの「クローン」を編集し、
 * OKを押したときだけ親へ返す（キャンセルで丸ごと破棄できる）。
 * クローンを忘れると、キャンセルしたのに設定画面側のリストが書き換わる。
 *
 * 並べ替え・削除・エディタからの書き戻しは添字操作なので、
 * 端（先頭のup／末尾のdown）と範囲外の添字を固定しておく。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: vi.fn(),
    close_dialog_via_history: vi.fn(),
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn(() => ({})),
}))

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { toRaw } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'
import { useEditSavedFindQueryListDialog } from '@/classes/use-edit-saved-find-query-list-dialog'
import type { EditSavedFindQueryListDialogProps } from '@/pages/dialogs/edit-saved-find-query-list-dialog-props'
import type { EditSavedFindQueryListDialogEmits } from '@/pages/dialogs/edit-saved-find-query-list-dialog-emits'

// ApplicationConfig の実物は循環importを引き込むので、既定条件生成が触る枝だけの構造フェイクを使う。
// rykv既定はカレンダー期間を持ち、mi既定は持たない（この差が query_type の分岐の目印になる）
function make_fake_application_config(): Record<string, unknown> {
    return {
        rep_struct: {
            rep_name: 'root',
            children: [{ rep_name: 'kmemo_dev_202601', children: null }],
        },
        rep_type_struct: {
            rep_type_name: 'root',
            key: 'root',
            children: [{ rep_type_name: 'kmemo', key: 'kmemo', check_when_inited: true, children: null }],
        },
        device_struct: {
            device_name: 'root',
            key: 'root',
            children: [{ device_name: 'dev', key: 'dev', check_when_inited: true, children: null }],
        },
        tag_struct: {
            tag_name: '',
            is_force_hide: false,
            check_when_inited: false,
            children: [{ tag_name: '旅行', is_force_hide: false, check_when_inited: true, children: null }],
        },
        rykv_default_period: 30,
    }
}

let uuid_seq = 0

function create_dialog(query_type: 'rykv' | 'mi') {
    uuid_seq = 0
    const props = {
        application_config: make_fake_application_config(),
        gkill_api: { generate_uuid: () => `generated-uuid-${++uuid_seq}` },
        app_content_height: 800,
        app_content_width: 1200,
        query_type: query_type,
    } as unknown as EditSavedFindQueryListDialogProps
    const emitted: Array<{ event: string, items: Array<SavedFindQueryItem> }> = []
    const emits = ((event: string, items: Array<SavedFindQueryItem>) => {
        emitted.push({ event: event, items: items })
    }) as unknown as EditSavedFindQueryListDialogEmits
    return { view: useEditSavedFindQueryListDialog({ props: props, emits: emits }), emitted }
}

function make_item(id: string, title: string): SavedFindQueryItem {
    const find_kyou_query = new FindKyouQuery()
    find_kyou_query.query_id = `query-${id}`
    return { id: id, title: title, find_kyou_query: find_kyou_query }
}

function item_ids(items: Array<SavedFindQueryItem>): Array<string> {
    return items.map((item) => item.id)
}

describe('add_item', () => {
    test('rykv用の既定条件はMi向けではなく、期間（カレンダー）を持つ', async () => {
        const { view } = create_dialog('rykv')
        await view.show([])

        view.add_item()

        expect(view.editing_items.value).toHaveLength(1)
        const added = view.editing_items.value[0]
        expect(added.id).toBe('generated-uuid-1')
        expect(added.title).toBe('SAVED_FIND_QUERY_DEFAULT_NAME')
        expect(added.find_kyou_query.for_mi, 'rykv用なのにMi向けの既定条件になっている').toBe(false)
        expect(added.find_kyou_query.calendar_start_date, 'rykv既定期間がカレンダーへ反映されていない').not.toBeNull()
        expect(added.find_kyou_query.tags).toEqual(['旅行'])
    })

    test('mi用の既定条件はMi向けで、期間を持たない', async () => {
        const { view } = create_dialog('mi')
        await view.show([])

        view.add_item()

        const added = view.editing_items.value[0]
        expect(added.find_kyou_query.for_mi, 'mi用なのにMi向けの既定条件になっていない').toBe(true)
        expect(added.find_kyou_query.calendar_start_date, 'mi既定にカレンダー期間は無い').toBeNull()
        expect(added.find_kyou_query.reps, 'Mi既定は全repを対象にする').toEqual(['kmemo_dev_202601'])
    })
})

describe('move_item', () => {
    test('先頭を上へ・末尾を下へは何も起きない', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A'), make_item('b', 'B'), make_item('c', 'C')])

        view.move_item(0, 'up')
        expect(item_ids(view.editing_items.value), '先頭を上へで並びが壊れている').toEqual(['a', 'b', 'c'])

        view.move_item(2, 'down')
        expect(item_ids(view.editing_items.value), '末尾を下へで並びが壊れている').toEqual(['a', 'b', 'c'])
    })

    test('隣と入れ替える', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A'), make_item('b', 'B'), make_item('c', 'C')])

        view.move_item(0, 'down')
        expect(item_ids(view.editing_items.value)).toEqual(['b', 'a', 'c'])

        view.move_item(2, 'up')
        expect(item_ids(view.editing_items.value)).toEqual(['b', 'c', 'a'])
    })
})

describe('delete_item', () => {
    test('指定した1件だけ消える', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A'), make_item('b', 'B'), make_item('c', 'C')])

        view.delete_item(1)

        expect(item_ids(view.editing_items.value)).toEqual(['a', 'c'])
    })
})

describe('apply_edited_query', () => {
    test('編集中の行へ書き戻す', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A'), make_item('b', 'B')])
        view.current_editing_index.value = 1

        const applied = new FindKyouQuery()
        applied.query_id = 'applied'
        view.apply_edited_query(applied)

        // ref 越しに reactive proxy が被るので、同一性は toRaw で見る
        expect(toRaw(view.editing_items.value[1].find_kyou_query)).toBe(applied)
        expect(view.editing_items.value[0].find_kyou_query.query_id, '無関係な行まで書き換えている').toBe('query-a')
    })

    test('編集中の行が無い（-1）なら何もしない', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A')])
        expect(view.current_editing_index.value, 'show() は編集中の行をリセットする').toBe(-1)

        view.apply_edited_query(new FindKyouQuery())

        expect(view.editing_items.value[0].find_kyou_query.query_id, '編集対象が無いのに書き込んでいる').toBe('query-a')
    })

    test('範囲外の添字なら何もしない（落ちない）', async () => {
        const { view } = create_dialog('rykv')
        await view.show([make_item('a', 'A')])

        view.current_editing_index.value = 1
        view.apply_edited_query(new FindKyouQuery())
        expect(view.editing_items.value).toHaveLength(1)
        expect(view.editing_items.value[0].find_kyou_query.query_id).toBe('query-a')

        view.current_editing_index.value = -5
        view.apply_edited_query(new FindKyouQuery())
        expect(view.editing_items.value[0].find_kyou_query.query_id).toBe('query-a')
    })
})

describe('show() が編集するのはクローン', () => {
    test('ダイアログ側の編集は呼び出し元の配列へ漏れない', async () => {
        const { view } = create_dialog('rykv')
        const original_query = new FindKyouQuery()
        original_query.query_id = 'query-a'
        original_query.keywords = '元の条件'
        const original_items: Array<SavedFindQueryItem> = [
            { id: 'a', title: 'A', find_kyou_query: original_query },
        ]

        await view.show(original_items)

        view.editing_items.value[0].title = '書き換えた'
        view.editing_items.value[0].find_kyou_query.keywords = '書き換えた条件'
        view.add_item()
        view.delete_item(0)

        expect(original_items, 'キャンセルできるはずの編集が呼び出し元の配列に漏れている').toHaveLength(1)
        expect(original_items[0].title).toBe('A')
        expect(original_items[0].find_kyou_query.keywords, '検索条件がクローンされていない（参照を共有している）').toBe('元の条件')
        expect(original_items[0].find_kyou_query).toBe(original_query)
    })

    test('OKで返すのは編集中のクローン（親がここで初めて受け取る）', async () => {
        const { view, emitted } = create_dialog('rykv')
        const original_items = [make_item('a', 'A')]
        await view.show(original_items)
        view.editing_items.value[0].title = '書き換えた'

        view.onSave()

        expect(emitted).toHaveLength(1)
        expect(emitted[0].event).toBe('requested_apply_saved_find_querys')
        expect(emitted[0].items[0].title).toBe('書き換えた')
        expect(original_items[0].title).toBe('A')
    })
})
