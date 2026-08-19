'use strict'

import type { KyouHistoriesDialogProps } from '@/pages/dialogs/kyou-histories-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { type Ref, ref } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useKyouHistoriesDialog(options: {
    props: KyouHistoriesDialogProps
    emits: KyouDialogEmits
}) {
    const { props: _props, emits } = options

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("kyou-histories-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const show_kyou: Ref<boolean> = ref(false)
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
        show_kyou,
        show,
        hide,
    }
}
