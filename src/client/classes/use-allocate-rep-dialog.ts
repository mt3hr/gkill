'use strict'

import { type Ref, ref } from 'vue'
import type { AllocateRepDialogEmits } from '@/pages/dialogs/allocate-rep-dialog-emits'
import type { AllocateRepDialogProps } from '@/pages/dialogs/allocate-rep-dialog-props'
import { Account } from '@/classes/datas/config/account';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAllocateRepDialog(options: {
    props: AllocateRepDialogProps
    emits: AllocateRepDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("allocate-rep-dialog", {
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
