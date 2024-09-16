<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmGenerateTLSFilesView :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmGenerateTLSFilesDialogEmits } from './confirm-generate-tls-files-dialog-emits';
import type { ConfirmGenerateTLSFilesDialogProps } from './confirm-generate-tls-files-dialog-props';
import ConfirmGenerateTLSFilesView from '../views/confirm-generate-tls-files-view.vue';

const props = defineProps<ConfirmGenerateTLSFilesDialogProps>();
const emits = defineEmits<ConfirmGenerateTLSFilesDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
