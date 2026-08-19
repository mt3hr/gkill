'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteShareKyousLinkDialogEmits } from '@/pages/dialogs/confirm-delete-share-kyou-link-dialog-emits'
import type { ConfirmDeleteShareKyousLinkDialogProps } from '@/pages/dialogs/confirm-delete-share-kyou-link-dialog-props'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteShareKyouListDialog(options: {
    props: ConfirmDeleteShareKyousLinkDialogProps
    emits: ConfirmDeleteShareKyousLinkDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const share_kyou_list_info: Ref<ShareKyousInfo | null> = ref(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-share-kyou-list-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(share_kyou_list_info_: ShareKyousInfo): Promise<void> {
        share_kyou_list_info.value = share_kyou_list_info_
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        share_kyou_list_info,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
