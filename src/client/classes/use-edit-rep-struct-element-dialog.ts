'use strict'

import { type Ref, ref } from 'vue'
import type { EditRepStructElementDialogEmits } from '@/pages/dialogs/edit-rep-struct-element-dialog-emits'
import type { EditRepStructElementDialogProps } from '@/pages/dialogs/edit-rep-struct-element-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditRepStructElementDialog(options: {
    props: EditRepStructElementDialogProps
    emits: EditRepStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const rep_struct: Ref<RepStructElementData> = ref(new RepStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-rep-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(rep_struct_obj: RepStructElementData): Promise<void> {
        rep_struct.value = rep_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        rep_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
