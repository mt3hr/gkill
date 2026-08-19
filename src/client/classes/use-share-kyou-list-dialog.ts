'use strict'

import type { ShareKyousListDialogEmits } from '@/pages/dialogs/share-kyou-list-dialog-emits'
import type { ShareKyousListDialogProps } from '@/pages/dialogs/share-kyou-list-dialog-props'
import { ref, type Ref } from 'vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useShareKyouListDialog(options: {
    props: ShareKyousListDialogProps
    emits: ShareKyousListDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("share-kyou-list-dialog", {
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
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
