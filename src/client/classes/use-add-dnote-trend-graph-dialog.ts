'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { ref, type Ref } from 'vue'
import type AddDnoteTrendGraphDialogEmits from '@/pages/dialogs/add-dnote-trend-graph-dialog-emits';
import type AddDnoteTrendGraphDialogProps from '@/pages/dialogs/add-dnote-trend-graph-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddDnoteTrendGraphDialog(options: {
    props: AddDnoteTrendGraphDialogProps
    emits: AddDnoteTrendGraphDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-dnote-trend-graph-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        ui,
        help_dialog,
        show,
        hide,
    }
}
