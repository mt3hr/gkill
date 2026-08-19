'use strict'

import { type Ref, ref } from 'vue'
import type { EditRepTypeDialogEmits } from '@/pages/dialogs/edit-rep-type-dialog-emits'
import type { EditRepTypeDialogProps } from '@/pages/dialogs/edit-rep-type-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditRepTypeStructDialog(options: {
    props: EditRepTypeDialogProps
    emits: EditRepTypeDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-rep-type-struct-dialog", {
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
