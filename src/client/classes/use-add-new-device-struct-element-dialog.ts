'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewDeviceStructElementDialogEmits } from '@/pages/dialogs/add-new-device-struct-element-dialog-emits'
import type { AddNewDeviceStructElementDialogProps } from '@/pages/dialogs/add-new-device-struct-element-dialog-props'
import AddNewDeviceStructElementView from '@/pages/views/add-new-device-struct-element-view.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewDeviceStructElementDialog(options: {
    props: AddNewDeviceStructElementDialogProps
    emits: AddNewDeviceStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_device_struct_element_view = ref<InstanceType<typeof AddNewDeviceStructElementView> | null>(null);
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-new-device-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_device_struct_element_view.value?.reset_device_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_device_struct_element_view.value?.reset_device_name()
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
