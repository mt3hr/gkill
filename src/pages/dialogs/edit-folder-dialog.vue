<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditFolderView :application_config="application_config" :folder_name="folder_name" :gkill_api="gkill_api"
            :struct_obj="struct_obj"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_folder="(...folder: any[]) => emits('requested_update_folder', folder[0] as FolderStructElementData)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditFolderDialogEmits } from './edit-folder-dialog-emits'
import type { EditFolderDialogProps } from './edit-folder-dialog-props'
import EditFolderView from '../views/edit-folder-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'

defineProps<EditFolderDialogProps>()
const emits = defineEmits<EditFolderDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
