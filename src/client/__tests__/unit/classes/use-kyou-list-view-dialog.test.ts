/**
 * KyouListViewDialog のイベント処理のテスト。
 *
 * このダイアログは Dnote の項目/集計リストをダブルクリックしたときに開く。
 * 中身のKyouにタグを足しても表示が変わらない、というバグの原因が3つあった。
 *   1. `requested_reload_kyou` を自分で処理していなかった（素通しで親へ投げるだけ）
 *   2. タグ/テキスト/通知のCRUDを「このリストの内容を変えないので」と握りつぶしていた
 *   3. `requested_open_rykv_dialog` を親へ持ち上げていたので、そこから出る
 *      `requested_reload_kyou` がページのreload_kyouにしか届かなかった
 * どれも再発しやすいのでここで固定する。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/classes/kyou-reload', () => ({
    new_reload_batch: vi.fn(() => 0),
    refresh_kyou: vi.fn(),
    refresh_kyou_in_list: vi.fn(),
}))

import { ref, type Ref } from 'vue'
import { refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { kyou_dialog_relay_event_names } from '@/classes/kyou-view-relay'
import { useKyouListViewDialog } from '@/classes/use-kyou-list-view-dialog'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouListViewDialogProps } from '@/pages/dialogs/kyou-list-view-dialog-props'
import type { KyouListViewEmits } from '@/pages/views/kyou-list-view-emits'

const refresh_kyou_mock = refresh_kyou as unknown as ReturnType<typeof vi.fn>
const refresh_kyou_in_list_mock = refresh_kyou_in_list as unknown as ReturnType<typeof vi.fn>

function make_kyou(id: string): Kyou {
    const kyou = {
        id: id,
        data_type: 'kmemo_create',
        clone: () => kyou,
    }
    return kyou as unknown as Kyou
}

function build_dialog(options?: { host_rykv_dialogs?: boolean, kyous?: Array<Kyou> }) {
    const emits = vi.fn() as unknown as KyouListViewEmits & ReturnType<typeof vi.fn>
    const model_value: Ref<Array<Kyou> | undefined> = ref(options?.kyous ?? [])
    const props = {
        application_config: {},
        gkill_api: { generate_uuid: vi.fn(() => 'dialog-uuid') },
        highlight_targets: [],
        list_height: 400,
        enable_context_menu: true,
        enable_dialog: true,
        force_show_latest_kyou_info: true,
        show_rep_name: true,
        host_rykv_dialogs: options?.host_rykv_dialogs,
    } as unknown as KyouListViewDialogProps

    const dialog = useKyouListViewDialog({ props, emits, model_value })
    return { dialog, emits, model_value }
}

beforeEach(() => {
    refresh_kyou_mock.mockReset()
    refresh_kyou_in_list_mock.mockReset()
    refresh_kyou_mock.mockResolvedValue(null)
    refresh_kyou_in_list_mock.mockResolvedValue(undefined)
})

describe('useKyouListViewDialog crudRelayHandlers', () => {
    it('ダイアログ層の20イベントを全部張る', () => {
        const { dialog } = build_dialog()

        expect(Object.keys(dialog.crudRelayHandlers).sort()).toEqual([...kyou_dialog_relay_event_names].sort())
    })

    // タグ/テキスト/通知の変更は updated_kyou を出さない。唯一の信号がこれ
    it('requested_reload_kyou で自分のリストを引き直し、親にも中継する', async () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits, model_value } = build_dialog({ kyous: [kyou] })

        dialog.crudRelayHandlers.requested_reload_kyou(kyou)
        await Promise.resolve()

        expect(refresh_kyou_in_list_mock).toHaveBeenCalledWith(model_value.value, kyou, { requested_at: 0 })
        expect(emits).toHaveBeenCalledWith('requested_reload_kyou', kyou)
    })

    it('updated_kyou でも自分のリストを引き直し、親にも中継する', async () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits, model_value } = build_dialog({ kyous: [kyou] })

        dialog.crudRelayHandlers.updated_kyou(kyou)
        await Promise.resolve()

        expect(refresh_kyou_in_list_mock).toHaveBeenCalledWith(model_value.value, kyou, { requested_at: 0 })
        expect(emits).toHaveBeenCalledWith('updated_kyou', kyou)
    })

    // 「このリストの内容を変えないので握りつぶす」は誤り。
    // 要素数は変えないが要素の中身は変えるし、タグ名一覧が変わったことは親が知りたい
    it('タグ/テキスト/通知のCRUDと新規Kyouを握りつぶさずに親へ流す', () => {
        const { dialog, emits } = build_dialog()
        const relayed_event_names = [
            'registered_kyou',
            'registered_tag', 'updated_tag', 'deleted_tag',
            'registered_text', 'updated_text', 'deleted_text',
            'registered_notification', 'updated_notification', 'deleted_notification',
        ] as const

        for (const event_name of relayed_event_names) {
            const handler = dialog.crudRelayHandlers[event_name] as (...args: Array<unknown>) => void
            handler('payload')
            expect(emits, `${event_name} が親へ中継されていない`).toHaveBeenCalledWith(event_name, 'payload')
        }
    })

    it('deleted_kyou は自分のリストから消してから親へ中継する', () => {
        const kyou = make_kyou('kyou-1')
        const other = make_kyou('kyou-2')
        const { dialog, emits, model_value } = build_dialog({ kyous: [kyou, other] })

        dialog.crudRelayHandlers.deleted_kyou(kyou)

        expect(model_value.value).toEqual([other])
        expect(emits).toHaveBeenCalledWith('deleted_kyou', kyou)
    })

    it('クリックはフォーカス移動も伴う', () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits } = build_dialog()

        dialog.crudRelayHandlers.clicked_kyou(kyou)

        expect(emits).toHaveBeenCalledWith('focused_kyou', kyou)
        expect(emits).toHaveBeenCalledWith('clicked_kyou', kyou)
    })
})

describe('useKyouListViewDialog rykvダイアログのホスト', () => {
    // 親へも emit すると同じダイアログが2つ開く
    it('requested_open_rykv_dialog は親へ流さず自分で開く', () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits } = build_dialog()

        dialog.crudRelayHandlers.requested_open_rykv_dialog('add_tag', kyou)

        expect(dialog.opened_dialogs.value).toHaveLength(1)
        expect(dialog.opened_dialogs.value[0].kind).toBe('add_tag')
        expect(emits).not.toHaveBeenCalledWith('requested_open_rykv_dialog', 'add_tag', kyou, undefined)
    })

    it('開いた直後にそのKyouを最新化する', async () => {
        const kyou = make_kyou('kyou-1')
        // 差し替わったことが分かるように別idにする（refは中身をproxyで包むので参照比較できない）
        const refreshed = make_kyou('kyou-1-refreshed')
        refresh_kyou_mock.mockResolvedValue(refreshed)
        const { dialog } = build_dialog()

        dialog.crudRelayHandlers.requested_open_rykv_dialog('kyou', kyou)
        await Promise.resolve()
        await Promise.resolve()

        expect(refresh_kyou_mock).toHaveBeenCalledWith(kyou)
        expect(dialog.opened_dialogs.value[0].kyou.id).toBe('kyou-1-refreshed')
    })

    it('host_rykv_dialogs=false なら従来どおり親へ中継する', () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits } = build_dialog({ host_rykv_dialogs: false })

        dialog.crudRelayHandlers.requested_open_rykv_dialog('add_tag', kyou)

        expect(dialog.opened_dialogs.value).toHaveLength(0)
        expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'add_tag', kyou, undefined)
    })

    it('closed で該当ダイアログだけ閉じる', () => {
        const kyou = make_kyou('kyou-1')
        const { dialog } = build_dialog()

        dialog.crudRelayHandlers.requested_open_rykv_dialog('add_tag', kyou)
        const dialog_id = dialog.opened_dialogs.value[0].id
        dialog.dialogHostHandlers.closed(dialog_id)

        expect(dialog.opened_dialogs.value).toHaveLength(0)
    })

    // ダイアログの中で起きたタグ追加が、このダイアログのリストに戻ってくる経路
    it('ホストしたダイアログからの requested_reload_kyou でリストを引き直す', async () => {
        const kyou = make_kyou('kyou-1')
        const { dialog, emits, model_value } = build_dialog({ kyous: [kyou] })

        dialog.dialogHostHandlers.requested_reload_kyou(kyou)
        await Promise.resolve()

        expect(refresh_kyou_in_list_mock).toHaveBeenCalledWith(model_value.value, kyou, { requested_at: 0 })
        expect(emits).toHaveBeenCalledWith('requested_reload_kyou', kyou)
    })
})
