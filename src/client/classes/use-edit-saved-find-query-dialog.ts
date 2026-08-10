'use strict'

import { ref, type Ref } from 'vue'
import type { EditSavedFindQueryDialogProps } from '@/pages/dialogs/edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from '@/pages/dialogs/edit-saved-find-query-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { SavedFindQueryConfig } from '@/classes/datas/config/saved-find-query-config'

export function useEditSavedFindQueryDialog(_options: {
    props: EditSavedFindQueryDialogProps
    emits: EditSavedFindQueryDialogEmits
}) {
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-saved-find-query-dialog", {
        centerMode: "always",
    })

    // キャンセルで破棄できるよう、show() で受け取った設定のクローンを編集する
    const current_config: Ref<SavedFindQueryConfig> = ref(new SavedFindQueryConfig())

    async function show(initial_config?: SavedFindQueryConfig): Promise<void> {
        current_config.value = initial_config?.clone() ?? new SavedFindQueryConfig()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        ui,
        current_config,
        show,
        hide,
    }
}
