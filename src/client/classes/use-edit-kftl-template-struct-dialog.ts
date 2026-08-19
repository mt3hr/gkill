'use strict'

import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructDialogEmits } from '@/pages/dialogs/edit-kftl-template-struct-dialog-emits.ts'
import type { EditKFTLTemplateStructDialogProps } from '@/pages/dialogs/edit-kftl-template-struct-dialog-props.ts'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEditKFTLTemplateStructDialog(options: {
    props: EditKFTLTemplateStructDialogProps
    emits: EditKFTLTemplateStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-kftl-template-struct-dialog", {
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
        help_dialog,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
