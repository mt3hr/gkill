'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteDeviceStructDialogEmits } from '@/pages/dialogs/confirm-delete-device-struct-dialog-emits.ts';
import type { ConfirmDeleteDeviceStructDialogProps } from '@/pages/dialogs/confirm-delete-device-struct-dialog-props.ts';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteDeviceStructDialog(options: {
    props: ConfirmDeleteDeviceStructDialogProps
    emits: ConfirmDeleteDeviceStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const device_struct: Ref<DeviceStructElementData> = ref(new DeviceStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-device-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(device_struct_obj: DeviceStructElementData): Promise<void> {
        device_struct.value = device_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        device_struct.value = new DeviceStructElementData()
    }

    return {
        device_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
