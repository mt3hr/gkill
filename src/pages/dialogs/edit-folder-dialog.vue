<template>
    <v-dialog v-model="is_show_dialog">
        <EditFolderView :application_config="application_config" :folder_name="folder_name" :gkill_api="gkill_api"
            :struct_obj="struct_obj" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_folder="(folder) => emits('requested_update_folder', folder)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditFolderDialogEmits } from './edit-folder-dialog-emits'
import type { EditFolderDialogProps } from './edit-folder-dialog-props'
import EditFolderView from '../views/edit-folder-view.vue'

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
