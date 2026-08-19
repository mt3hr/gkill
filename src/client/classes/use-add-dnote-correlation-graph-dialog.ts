'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { ref, type Ref } from 'vue'
import type AddDnoteCorrelationGraphDialogEmits from '@/pages/dialogs/add-dnote-correlation-graph-dialog-emits';
import type AddDnoteCorrelationGraphDialogProps from '@/pages/dialogs/add-dnote-correlation-graph-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddDnoteCorrelationGraphDialog(options: {
    props: AddDnoteCorrelationGraphDialogProps
    emits: AddDnoteCorrelationGraphDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-dnote-correlation-graph-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    // 本文は Teleport の v-if 配下なので、開くたびに作り直される。
    // 追加フォームの初期化は AddDnoteCorrelationGraphView 側のマウント時に行われ、
    // 前回入力した指標が残ることはない
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
