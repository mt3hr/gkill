'use strict'

import { type Ref, ref } from 'vue'
import type { AddNewFolderDialogEmits } from '@/pages/dialogs/add-new-folder-dialog-emits'
import type { AddNewFolderDialogProps } from '@/pages/dialogs/add-new-folder-dialog-props'
import AddNewFolderView from '@/pages/views/add-new-folder-view.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useAddNewFolderDialog(options: {
    props: AddNewFolderDialogProps
    emits: AddNewFolderDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const add_new_folder_view = ref<InstanceType<typeof AddNewFolderView> | null>(null);
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    // キーは旧ファイル名綴りのまま(localStorageのダイアログ位置・透過設定の互換維持)
    const ui = useFloatingDialog("add-new-folder-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        add_new_folder_view.value?.reset_folder_name()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        add_new_folder_view.value?.reset_folder_name()
    }

    return {
        add_new_folder_view,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
