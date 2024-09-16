<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmResetPasswordView :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" :user_id="user_id"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits';
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props';
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue';

const props = defineProps<ConfirmResetPasswordDialogProps>();
const emits = defineEmits<ConfirmResetPasswordDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
