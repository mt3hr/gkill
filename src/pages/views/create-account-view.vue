<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>アカウント追加</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-model="new_user_id" label="ユーザID" />
        <v-checkbox v-model="do_initialize" label="ユーザ用の各Repを作成し設定する" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="create_account">アカウント作成</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { CreateAccountViewEmits } from './create-account-view-emits'
import type { CreateAccountViewProps } from './create-account-view-props'
import { AddAccountRequest } from '@/classes/api/req_res/add-account-request';
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetServerConfigRequest } from '@/classes/api/req_res/get-server-config-request';
import type { Account } from '@/classes/datas/config/account';

const props = defineProps<CreateAccountViewProps>()
const emits = defineEmits<CreateAccountViewEmits>()
const new_user_id: Ref<string> = ref("")
const do_initialize: Ref<boolean> = ref(false)

async function create_account(): Promise<void> {
    const req = new AddAccountRequest()
    req.account_info.is_enable = true
    req.account_info.is_admin = false
    req.account_info.password_reset_token = null
    req.account_info.user_id = new_user_id.value
    req.do_initialize = do_initialize.value
    req.session_id = GkillAPI.get_instance().get_session_id()

    const res = await GkillAPI.get_instance().add_account(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const server_config_req = new GetServerConfigRequest()
    server_config_req.session_id = GkillAPI.get_instance().get_session_id()
    const server_config_res = await GkillAPI.get_instance().get_server_config(server_config_req)
    if (server_config_res.errors && server_config_res.errors.length !== 0) {
        emits('received_errors', server_config_res.errors)
        return
    }
    if (server_config_res.messages && server_config_res.messages.length !== 0) {
        emits('received_messages', server_config_res.messages)
    }

    emits('created_account', res.added_account_info)
    emits('requested_reload_server_config', server_config_res.server_config)
    emits('requested_close_dialog')
}
</script>
