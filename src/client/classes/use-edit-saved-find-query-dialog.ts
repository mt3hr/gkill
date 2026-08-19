'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import EditSavedFindQueryListDialog from '@/pages/dialogs/edit-saved-find-query-list-dialog.vue'
import { ref, type Ref } from 'vue'
import type { EditSavedFindQueryDialogProps } from '@/pages/dialogs/edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from '@/pages/dialogs/edit-saved-find-query-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'

export function useEditSavedFindQueryDialog(options: {
    props: EditSavedFindQueryDialogProps
    emits: EditSavedFindQueryDialogEmits
}) {
    const { emits } = options

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

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const rykv_list_dialog = ref<InstanceType<typeof EditSavedFindQueryListDialog> | null>(null)
    const mi_list_dialog = ref<InstanceType<typeof EditSavedFindQueryListDialog> | null>(null)
    function open_rykv_list_dialog(): void {
            rykv_list_dialog.value?.show(current_config.value.saved_rykv_find_kyou_querys)
    }
    function open_mi_list_dialog(): void {
            mi_list_dialog.value?.show(current_config.value.saved_mi_find_kyou_querys)
    }
    // 一覧ダイアログの適用はローカル反映のみ。永続化はこのダイアログの保存で親の clone へ渡し、
    // 設定画面全体の「適用」で確定する
    // (一覧側で適用した時点で親に伝えてしまうと、ここでキャンセルしても戻らなくなる)
    function onAppliedRykvItems(items: Array<SavedFindQueryItem>): void {
            current_config.value.saved_rykv_find_kyou_querys = items
    }
    function onAppliedMiItems(items: Array<SavedFindQueryItem>): void {
            current_config.value.saved_mi_find_kyou_querys = items
    }
    function emit_current_config(): void {
            emits('requested_apply_saved_find_query_struct', current_config.value.to_json())
    }
    function onSave(): void {
            emit_current_config()
            hide()
    }
    function onCancel(): void {
            hide()
    }

    return {
        help_dialog,
        rykv_list_dialog,
        mi_list_dialog,
        open_rykv_list_dialog,
        open_mi_list_dialog,
        onAppliedRykvItems,
        onAppliedMiItems,
        onSave,
        onCancel,
        is_show_dialog,
        ui,
        current_config,
        show,
        hide,
    }
}
