'use strict'

import { computed, type Ref, ref } from 'vue'
import type { ConfirmSaveDuplicatedSharedDataDialogEmits } from '@/pages/dialogs/confirm-save-duplicated-shared-data-dialog-emits'
import type { ConfirmSaveDuplicatedSharedDataDialogProps } from '@/pages/dialogs/confirm-save-duplicated-shared-data-dialog-props'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmSaveDuplicatedSharedDataDialog(options: {
    props: ConfirmSaveDuplicatedSharedDataDialogProps
    emits: ConfirmSaveDuplicatedSharedDataDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-save-duplicated-shared-data-dialog", {
        centerMode: "always",
        onEscape: () => cancel(),
    })
    // 何が重複したのかを出す。URL 共有が大半なので URL を優先する
    const shared_summary = computed(() => {
        const payload = props.entry?.payload
        if (!payload) {
            return ""
        }
        return payload.url || payload.text || payload.title
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }
    function confirm(): void {
        hide()
        emits('requested_save')
    }
    function cancel(): void {
        hide()
        emits('requested_cancel')
    }

    return {
        is_show_dialog,
        ui,
        shared_summary,
        show,
        hide,
        confirm,
        cancel,
    }
}
