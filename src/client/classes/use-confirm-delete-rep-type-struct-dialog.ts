'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepTypeStructDialogEmits } from '@/pages/dialogs/confirm-delete-rep-type-struct-dialog-emits';
import type { ConfirmDeleteRepTypeStructDialogProps } from '@/pages/dialogs/confirm-delete-rep-type-struct-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteRepTypeStructDialog(options: {
    props: ConfirmDeleteRepTypeStructDialogProps
    emits: ConfirmDeleteRepTypeStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const rep_type_struct: Ref<RepTypeStructElementData> = ref(new RepTypeStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-rep-type-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(rep_type_struct_obj: RepTypeStructElementData): Promise<void> {
        rep_type_struct.value = rep_type_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        rep_type_struct.value = new RepTypeStructElementData()
    }

    return {
        rep_type_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
