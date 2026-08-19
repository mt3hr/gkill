'use strict'

import { type Ref, ref } from 'vue'
import type { EditTagStructDialogEmits } from '@/pages/dialogs/edit-tag-struct-dialog-emits'
import type { EditTagStructDialogProps } from '@/pages/dialogs/edit-tag-struct-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditTagStructDialog(options: {
    props: EditTagStructDialogProps
    emits: EditTagStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-tag-struct-dialog", {
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
