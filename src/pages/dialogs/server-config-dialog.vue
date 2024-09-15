<template>
    <ServerConfigView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ServerConfigDialogEmits } from './server-config-dialog-emits';
import type { ServerConfigDialogProps } from './server-config-dialog-props';
import { ServerConfig } from '@/classes/datas/config/server-config';
import { GetServerConfigRequest } from '@/classes/api/req_res/get-server-config-request';
import ServerConfigView from '../views/server-config-view.vue';

const props = defineProps<ServerConfigDialogProps>();
const emits = defineEmits<ServerConfigDialogEmits>();
const cloned_server_config: Ref<ServerConfig> = ref(async () => {
    const req = new GetServerConfigRequest()
    req.session_id = ""//TODO セッションID取得
    const res = await props.gkill_api.get_server_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    return res.server_config
});
</script>