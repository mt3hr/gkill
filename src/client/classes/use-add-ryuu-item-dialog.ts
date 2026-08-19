'use strict'

import { ref, type Ref } from 'vue'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import type AddRyuuItemDialogProps from '@/pages/dialogs/add-ryuu-item-dialog-props';
import type AddRyuuItemDialogEmits from '@/pages/dialogs/add-ryuu-item-dialog-emits';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddRyuuItemDialog(options: {
    props: AddRyuuItemDialogProps
    emits: AddRyuuItemDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("add-ryuu-item-dialog", {
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
