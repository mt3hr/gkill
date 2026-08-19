'use strict'

import { type Ref, ref } from 'vue'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import type { ConfirmDeleteRyuuItemDialogProps } from '@/pages/dialogs/confirm-delete-ryuu-item-dialog-props';
import type { ConfirmDeleteRyuuItemDialogEmits } from '@/pages/dialogs/confirm-delete-ryuu-item-dialog-emits';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteRyuuItemDialog(options: {
    props: ConfirmDeleteRyuuItemDialogProps
    emits: ConfirmDeleteRyuuItemDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const related_kyou_query: Ref<RelatedKyouQuery> = ref(new RelatedKyouQuery())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-ryuu-item-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(related_kyou_query_: RelatedKyouQuery): Promise<void> {
        related_kyou_query.value = related_kyou_query_
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        related_kyou_query.value = new RelatedKyouQuery()
    }

    return {
        related_kyou_query,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
