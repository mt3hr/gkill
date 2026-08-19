'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import FindQueryEditorDialog from '@/pages/dialogs/find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from '@/pages/dialogs/mi-find-query-editor-dialog.vue'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import { ref, type Ref } from 'vue'
import type { EditDashboardDialogProps } from '@/pages/dialogs/edit-dashboard-dialog-props'
import type { EditDashboardDialogEmits } from '@/pages/dialogs/edit-dashboard-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export function useEditDashboardDialog(options: {
    props: EditDashboardDialogProps
    emits: EditDashboardDialogEmits
}) {
    const { emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-dashboard-dialog", {
        centerMode: "always",
    })

    const current_dnote_query = ref<FindKyouQuery>(new FindKyouQuery())
    const current_mi_query = ref<FindKyouQuery>(new FindKyouQuery())

    async function show(
        initial_dnote_query?: FindKyouQuery,
        initial_mi_query?: FindKyouQuery
    ): Promise<void> {
        current_dnote_query.value = initial_dnote_query ?? new FindKyouQuery()
        current_mi_query.value = initial_mi_query ?? new FindKyouQuery()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const dnote_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null)
    const mi_query_editor_dialog = ref<InstanceType<typeof MiFindQueryEditorDialog> | null>(null)
    function open_dnote_query_editor(): void {
            dnote_query_editor_dialog.value?.show(current_dnote_query.value)
    }
    function open_mi_query_editor(): void {
            mi_query_editor_dialog.value?.show(current_mi_query.value)
    }
    // クエリエディタの適用はローカル反映のみ。永続化はこのダイアログの保存で確定する
    // （エディタで適用した時点で親に伝えてしまうと、ここでキャンセルしても戻らなくなる）
    function onAppliedDnoteQuery(query: FindKyouQuery): void {
            current_dnote_query.value = query
    }
    function onAppliedMiQuery(query: FindKyouQuery): void {
            current_mi_query.value = query
    }
    function emit_current_config(): void {
            const config = new DashboardConfig()
            config.dashboard_dnote_find_kyou_query = current_dnote_query.value
            config.dashboard_mi_find_kyou_query = current_mi_query.value
            emits('requested_apply_dashboard_struct', config.to_json())
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
        dnote_query_editor_dialog,
        mi_query_editor_dialog,
        open_dnote_query_editor,
        open_mi_query_editor,
        onAppliedDnoteQuery,
        onAppliedMiQuery,
        onSave,
        onCancel,
        is_show_dialog,
        ui,
        current_dnote_query,
        current_mi_query,
        show,
        hide,
    }
}
