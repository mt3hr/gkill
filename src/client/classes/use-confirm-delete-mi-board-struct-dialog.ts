'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteMiBoardStructDialogEmits } from '@/pages/dialogs/confirm-delete-mi-board-struct-dialog-emits';
import type { ConfirmDeleteMiBoardStructDialogProps } from '@/pages/dialogs/confirm-delete-mi-board-struct-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteMiBoardStructDialog(options: {
    props: ConfirmDeleteMiBoardStructDialogProps
    emits: ConfirmDeleteMiBoardStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const mi_board_struct: Ref<MiBoardStructElementData> = ref(new MiBoardStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-mi-board-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(mi_board_struct_obj: MiBoardStructElementData): Promise<void> {
        mi_board_struct.value = mi_board_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        mi_board_struct.value = new MiBoardStructElementData()
    }

    return {
        mi_board_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
