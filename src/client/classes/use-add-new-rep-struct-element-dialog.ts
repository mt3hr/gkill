'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewRepStructElementDialogEmits } from '@/pages/dialogs/add-new-rep-struct-element-dialog-emits'
import type { AddNewRepStructElementDialogProps } from '@/pages/dialogs/add-new-rep-struct-element-dialog-props'
import AddNewRepStructElementView from '@/pages/views/add-new-rep-struct-element-view.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewRepStructElementDialog(options: {
    props: AddNewRepStructElementDialogProps
    emits: AddNewRepStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_rep_struct_element_view = ref<InstanceType<typeof AddNewRepStructElementView> | null>(null);
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-new-rep-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_rep_struct_element_view.value?.reset_rep_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_rep_struct_element_view.value?.reset_rep_name()
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
