'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteDnoteItemListDialogEmits } from '@/pages/dialogs/confirm-delete-dnote-item-list-dialog-emits';
import type { ConfirmDeleteDnoteItemListDialogProps } from '@/pages/dialogs/confirm-delete-dnote-item-list-dialog-props';
import DnoteItem from '@/classes/dnote/dnote-item';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteDnoteItemListDialog(options: {
    props: ConfirmDeleteDnoteItemListDialogProps
    emits: ConfirmDeleteDnoteItemListDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const dnote_item: Ref<DnoteItem> = ref(new DnoteItem())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-dnote-item-list-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(_dnote_item: DnoteItem): Promise<void> {
        dnote_item.value = _dnote_item
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        dnote_item.value = new DnoteItem()
    }

    return {
        dnote_item,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
