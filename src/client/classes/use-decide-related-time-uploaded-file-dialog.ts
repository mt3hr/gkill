'use strict'

import { type Ref, ref } from 'vue'
import type { DecideRelatedTimeUploadedFileDialogEmits } from '@/pages/dialogs/decide-related-time-uploaded-file-dialog-emits'
import type { DecideRelatedTimeUploadedFileDialogProps } from '@/pages/dialogs/decide-related-time-uploaded-file-dialog-props'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useDecideRelatedTimeUploadedFileDialog(options: {
    props: DecideRelatedTimeUploadedFileDialogProps
    emits: DecideRelatedTimeUploadedFileDialogEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    // 中身のビューは KyouViewEmits を全部上げてくる。
    // 手書きで並べていたころはタグ・テキスト・通知のCRUD 9件が抜けていた
    const crudRelayHandlers = build_kyou_dialog_relay(emits)
    const ui = useFloatingDialog("decide-related-time-uploaded-file-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        crudRelayHandlers,
        ui,
        show,
        hide,
    }
}
