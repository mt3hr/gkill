'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepDialogEmits } from '@/pages/dialogs/confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from '@/pages/dialogs/confirm-delete-rep-dialog-props'
import { Repository } from '@/classes/datas/config/repository';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteRepDialog(options: {
    props: ConfirmDeleteRepDialogProps
    emits: ConfirmDeleteRepDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-rep-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const cloned_repository: Ref<Repository> = ref(new Repository())
    async function show(repository: Repository): Promise<void> {
        cloned_repository.value = repository
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        cloned_repository.value = new Repository()
    }

    return {
        is_show_dialog,
        ui,
        cloned_repository,
        show,
        hide,
    }
}
