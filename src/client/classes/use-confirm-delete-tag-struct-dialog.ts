'use strict'

import { type Ref, ref } from 'vue'
import type { ConfirmDeleteTagStructDialogEmits } from '@/pages/dialogs/confirm-delete-tag-struct-dialog-emits';
import type { ConfirmDeleteTagStructDialogProps } from '@/pages/dialogs/confirm-delete-tag-struct-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data';
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useConfirmDeleteTagStructDialog(options: {
    props: ConfirmDeleteTagStructDialogProps
    emits: ConfirmDeleteTagStructDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const tag_struct: Ref<TagStructElementData> = ref(new TagStructElementData())
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("confirm-delete-tag-struct-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(tag_struct_obj: TagStructElementData): Promise<void> {
        tag_struct.value = tag_struct_obj
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        tag_struct.value = new TagStructElementData()
    }

    return {
        tag_struct,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
