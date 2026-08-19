'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewKFTLTemplateStructElementDialogEmits } from '@/pages/dialogs/add-new-kftl-template-struct-element-dialog-emits'
import type { AddNewKFTLTemplateStructElementDialogProps } from '@/pages/dialogs/add-new-kftl-template-struct-element-dialog-props'
import AddNewKFTLTemplateStructElementView from '@/pages/views/add-new-kftl-template-struct-element-view.vue'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewKFTLTemplateStructElementDialog(options: {
    props: AddNewKFTLTemplateStructElementDialogProps
    emits: AddNewKFTLTemplateStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_kftl_template_struct_element_view = ref<InstanceType<typeof AddNewKFTLTemplateStructElementView> | null>(null);
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-new-kftl-template-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_kftl_template_struct_element_view.value?.reset_kftl_template_name()
    }

    return {
        help_dialog,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
