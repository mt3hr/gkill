'use strict'

import { computed, ref, type Ref } from 'vue'
import type { MKFLDialogProps } from '@/pages/dialogs/mkfl-dialog-props'
import type { MKFLDialogEmits } from '@/pages/dialogs/mkfl-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useMKFLDialog(options: {
    props: MKFLDialogProps
    emits: MKFLDialogEmits
}) {
    const { props } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("mkfl-dialog", {
        centerMode: "always",
    })

    const view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.85, 600))
    const view_height = computed(() => props.app_content_height.valueOf() * 0.85)

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        view_width,
        view_height,
        show,
        hide,
    }
}
