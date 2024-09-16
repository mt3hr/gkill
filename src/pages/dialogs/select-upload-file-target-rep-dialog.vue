<template>
    <v-dialog v-model="is_show_dialog">
        <SelectUploadFileTargetRepView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_upload_file_target_rep="(target_rep) => emits('requested_update_upload_file_target_rep', target_rep)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { SelectUploadFileTargetRepDialogEmits } from './select-upload-file-target-rep-dialog-emits';
import type { SelectUploadFileTargetRepDialogProps } from './select-upload-file-target-rep-dialog-props';
import SelectUploadFileTargetRepView from '../views/select-upload-file-target-rep-view.vue';

const props = defineProps<SelectUploadFileTargetRepDialogProps>();
const emits = defineEmits<SelectUploadFileTargetRepDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
