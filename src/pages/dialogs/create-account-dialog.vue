<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <CreateAccountView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide" @created_account="(account) => emits('added_account', account)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { CreateAccountDialogEmits } from './create-account-dialog-emits'
import type { CreateAccountDialogProps } from './create-account-dialog-props'
import CreateAccountView from '../views/create-account-view.vue'

defineProps<CreateAccountDialogProps>()
const emits = defineEmits<CreateAccountDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
