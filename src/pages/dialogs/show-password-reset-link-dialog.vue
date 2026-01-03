<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ShowPasswordResetLinkView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" :account="cloned_account"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide" />
    </v-dialog>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ShowPasswordResetLinkDialogEmits } from './show-password-reset-link-dialog-emits'
import type { ShowPasswordResetLinkDialogProps } from './show-password-reset-link-dialog-props'
import ShowPasswordResetLinkView from '../views/show-password-reset-link-view.vue'
import { Account } from '@/classes/datas/config/account';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ShowPasswordResetLinkDialogProps>()
const emits = defineEmits<ShowPasswordResetLinkDialogEmits>()
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
