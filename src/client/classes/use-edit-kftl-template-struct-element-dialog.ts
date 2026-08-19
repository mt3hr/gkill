'use strict'

import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementDialogEmits } from '@/pages/dialogs/edit-kftl-template-struct-element-dialog-emits'
import type { EditKFTLTemplateStructElementDialogProps } from '@/pages/dialogs/edit-kftl-template-struct-element-dialog-props'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditKFTLTemplateStructElementDialog(options: {
    props: EditKFTLTemplateStructElementDialogProps
    emits: EditKFTLTemplateStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const kftl_template_struct: Ref<KFTLTemplateElementData> = ref(new KFTLTemplateElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-kftl-template-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(kftl_template_struct_obj: KFTLTemplateElementData): Promise<void> {
        kftl_template_struct.value = kftl_template_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        help_dialog,
        kftl_template_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
