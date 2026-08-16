'use strict'

import { ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { KFTLDialogHostEmits } from '@/pages/views/kftl-dialog-host-emits'

/** 同時に開けるメモ帳ウィンドウの上限。これ以上は増やさず、最後の1枚へフォーカスを移す */
export const KFTL_DIALOG_MAX_COUNT = 8

export interface OpenedKFTLDialog {
    id: string

    /**
     * 位置・サイズの保存キーと、中央からずらす量を決める番号。
     * 閉じたら空くので、次に開くウィンドウが**空いている最小の番号**を取る
     */
    slot_index: number
}

/**
 * メモ帳ウィンドウの一覧を持つ。
 *
 * `rykv-dialog-host` と同じ形（配列へ push して開き、`closed` で splice）。
 * ホストコンポーネントが `show()` を expose するので、
 * 呼び出し側（5画面）は `kftl_dialog.value?.show()` のままでよい。
 */
export function useKftlDialogHost(options: {
    emits: KFTLDialogHostEmits,
}) {
    const { emits } = options

    // ── State refs ──
    const opened_dialogs: Ref<Array<OpenedKFTLDialog>> = ref([])

    // 上限に達したときにフォーカスを移す先。kftl-dialog.vue の show() は
    // 「開いていればテキストエリアに再フォーカスするだけ」なので、そのまま呼べばよい
    const dialog_refs: Ref<Array<{ show?: () => unknown } | null>> = ref([])

    // ── Business logic ──
    function next_slot_index(): number {
        const used = new Set(opened_dialogs.value.map(dialog => dialog.slot_index))
        for (let index = 0; index < KFTL_DIALOG_MAX_COUNT; index++) {
            if (!used.has(index)) {
                return index
            }
        }
        return KFTL_DIALOG_MAX_COUNT - 1
    }

    /** ＋メニューから呼ばれる。呼ぶたびに1枚増える */
    function show(): void {
        if (opened_dialogs.value.length >= KFTL_DIALOG_MAX_COUNT) {
            focus_newest_dialog()
            return
        }
        opened_dialogs.value.push({
            id: GkillAPI.get_instance().generate_uuid(),
            slot_index: next_slot_index(),
        })
    }

    function close(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                return
            }
        }
    }

    function focus_newest_dialog(): void {
        const newest = dialog_refs.value[dialog_refs.value.length - 1]
        newest?.show?.()
    }

    // ── Event relay objects ──
    // どのウィンドウから来たかに依らないので、同じオブジェクトを全枚数へ配ってよい
    const dialogRelayHandlers = {
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'saved_kyou_by_kftl': (last_added_request_time: Date) => emits('saved_kyou_by_kftl', last_added_request_time),
    }

    // ── Return ──
    return {
        // State
        opened_dialogs,
        dialog_refs,

        // Business logic
        show,
        close,

        // Event relay objects
        dialogRelayHandlers,
    }
}
