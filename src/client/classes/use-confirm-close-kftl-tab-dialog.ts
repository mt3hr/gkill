'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmCloseKFTLTabDialogEmits } from '@/pages/dialogs/confirm-close-kftl-tab-dialog-emits'
import type { ConfirmCloseKFTLTabDialogProps } from '@/pages/dialogs/confirm-close-kftl-tab-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmCloseKFTLTabDialog(options: {
    props: ConfirmCloseKFTLTabDialogProps
    emits: ConfirmCloseKFTLTabDialogEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-close-kftl-tab-dialog", {
        centerMode: "always",
        onEscape: () => cancel(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }
    function confirm(): void {
        hide()
        emits('requested_confirm')
    }
    function cancel(): void {
        hide()
        emits('requested_cancel')
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
        confirm,
        cancel,
    }
}
