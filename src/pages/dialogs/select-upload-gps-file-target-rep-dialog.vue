<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <SelectUploadGPSFileTargetRepView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_upload_gps_file_target_rep="(target_rep) => emits('requested_update_upload_gps_file_target_rep', target_rep)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { SelectUploadGPSFileTargetRepDialogEmits } from './select-upload-gps-file-target-rep-dialog-emits'
import type { SelectUploadGPSFileTargetRepDialogProps } from './select-upload-gps-file-target-rep-dialog-props'
import SelectUploadGPSFileTargetRepView from '../views/select-upload-gps-file-target-rep-view.vue'

defineProps<SelectUploadGPSFileTargetRepDialogProps>()
const emits = defineEmits<SelectUploadGPSFileTargetRepDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
