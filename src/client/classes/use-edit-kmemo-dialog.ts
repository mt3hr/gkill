'use strict'

import { ref, type Ref } from 'vue'
import type { EditKmemoDialogProps } from '@/pages/dialogs/edit-kmemo-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useEditKmemoDialog(options: {
    props: EditKmemoDialogProps
    emits: KyouDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("edit-kmemo-dialog", {
        centerMode: "always",
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
