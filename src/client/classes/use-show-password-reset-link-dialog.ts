'use strict'

import { type Ref, ref } from 'vue'
import type { ShowPasswordResetLinkDialogEmits } from '@/pages/dialogs/show-password-reset-link-dialog-emits'
import type { ShowPasswordResetLinkDialogProps } from '@/pages/dialogs/show-password-reset-link-dialog-props'
import { Account } from '@/classes/datas/config/account';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useShowPasswordResetLinkDialog(options: {
    props: ShowPasswordResetLinkDialogProps
    emits: ShowPasswordResetLinkDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("show-password-reset-link-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const cloned_account: Ref<Account> = ref(new Account())
    async function show(account: Account): Promise<void> {
        cloned_account.value = account
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        cloned_account.value = new Account()
    }

    return {
        is_show_dialog,
        ui,
        cloned_account,
        show,
        hide,
    }
}
