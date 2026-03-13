'use strict'

import { ref, type Ref } from 'vue'
import type { NewBoardNameDialogProps } from '@/pages/dialogs/new-board-name-dialog-props'
import type { NewBoardNameDialogEmits } from '@/pages/dialogs/new-board-name-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useNewBoardNameDialog(options: {
    props: NewBoardNameDialogProps
    emits: NewBoardNameDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("new-board-name-dialog", {
        centerMode: "always",
    })

    const board_name: Ref<string> = ref("")

    async function show(): Promise<void> {
        board_name.value = ""
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }
    function emits_board_name(): void {
        emits('setted_new_board_name', board_name.value)
        hide()
    }

    return {
        is_show_dialog,
        ui,
        board_name,
        show,
        hide,
        emits_board_name,
    }
}
