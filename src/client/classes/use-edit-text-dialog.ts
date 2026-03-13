'use strict'

import { computed, ref, type Ref } from 'vue'
import type { EditTextDialogProps } from '@/pages/dialogs/edit-text-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useEditTextDialog(options: {
    props: EditTextDialogProps
    emits: KyouDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-text-dialog", {
        centerMode: "always",
    })

    const text_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifer = props.text.generate_info_identifer()
        return [info_identifer]
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
        emits('closed')
    }

    return {
        is_show_dialog,
        ui,
        text_highlight_targets,
        show,
        hide,
    }
}
