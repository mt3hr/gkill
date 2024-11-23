<template>
    <v-dialog v-model="is_show_dialog">
        <ServerConfigView :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ServerConfigDialogEmits } from './server-config-dialog-emits'
import type { ServerConfigDialogProps } from './server-config-dialog-props'
import ServerConfigView from '../views/server-config-view.vue'

const props = defineProps<ServerConfigDialogProps>()
const emits = defineEmits<ServerConfigDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
