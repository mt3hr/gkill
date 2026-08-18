'use strict'

import { onUnmounted, ref, type Ref } from 'vue'
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
 * 使用中の slot_index。**ホスト単位ではなくアプリ全体で1つ**。
 *
 * slot_index は `kftl-dialog` / `kftl-dialog-2` … という useFloatingDialog の
 * 保存キーそのものなので、2つのホストがそれぞれ 0 を配ると
 * 位置とサイズを奪い合う。ポート(rudbeckia)ではポート自身とホストした各画面が
 * 同時に KFTLDialogHost を持つので、ホスト内だけの採番では必ず衝突する。
 */
const used_kftl_slot_indexes = new Set<number>()

/** テスト用。モジュール共有なので、テスト間で持ち越さないよう beforeEach で呼ぶ */
export function reset_kftl_dialog_host_slots_for_test(): void {
    used_kftl_slot_indexes.clear()
}

/**
 * メモ帳ウィンドウの一覧を持つ。
 *
 * `rykv-dialog-host` と同じ形（配列へ push して開き、`closed` で splice）。
 * ホストコンポーネントが `show()` を expose するので、
 * 呼び出し側は `kftl_dialog.value?.show()` のままでよい。
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
    /** 空いている最小の番号。**全ホストを通して**空いているもの */
    function next_slot_index(): number {
        for (let index = 0; index < KFTL_DIALOG_MAX_COUNT; index++) {
            if (!used_kftl_slot_indexes.has(index)) {
                return index
            }
        }
        return KFTL_DIALOG_MAX_COUNT - 1
    }

    // ホストごと消えるとき（ポートのウィンドウを閉じたときなど）に
    // 掴んだままの番号を返す。返さないと二度と使えない番号が増えていく
    onUnmounted(() => {
        for (const dialog of opened_dialogs.value) {
            used_kftl_slot_indexes.delete(dialog.slot_index)
        }
    })

    /** ＋メニューから呼ばれる。呼ぶたびに1枚増える */
    function show(): void {
        // 上限も全ホスト通し。ホストごとに8枚だと、ポートでは画面の数だけ増えてしまう
        if (used_kftl_slot_indexes.size >= KFTL_DIALOG_MAX_COUNT) {
            focus_newest_dialog()
            return
        }
        const slot_index = next_slot_index()
        used_kftl_slot_indexes.add(slot_index)
        opened_dialogs.value.push({
            id: GkillAPI.get_instance().generate_uuid(),
            slot_index: slot_index,
        })
    }

    function close(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                used_kftl_slot_indexes.delete(opened_dialogs.value[i].slot_index)
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
