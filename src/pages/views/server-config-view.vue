<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>サーバ設定</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="show_manage_account_dialog">アカウント管理</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        <v-checkbox v-model="is_local_only_access" hide-detail label="ローカルアクセスのみ許可" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="enable_tls" hide-detail label="TLS有効" />
                    </td>
                    <v-btn color='primary' @click="show_confirm_generate_tls_files_dialog">オレオレTLSファイル生成</v-btn>
                </tr>
                <tr>
                    <td>
                        アドレス
                    </td>
                    <td>
                        <v-text-field width="400" v-model="address"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        TLS CERTファイル
                    </td>
                    <td>
                        <v-text-field width="400" v-model="tls_cert_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        TLS KEYファイル
                    </td>
                    <td>
                        <v-text-field width="400" v-model="tls_key_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        ディレクトリを開くコマンド
                    </td>
                    <td>
                        <v-text-field width="400" v-model="open_directory_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        ファイルを開くコマンド
                    </td>
                    <td>
                        <v-text-field width="400" v-model="open_file_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        URLogタイムアウト
                    </td>
                    <td>
                        <v-text-field width="400" type="number" min="5" v-model="urlog_timeout"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        URLog U/A
                    </td>
                    <td>
                        <v-text-field width="400" v-model="urlog_useragent"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        アップロード容量月額制限
                    </td>
                    <td>
                        <v-text-field width="400" v-model="upload_size_limit_month"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        ユーザデータディレクトリ
                    </td>
                    <td>
                        <v-text-field width="400" v-model="user_data_directory"></v-text-field>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="update_server_config">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <ConfirmGenerateTLSFilesDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="confirm_generate_tls_files_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="create_account_dialog" />
        <ManageAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="manage_account_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { ServerConfig } from '@/classes/datas/config/server-config'
import { computed, ref, watch, type Ref } from 'vue'
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ManageAccountDialog from '../dialogs/manage-account-dialog.vue'
import type { ServerConfigViewEmits } from './server-config-view-emits'
import type { ServerConfigViewProps } from './server-config-view-props'
import { GkillAPI } from '@/classes/api/gkill-api'
import { UpdateServerConfigRequest } from '@/classes/api/req_res/update-server-config-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

const confirm_generate_tls_files_dialog = ref<InstanceType<typeof ConfirmGenerateTLSFilesDialog> | null>(null);
const manage_account_dialog = ref<InstanceType<typeof ManageAccountDialog> | null>(null);

const props = defineProps<ServerConfigViewProps>()
const emits = defineEmits<ServerConfigViewEmits>()

const is_local_only_access: Ref<boolean> = ref(props.server_config.is_local_only_access)
const address: Ref<string> = ref(props.server_config.address)
const enable_tls: Ref<boolean> = ref(props.server_config.enable_tls)
const tls_cert_file: Ref<string> = ref(props.server_config.tls_cert_file)
const tls_key_file: Ref<string> = ref(props.server_config.tls_key_file)
const open_directory_command: Ref<string> = ref(props.server_config.open_directory_command)
const open_file_command: Ref<string> = ref(props.server_config.open_file_command)
const urlog_timeout: Ref<string> = ref(props.server_config.urlog_timeout)
const urlog_useragent: Ref<string> = ref(props.server_config.urlog_useragent)
const upload_size_limit_month: Ref<Number> = ref(props.server_config.upload_size_limit_month)
const user_data_directory: Ref<string> = ref(props.server_config.user_data_directory)

watch(() => props.server_config, () => {
    is_local_only_access.value = (props.server_config.is_local_only_access)
    address.value = (props.server_config.address)
    enable_tls.value = (props.server_config.enable_tls)
    tls_cert_file.value = (props.server_config.tls_cert_file)
    tls_key_file.value = (props.server_config.tls_key_file)
    open_directory_command.value = (props.server_config.open_directory_command)
    open_file_command.value = (props.server_config.open_file_command)
    urlog_timeout.value = (props.server_config.urlog_timeout)
    urlog_useragent.value = (props.server_config.urlog_useragent)
    upload_size_limit_month.value = (props.server_config.upload_size_limit_month)
    user_data_directory.value = (props.server_config.user_data_directory)
})

function show_manage_account_dialog(): void {
    manage_account_dialog.value?.show()
}

function show_confirm_generate_tls_files_dialog(): void {
    confirm_generate_tls_files_dialog.value?.show()
}

async function update_server_config(): Promise<void> {
    const gkill_info_req = new GetGkillInfoRequest()
    gkill_info_req.session_id = GkillAPI.get_instance().get_session_id()
    const gkill_info_res = await GkillAPI.get_instance().get_gkill_info(gkill_info_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }
    if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
        emits('received_messages', gkill_info_res.messages)
    }

    const server_config = new ServerConfig()
    server_config.enable_this_device = props.server_config.enable_this_device
    server_config.accounts = props.server_config.accounts
    server_config.repositories = props.server_config.repositories
    server_config.address = address.value
    server_config.device = gkill_info_res.device
    server_config.enable_tls = enable_tls.value
    server_config.is_local_only_access = is_local_only_access.value
    server_config.open_directory_command = open_directory_command.value
    server_config.open_file_command = open_file_command.value
    server_config.tls_cert_file = tls_cert_file.value
    server_config.tls_key_file = tls_key_file.value
    server_config.upload_size_limit_month = upload_size_limit_month.value
    server_config.urlog_timeout = urlog_timeout.value
    server_config.urlog_useragent = urlog_useragent.value
    server_config.user_data_directory = user_data_directory.value

    const req = new UpdateServerConfigRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.server_config = server_config

    const res = await GkillAPI.get_instance().update_server_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_reload_server_config', res.server_config)
    emits('requested_close_dialog')
}
</script>
