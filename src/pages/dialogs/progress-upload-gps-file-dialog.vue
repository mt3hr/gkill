<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ProgressUploadGPSFileView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @uploaded_kyous="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ProgressUploadGPSFileDialogEmits } from './progress-upload-gps-file-dialog-emits'
import type { ProgressUploadGPSFileDialogProps } from './progress-upload-gps-file-dialog-props'
import ProgressUploadGPSFileView from '../views/progress-upload-gps-file-view.vue'
defineProps<ProgressUploadGPSFileDialogProps>()
const emits = defineEmits<ProgressUploadGPSFileDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
