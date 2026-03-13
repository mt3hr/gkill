'use strict'

import { nextTick, ref, type Ref } from 'vue'
import type { EditDnoteDialogProps } from '@/pages/dialogs/edit-dnote-dialog-props'
import type { EditDnoteDialogEmits } from '@/pages/dialogs/edit-dnote-dialog-emits'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import type Dnote from '@/pages/views/dnote-view.vue'

export function useEditDnoteDialog(options: {
    props: EditDnoteDialogProps
    emits: EditDnoteDialogEmits
    dnote_view: Ref<InstanceType<typeof Dnote> | null>
}) {
    const { dnote_view } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-dnote-dialog", {
        centerMode: "always",
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
        nextTick(() => dnote_view.value?.reload([], new FindKyouQuery()))
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
