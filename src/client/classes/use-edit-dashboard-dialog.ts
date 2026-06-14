'use strict'

import { ref, type Ref } from 'vue'
import type { EditDashboardDialogProps } from '@/pages/dialogs/edit-dashboard-dialog-props'
import type { EditDashboardDialogEmits } from '@/pages/dialogs/edit-dashboard-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export function useEditDashboardDialog(_options: {
    props: EditDashboardDialogProps
    emits: EditDashboardDialogEmits
}) {
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
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        current_dnote_query,
        current_mi_query,
        show,
        hide,
    }
}
