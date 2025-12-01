<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ManageAccountView :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { ManageAccountDialogEmits } from './manage-account-dialog-emits'
import type { ManageAccountDialogProps } from './manage-account-dialog-props'
import ManageAccountView from '../views/manage-account-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

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
