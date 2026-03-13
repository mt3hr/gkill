'use strict'

import { ref, type Ref } from 'vue'
import type { NewDeviceNameDialogProps } from '@/pages/dialogs/new-device-name-dialog-props'
import type { NewDeviceNameDialogEmits } from '@/pages/dialogs/new-device-name-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useNewDeviceNameDialog(options: {
    props: NewDeviceNameDialogProps
    emits: NewDeviceNameDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("new-device-name-dialog", {
        centerMode: "always",
    })

    const device_name: Ref<string> = ref("")

    async function show(): Promise<void> {
        device_name.value = ""
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }
    function emits_board_name(): void {
        emits('setted_new_device_name', device_name.value)
        hide()
    }

    return {
        is_show_dialog,
        ui,
        device_name,
        show,
        hide,
        emits_board_name,
    }
}
