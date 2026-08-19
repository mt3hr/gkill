'use strict'

import { type Ref, ref } from 'vue'
import type { EditMiBoardStructDialogEmits } from '@/pages/dialogs/edit-mi-board-struct-dialog-emits'
import type { EditMiBoardStructDialogProps } from '@/pages/dialogs/edit-mi-board-struct-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditMiBoardStructDialog(options: {
    props: EditMiBoardStructDialogProps
    emits: EditMiBoardStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-mi-board-struct-dialog", {
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
