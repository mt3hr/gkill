'use strict'

import { type Ref, ref } from 'vue'
import { DnoteCorrelationGraphQuery } from '@/pages/../classes/dnote/dnote-correlation';
import type { ConfirmDeleteDnoteCorrelationGraphDialogEmits } from '@/pages/dialogs/confirm-delete-dnote-correlation-graph-dialog-emits';
import type { ConfirmDeleteDnoteCorrelationGraphDialogProps } from '@/pages/dialogs/confirm-delete-dnote-correlation-graph-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteDnoteCorrelationGraphDialog(options: {
    props: ConfirmDeleteDnoteCorrelationGraphDialogProps
    emits: ConfirmDeleteDnoteCorrelationGraphDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const dnote_correlation_graph_query: Ref<DnoteCorrelationGraphQuery> = ref(new DnoteCorrelationGraphQuery())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-dnote-correlation-graph-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(target: DnoteCorrelationGraphQuery): Promise<void> {
        dnote_correlation_graph_query.value = target
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        dnote_correlation_graph_query,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
