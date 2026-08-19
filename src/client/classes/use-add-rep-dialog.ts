'use strict'

import { type Ref, ref } from 'vue'
import type { AddRepDialogEmits } from '@/pages/dialogs/add-rep-dialog-emits'
import type { AddRepDialogProps } from '@/pages/dialogs/add-rep-dialog-props'
import { Account } from '@/classes/datas/config/account';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddRepDialog(options: {
    props: AddRepDialogProps
    emits: AddRepDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-rep-dialog", {
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
