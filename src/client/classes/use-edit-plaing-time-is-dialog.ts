'use strict'

import { computed, ref, type Ref } from 'vue'
import type { EditPlaingTimeIsDialogProps } from '@/pages/dialogs/edit-plaing-time-is-dialog-props'
import type { EditPlaingTimeIsDialogEmits } from '@/pages/dialogs/edit-plaing-time-is-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export function useEditPlaingTimeIsDialog(options: {
    props: EditPlaingTimeIsDialogProps
    emits: EditPlaingTimeIsDialogEmits
}) {
    const { props } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-plaing-time-is-dialog", {
        centerMode: "always",
    })

    // null = 未設定（デフォルト動作）。チェックボックスのOFFを表現するため
    // dashboard版と違い null を第一級の状態として持つ
    const current_query = ref<FindKyouQuery | null>(null)
    // エディタダイアログの v-model 用。null だとエディタが描画されないので別持ちにする
    const editor_model = ref<FindKyouQuery>(new FindKyouQuery())

    // Ryuu の「検索条件をカスタマイズする」と同じ意味論。
    // OFFにすると null に戻る＝デフォルト動作（旧「デフォルトに戻す」ボタン相当）
    const is_use_custom_find_kyou_query = computed<boolean>({
        get: () => current_query.value !== null,
        set: (value: boolean) => {
            if (!value) {
                current_query.value = null
                return
            }
            if (current_query.value === null) {
                current_query.value = FindKyouQuery.generate_default_query_for_plaing_timeis(props.application_config)
            }
        },
    })

    async function show(initial_query?: FindKyouQuery): Promise<void> {
        current_query.value = initial_query ?? null
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        ui,
        current_query,
        editor_model,
        is_use_custom_find_kyou_query,
        show,
        hide,
    }
}
