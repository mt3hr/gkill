'use strict'

import { type Ref, ref } from 'vue'
import type { SelectMoveTargetFolderDialogEmits } from '@/pages/dialogs/select-move-target-folder-dialog-emits'
import type { SelectMoveTargetFolderDialogProps } from '@/pages/dialogs/select-move-target-folder-dialog-props'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'
import { list_move_target_folders, type MoveTargetFolderCandidate } from '@/classes/foldable-struct-move'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useSelectMoveTargetFolderDialog(options: {
    props: SelectMoveTargetFolderDialogProps
    emits: SelectMoveTargetFolderDialogEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    const moving_struct_id: Ref<string> = ref("")
    const folder_candidates: Ref<Array<MoveTargetFolderCandidate>> = ref([])
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("select-move-target-folder-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(root_struct_obj: FoldableStructModel, struct_id: string): Promise<void> {
        moving_struct_id.value = struct_id
        folder_candidates.value = list_move_target_folders(root_struct_obj, struct_id)
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }
    function select_folder(target_folder_id: string | null): void {
        emits('requested_move_struct_obj_to_folder', moving_struct_id.value, target_folder_id)
        hide()
    }

    return {
        is_show_dialog,
        folder_candidates,
        ui,
        show,
        hide,
        select_folder,
    }
}
