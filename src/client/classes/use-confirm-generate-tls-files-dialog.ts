'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmGenerateTLSFilesDialogEmits } from '@/pages/dialogs/confirm-generate-tls-files-dialog-emits'
import type { ConfirmGenerateTLSFilesDialogProps } from '@/pages/dialogs/confirm-generate-tls-files-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmGenerateTLSFilesDialog(options: {
    props: ConfirmGenerateTLSFilesDialogProps
    emits: ConfirmGenerateTLSFilesDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-generate-tls-files-dialog", {
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
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
