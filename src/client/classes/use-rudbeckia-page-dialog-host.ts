'use strict'

import { ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import {
    RUDBECKIA_PAGE_DIALOG_CASCADE_SLOT_COUNT,
    RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND,
    type OpenedRudbeckiaPageDialog,
    type RudbeckiaPageKind,
} from '@/pages/views/rudbeckia-page-kind'
import { build_rudbeckia_page_dialog_host_relay } from '@/classes/rudbeckia-hosted-view-relay'
import type { RudbeckiaPageDialogHostEmits } from '@/pages/views/rudbeckia-page-dialog-host-emits'

/**
 * ポート（開発コード rudbeckia）が開いている画面ウィンドウの一覧を持つ。
 *
 * `kftl-dialog-host` と同じ形（配列へ push して開き、`closed` で splice）。
 * ホストコンポーネントが `show(kind)` を expose するので、
 * ポート側は `page_dialog_host.value?.show('rykv')` と呼ぶだけでよい。
 */
export function useRudbeckiaPageDialogHost(options: {
    emits: RudbeckiaPageDialogHostEmits,
}) {
    const { emits } = options

    // ── State refs ──
    const opened_dialogs: Ref<Array<OpenedRudbeckiaPageDialog>> = ref([])

    // 上限に達したときにフォーカスを移す先
    const dialog_refs: Ref<Array<{ show?: () => unknown } | null>> = ref([])

    // ── Business logic ──
    /** その種類で空いている最小の番号。閉じたら空くので使い回す */
    function next_slot_index(kind: RudbeckiaPageKind): number {
        const used = new Set(opened_dialogs.value
            .filter(dialog => dialog.kind === kind)
            .map(dialog => dialog.slot_index))
        for (let index = 0; index < RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND; index++) {
            if (!used.has(index)) {
                return index
            }
        }
        return RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND - 1
    }

    /**
     * ずらす段数。**種類をまたいで**空いている最小の番号を取る。
     * slot_index は種類ごとの採番なので、そのまま使うと4種類とも 0 になって
     * ウィンドウが完全に重なる
     */
    function next_cascade_index(): number {
        const used = new Set(opened_dialogs.value.map(dialog => dialog.cascade_index))
        for (let index = 0; index < RUDBECKIA_PAGE_DIALOG_CASCADE_SLOT_COUNT; index++) {
            if (!used.has(index)) {
                return index
            }
        }
        return opened_dialogs.value.length % RUDBECKIA_PAGE_DIALOG_CASCADE_SLOT_COUNT
    }

    function count_of(kind: RudbeckiaPageKind): number {
        return opened_dialogs.value.filter(dialog => dialog.kind === kind).length
    }

    /** ＋メニュー / 中の画面切替メニューから呼ばれる。呼ぶたびに1枚増える */
    function show(kind: RudbeckiaPageKind): void {
        if (count_of(kind) >= RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND) {
            focus_newest_dialog_of(kind)
            return
        }
        opened_dialogs.value.push({
            id: GkillAPI.get_instance().generate_uuid(),
            kind: kind,
            slot_index: next_slot_index(kind),
            cascade_index: next_cascade_index(),
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

    function focus_newest_dialog_of(kind: RudbeckiaPageKind): void {
        for (let i = opened_dialogs.value.length - 1; i >= 0; i--) {
            if (opened_dialogs.value[i].kind !== kind) {
                continue
            }
            dialog_refs.value[i]?.show?.()
            return
        }
    }

    // ── Event relay objects ──
    // どのウィンドウから来たかに依らないので、同じオブジェクトを全枚数へ配ってよい
    const dialogRelayHandlers = build_rudbeckia_page_dialog_host_relay(emits)

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
