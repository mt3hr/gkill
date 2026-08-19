'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewRepTypeStructElementDialogEmits } from '@/pages/dialogs/add-new-rep-type-struct-element-dialog-emits'
import type { AddNewRepTypeStructElementDialogProps } from '@/pages/dialogs/add-new-rep-type-struct-element-dialog-props'
import AddNewRepTypeStructElementView from '@/pages/views/add-new-rep-type-struct-element-view.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewRepTypeStructElementDialog(options: {
    props: AddNewRepTypeStructElementDialogProps
    emits: AddNewRepTypeStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_rep_type_struct_element_view = ref<InstanceType<typeof AddNewRepTypeStructElementView> | null>(null);
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-new-rep-type-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
