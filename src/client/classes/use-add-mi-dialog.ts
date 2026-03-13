'use strict'

import { ref, type Ref } from 'vue'
import type { AddMiDialogProps } from '@/pages/dialogs/add-mi-dialog-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useAddMiDialog(options: {
    props: AddMiDialogProps
    emits: KyouViewEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-mi-dialog", {
        centerMode: "always",
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
