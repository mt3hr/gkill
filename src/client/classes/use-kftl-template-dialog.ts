'use strict'

import { type Ref, ref } from 'vue'
import type { KFTLTemplateDialogEmits } from '@/pages/dialogs/kftl-template-dialog-emits'
import type { KFTLTemplateDialogProps } from '@/pages/dialogs/kftl-template-dialog-props'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useKFTLTemplateDialog(options: {
    props: KFTLTemplateDialogProps
    emits: KFTLTemplateDialogEmits
}) {
    const { props: _props, emits } = options

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("kftl-template-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        emits('closed_dialog')
    }

    return {
        help_dialog,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
