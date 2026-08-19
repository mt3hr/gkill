'use strict'

import { type Ref, ref } from 'vue'
import type { ShareKyousListLinkDialogEmits } from '@/pages/dialogs/share-kyou-list-link-dialog-emits'
import type { ShareKyousListLinkDialogProps } from '@/pages/dialogs/share-kyou-list-link-dialog-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useShareKyouListLinkDialog(options: {
    props: ShareKyousListLinkDialogProps
    emits: ShareKyousListLinkDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("share-kyou-list-link-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const share_kyou_list_info: Ref<ShareKyousInfo | null> = ref(null)
    async function show(share_kyou_list_info_: ShareKyousInfo): Promise<void> {
        share_kyou_list_info.value = share_kyou_list_info_
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        share_kyou_list_info.value = null
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        ui,
        share_kyou_list_info,
        show,
        hide,
    }
}
