'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewTagStructElementDialogEmits } from '@/pages/dialogs/add-new-tag-struct-element-dialog-emits'
import type { AddNewTagStructElementDialogProps } from '@/pages/dialogs/add-new-tag-struct-element-dialog-props'
import AddNewTagStructElementView from '@/pages/views/add-new-tag-struct-element-view.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewTagStructElementDialog(options: {
    props: AddNewTagStructElementDialogProps
    emits: AddNewTagStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_tag_struct_element_view = ref<InstanceType<typeof AddNewTagStructElementView> | null>(null);
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-new-tag-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_tag_struct_element_view.value?.reset_tag_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_tag_struct_element_view.value?.reset_tag_name()
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
