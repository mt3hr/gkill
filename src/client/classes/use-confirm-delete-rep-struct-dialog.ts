'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepStructDialogEmits } from '@/pages/dialogs/confirm-delete-rep-struct-dialog-emits.ts';
import type { ConfirmDeleteRepStructDialogProps } from '@/pages/dialogs/confirm-delete-rep-struct-dialog-props.ts';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteRepStructDialog(options: {
    props: ConfirmDeleteRepStructDialogProps
    emits: ConfirmDeleteRepStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const rep_struct: Ref<RepStructElementData> = ref(new RepStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-rep-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(rep_struct_obj: RepStructElementData): Promise<void> {
        rep_struct.value = rep_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        rep_struct.value = new RepStructElementData()
    }

    return {
        rep_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
