'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmUnknownTagDialogEmits } from '@/pages/dialogs/confirm-unknown-tag-dialog-emits'
import type { ConfirmUnknownTagDialogProps } from '@/pages/dialogs/confirm-unknown-tag-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmUnknownTagDialog(options: {
    props: ConfirmUnknownTagDialogProps
    emits: ConfirmUnknownTagDialogEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    // ×・Escape・ブラウザバックのどれで閉じても1回だけ上がる。
    // 呼び出し元が「確認が開いているか」を持つときの唯一の倒し方
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("confirm-unknown-tag-dialog", {
        centerMode: "always",
        onEscape: () => cancel(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }
    function confirm(): void {
        hide()
        emits('requested_confirm')
    }
    function cancel(): void {
        hide()
        emits('requested_cancel')
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
        confirm,
        cancel,
    }
}
