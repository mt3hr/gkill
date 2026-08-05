'use strict'

import { computed, ref, type Ref } from 'vue'
import type { ConfirmDeleteTagDialogProps } from '@/pages/dialogs/confirm-delete-tag-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useConfirmDeleteTagDialog(options: {
    props: ConfirmDeleteTagDialogProps
    emits: KyouDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("confirm-delete-tag-dialog", {
        centerMode: "always",
    })

    const tag_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.tag.generate_info_identifier()
        return [info_identifier]
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
        tag_highlight_targets,
        show,
        hide,
    }
}
