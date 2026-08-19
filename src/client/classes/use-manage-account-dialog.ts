'use strict'

import { type Ref, ref } from 'vue'
import type { ManageAccountDialogEmits } from '@/pages/dialogs/manage-account-dialog-emits'
import type { ManageAccountDialogProps } from '@/pages/dialogs/manage-account-dialog-props'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useManageAccountDialog(options: {
    props: ManageAccountDialogProps
    emits: ManageAccountDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("manage-account-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        help_dialog,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
