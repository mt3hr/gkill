/**
 * plaing検索のカスタム条件ダイアログ（use-edit-plaing-time-is-dialog）の検証。
 *
 * 「検索条件をカスタマイズする」チェックの意味論は3状態で決まる:
 *   - OFF にしたら null（未設定＝従来どおりの既定動作）へ戻す
 *   - null から ON にしたら ApplicationConfig 由来の既定条件を生成する
 *   - 既に条件があるのに ON を押し直しても、その条件を潰さない
 * 3つ目を落とすと、チェックを触っただけで編集中の条件が消える。
 */
import { describe, expect, test, vi } from 'vitest'

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
import { useEditPlaingTimeIsDialog } from '@/classes/use-edit-plaing-time-is-dialog'
import type { EditPlaingTimeIsDialogProps } from '@/pages/dialogs/edit-plaing-time-is-dialog-props'
import type { EditPlaingTimeIsDialogEmits } from '@/pages/dialogs/edit-plaing-time-is-dialog-emits'

// ApplicationConfig の実物は循環importを引き込むので、既定条件生成が触る枝だけの構造フェイクを使う
function make_fake_application_config(): Record<string, unknown> {
    return {
        rep_struct: {
            rep_name: 'root',
            children: [{ rep_name: 'timeis_dev_202601', children: null }],
        },
        tag_struct: {
            tag_name: '',
            is_force_hide: false,
            check_when_inited: false,
            children: null,
        },
    }
}

function create_dialog() {
    const props = {
        application_config: make_fake_application_config(),
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        app_content_height: 800,
        app_content_width: 1200,
    } as unknown as EditPlaingTimeIsDialogProps
    const emits = (() => { }) as unknown as EditPlaingTimeIsDialogEmits
    return useEditPlaingTimeIsDialog({ props: props, emits: emits })
}

describe('show()', () => {
    test('引数なしなら未設定（null）から始まり、チェックはOFF', async () => {
        const view = create_dialog()
        expect(view.is_show_dialog.value).toBe(false)

        await view.show()

        expect(view.is_show_dialog.value).toBe(true)
        expect(view.current_query.value, '未設定は null で表す').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
        expect(view.editor_model.value, 'null だとエディタが描画されないので別持ちにしてある').not.toBeNull()
    })

    test('保存済みの条件を渡したらそれをそのまま編集対象にし、チェックはON', async () => {
        const view = create_dialog()
        const stored = new FindKyouQuery()
        stored.query_id = 'stored-id'
        stored.keywords = '保存済みの条件'

        await view.show(stored)

        // ref 越しに reactive proxy が被るので、同一性は toRaw で見る
        expect(toRaw(view.current_query.value), '渡した条件が編集対象になっていない').toBe(stored)
        expect(view.is_use_custom_find_kyou_query.value).toBe(true)
    })

    test('開き直しは前回の編集内容を引きずらない', async () => {
        const view = create_dialog()
        const stored = new FindKyouQuery()
        stored.query_id = 'stored-id'
        await view.show(stored)

        await view.show()

        expect(view.current_query.value, '未設定で開き直したのに前回の条件が残っている').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
    })
})

describe('is_use_custom_find_kyou_query の3状態', () => {
    test('OFF にすると null（既定動作）へ戻る', async () => {
        const view = create_dialog()
        const stored = new FindKyouQuery()
        stored.keywords = '保存済みの条件'
        await view.show(stored)

        view.is_use_custom_find_kyou_query.value = false

        expect(view.current_query.value, 'OFFは null でなければ「未設定」を表せない').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
    })

    test('null から ON にすると既定条件を生成する', async () => {
        const view = create_dialog()
        await view.show()

        view.is_use_custom_find_kyou_query.value = true

        const generated = view.current_query.value
        expect(generated, 'ONにしても条件が生成されない（エディタが空になる）').not.toBeNull()
        expect(generated?.tags, '未設定時のplaing検索と同じくタグフィルタ未使用').toBeNull()
        expect(generated?.reps, '既定は全rep').toEqual(['timeis_dev_202601'])
        expect(view.is_use_custom_find_kyou_query.value).toBe(true)
    })

    test('既に条件があるのに ON を押し直しても既存を潰さない', async () => {
        const view = create_dialog()
        const stored = new FindKyouQuery()
        stored.query_id = 'stored-id'
        stored.keywords = '保存済みの条件'
        await view.show(stored)

        view.is_use_custom_find_kyou_query.value = true

        expect(toRaw(view.current_query.value), 'ONを押し直しただけで編集中の条件が既定へ差し替わっている').toBe(stored)
        expect(view.current_query.value?.keywords).toBe('保存済みの条件')
    })
})
