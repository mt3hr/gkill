'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmLogoutDialogEmits } from '@/pages/dialogs/confirm-logout-dialog-emits'
import type { ConfirmLogoutDialogProps } from '@/pages/dialogs/confirm-logout-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmLogoutDialog(options: {
    props: ConfirmLogoutDialogProps
    emits: ConfirmLogoutDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-logout-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const close_database_value: Ref<boolean> = ref(false)
    async function show(close_database: boolean): Promise<void> {
        close_database_value.value = close_database
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        close_database_value.value = false
    }

    return {
        is_show_dialog,
        ui,
        close_database_value,
        show,
        hide,
    }
}
