'use strict'

import { type Ref, ref } from 'vue'
import type { EditDeviceStructDialogEmits } from '@/pages/dialogs/edit-device-struct-dialog-emits.ts'
import type { EditDeviceStructDialogProps } from '@/pages/dialogs/edit-device-struct-dialog-props.ts'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditDeviceStructDialog(options: {
    props: EditDeviceStructDialogProps
    emits: EditDeviceStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-device-struct-dialog", {
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
