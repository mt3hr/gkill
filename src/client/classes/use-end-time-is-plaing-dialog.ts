'use strict'

import { ref, type Ref } from 'vue'
import type { EndTimeIsPlaingDialogProps } from '@/pages/dialogs/end-time-is-plaing-dialog-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useEndTimeIsPlaingDialog(options: {
    props: EndTimeIsPlaingDialogProps
    emits: KyouViewEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("end-time-is-plaing-dialog", {
        centerMode: "always",
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
