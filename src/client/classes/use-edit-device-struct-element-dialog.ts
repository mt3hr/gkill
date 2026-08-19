'use strict'

import { type Ref, ref } from 'vue'
import type { EditDeviceStructElementDialogEmits } from '@/pages/dialogs/edit-device-struct-element-dialog-emits.ts'
import type { EditDeviceStructElementDialogProps } from '@/pages/dialogs/edit-device-struct-element-dialog-props.ts'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data.js'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditDeviceStructElementDialog(options: {
    props: EditDeviceStructElementDialogProps
    emits: EditDeviceStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const device_struct: Ref<DeviceStructElementData> = ref(new DeviceStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-device-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(device_struct_obj: DeviceStructElementData): Promise<void> {
        device_struct.value = device_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        device_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
