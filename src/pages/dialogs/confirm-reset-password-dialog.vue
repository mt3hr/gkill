<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmResetPasswordView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" :account="cloned_account"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="hide"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @requested_show_show_password_reset_dialog="(...account: any[]) => emits('requested_show_show_password_reset_dialog', account[0] as Account)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits'
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props'
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue'
import { Account } from '@/classes/datas/config/account';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ConfirmResetPasswordDialogProps>()
const emits = defineEmits<ConfirmResetPasswordDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
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
