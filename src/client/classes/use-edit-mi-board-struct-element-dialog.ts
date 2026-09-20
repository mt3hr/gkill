'use strict'

import { type Ref, ref } from 'vue'
import type { EditMiBoardStructElementDialogEmits } from '@/pages/dialogs/edit-mi-board-struct-element-dialog-emits'
import type { EditMiBoardStructElementDialogProps } from '@/pages/dialogs/edit-mi-board-struct-element-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditMiBoardStructElementDialog(options: {
    props: EditMiBoardStructElementDialogProps
    emits: EditMiBoardStructElementDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const mi_board_struct: Ref<MiBoardStructElementData> = ref(new MiBoardStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-mi-board-struct-element-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(mi_board_struct_obj: MiBoardStructElementData): Promise<void> {
        mi_board_struct.value = mi_board_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        mi_board_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
