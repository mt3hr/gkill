'use strict'

import { type Ref, ref } from 'vue'
import DnoteTrendGraphQuery from '@/pages/views/dnote-trend-graph-query';
import type { ConfirmDeleteDnoteTrendGraphDialogEmits } from '@/pages/dialogs/confirm-delete-dnote-trend-graph-dialog-emits';
import type { ConfirmDeleteDnoteTrendGraphDialogProps } from '@/pages/dialogs/confirm-delete-dnote-trend-graph-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteDnoteTrendGraphDialog(options: {
    props: ConfirmDeleteDnoteTrendGraphDialogProps
    emits: ConfirmDeleteDnoteTrendGraphDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const dnote_trend_graph_query: Ref<DnoteTrendGraphQuery> = ref(new DnoteTrendGraphQuery())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-dnote-trend-graph-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(_dnote_trend_graph_query: DnoteTrendGraphQuery): Promise<void> {
        dnote_trend_graph_query.value = _dnote_trend_graph_query
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        dnote_trend_graph_query.value = new DnoteTrendGraphQuery()
    }

    return {
        dnote_trend_graph_query,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
