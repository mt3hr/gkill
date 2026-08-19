'use strict'

import { type Ref, ref } from 'vue'
import type { EditIDFKyouDialogProps } from '@/pages/dialogs/edit-idf-kyou-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditIDFKyouDialog(options: {
    props: EditIDFKyouDialogProps
    emits: KyouDialogEmits
}) {
    const { props: _props, emits } = options

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("edit-idf-kyou-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        crudRelayHandlers,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
