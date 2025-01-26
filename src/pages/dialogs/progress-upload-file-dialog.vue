<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog" persistent>
        <ProgressUploadFileView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ProgressUploadFileDialogEmits } from './progress-upload-file-dialog-emits'
import type { ProgressUploadFileDialogProps } from './progress-upload-file-dialog-props'
import ProgressUploadFileView from '../views/progress-upload-file-view.vue'

defineProps<ProgressUploadFileDialogProps>()
const emits = defineEmits<ProgressUploadFileDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
