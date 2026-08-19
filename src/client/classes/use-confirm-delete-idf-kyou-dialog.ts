'use strict'

import type { Kyou } from '@/classes/datas/kyou';
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { ref, type Ref } from 'vue'
import type { ConfirmDeleteIDFKyouDialogProps } from '@/pages/dialogs/confirm-delete-idf-kyou-dialog-props'
import type { ConfirmDeleteIDFKyouDialogEmits } from '@/pages/dialogs/confirm-delete-idf-kyou-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmDeleteIDFKyouDialog(options: {
    props: ConfirmDeleteIDFKyouDialogProps
    emits: ConfirmDeleteIDFKyouDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("confirm-delete-idf-kyou-dialog", {
        centerMode: "always",
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    return {
        crudRelayHandlers,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
