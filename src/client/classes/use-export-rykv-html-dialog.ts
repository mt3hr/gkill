'use strict'

import type { ExportRykvHTMLDialogEmits } from '@/pages/dialogs/export-rykv-html-dialog-emits'
import type { ExportRykvHTMLDialogProps } from '@/pages/dialogs/export-rykv-html-dialog-props'
import { ref, type Ref } from 'vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useExportRykvHTMLDialog(options: {
    props: ExportRykvHTMLDialogProps
    emits: ExportRykvHTMLDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("export-rykv-html-dialog", {
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
        ui,
        show,
        hide,
    }
}
