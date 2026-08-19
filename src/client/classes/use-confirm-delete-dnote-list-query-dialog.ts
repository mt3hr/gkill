'use strict'

import { type Ref, ref } from 'vue'
import DnoteListQuery from '@/pages/views/dnote-list-query';
import type { ConfirmDeleteDnoteListQueryDialogEmits } from '@/pages/dialogs/confirm-delete-dnote-list-query-dialog-emits';
import type { ConfirmDeleteDnoteListQueryDialogProps } from '@/pages/dialogs/confirm-delete-dnote-list-query-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteDnoteListQueryDialog(options: {
    props: ConfirmDeleteDnoteListQueryDialogProps
    emits: ConfirmDeleteDnoteListQueryDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const dnote_list_query: Ref<DnoteListQuery> = ref(new DnoteListQuery())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-dnote-list-query-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(_dnote_item: DnoteListQuery): Promise<void> {
        dnote_list_query.value = _dnote_item
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        dnote_list_query.value = new DnoteListQuery()
    }

    return {
        dnote_list_query,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
