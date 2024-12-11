<template>
    <v-dialog v-model="is_show_dialog">
        <ServerConfigView :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_server_config="load_server_config()"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ServerConfigDialogEmits } from './server-config-dialog-emits'
import type { ServerConfigDialogProps } from './server-config-dialog-props'
import ServerConfigView from '../views/server-config-view.vue'
import { ServerConfig } from '@/classes/datas/config/server-config';
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetServerConfigRequest } from '@/classes/api/req_res/get-server-config-request';

const props = defineProps<ServerConfigDialogProps>()
const emits = defineEmits<ServerConfigDialogEmits>()
defineExpose({ show, hide })

const server_config: Ref<ServerConfig> = ref(new ServerConfig())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    load_server_config()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
async function load_server_config(): Promise<void> {
    const req = new GetServerConfigRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    const res = await GkillAPI.get_instance().get_server_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    server_config.value = res.server_config
}

</script>
