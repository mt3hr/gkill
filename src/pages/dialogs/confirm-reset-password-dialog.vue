<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmResetPasswordView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" :account="cloned_account"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @requested_show_show_password_reset_dialog="(account) => emits('requested_show_show_password_reset_dialog', account)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref, watch } from 'vue'
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits'
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props'
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue'
import { Account } from '@/classes/datas/config/account';

const props = defineProps<ConfirmResetPasswordDialogProps>()
const emits = defineEmits<ConfirmResetPasswordDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const cloned_account: Ref<Account> = ref(new Account())

async function show(account: Account): Promise<void> {
    cloned_account.value = account
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    cloned_account.value = new Account()
}
</script>
