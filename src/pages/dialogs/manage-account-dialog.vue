<template>
    <v-dialog v-model="is_show_dialog">
        <ManageAccountView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ManageAccountDialogEmits } from './manage-account-dialog-emits'
import type { ManageAccountDialogProps } from './manage-account-dialog-props'
import ManageAccountView from '../views/manage-account-view.vue'

defineProps<ManageAccountDialogProps>()
const emits = defineEmits<ManageAccountDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
