'use strict'

import { ref, type Ref } from 'vue'
import type { ConfirmDeleteTextDialogProps } from '@/pages/dialogs/confirm-delete-text-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmDeleteTextDialog(options: {
    props: ConfirmDeleteTextDialogProps
    emits: KyouDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-text-dialog", {
        centerMode: "always",
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
        emits('closed')
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
