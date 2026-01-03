<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ServerConfigView v-show="server_configs.length !== 0" :application_config="application_config"
            :gkill_api="gkill_api" :server_configs="server_configs"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_server_config="load_server_configs()" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ServerConfigDialogEmits } from './server-config-dialog-emits'
import type { ServerConfigDialogProps } from './server-config-dialog-props'
import ServerConfigView from '../views/server-config-view.vue'
import { ServerConfig } from '@/classes/datas/config/server-config';
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<ServerConfigDialogProps>()
const emits = defineEmits<ServerConfigDialogEmits>()
defineExpose({ show, hide })

const server_configs: Ref<Array<ServerConfig>> = ref(new Array<ServerConfig>())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    load_server_configs()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
async function load_server_configs(): Promise<void> {
    server_configs.value.splice(0)
    const req = new GetServerConfigsRequest()
    const res = await props.gkill_api.get_server_configs(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    server_configs.value = res.server_configs
}
</script>
