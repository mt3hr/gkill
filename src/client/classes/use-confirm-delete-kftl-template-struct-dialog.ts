'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteKFTLTemplateStructDialogEmits } from '@/pages/dialogs/confirm-delete-kftl-template-struct-dialog-emits.ts';
import type { ConfirmDeleteKFTLTemplateStructDialogProps } from '@/pages/dialogs/confirm-delete-kftl-template-struct-dialog-props.ts';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteKFTLTemplateStructDialog(options: {
    props: ConfirmDeleteKFTLTemplateStructDialogProps
    emits: ConfirmDeleteKFTLTemplateStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const kftl_template_struct: Ref<KFTLTemplateElementData> = ref(new KFTLTemplateElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-kftl-template-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(kftl_template_struct_obj: KFTLTemplateElementData): Promise<void> {
        kftl_template_struct.value = kftl_template_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        kftl_template_struct.value = new KFTLTemplateElementData()
    }

    return {
        kftl_template_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
