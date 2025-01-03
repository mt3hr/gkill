<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddRepView :application_config="application_config" :gkill_api="gkill_api" :server_configs="server_configs"
            :account="cloned_account" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_rep="(rep) => emits('requested_add_rep', rep)" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddRepDialogEmits } from './add-rep-dialog-emits'
import type { AddRepDialogProps } from './add-rep-dialog-props'
import AddRepView from '../views/add-rep-view.vue'
import { Account } from '@/classes/datas/config/account';

defineProps<AddRepDialogProps>()
const emits = defineEmits<AddRepDialogEmits>()
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
