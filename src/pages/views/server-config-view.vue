<template>
    <v-card>
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>サーバ設定</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_manage_account_dialog">アカウント管理</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        <v-select v-model="device" :items="devices" label="プロファイル" />
                    </td>
                    <td>
                        <table>
                            <tr>
                                <td>
                                    <v-btn color="primary" @click="show_new_device_name_dialog()" icon="mdi-plus" dark
                                        size="small"></v-btn>
                                </td>
                                <td>
                                    <v-btn color="secondary" v-if="cloned_server_configs.length >= 2"
                                        @click="delete_current_server_config()" dark>削除</v-btn>
                                </td>
                            </tr>
                        </table>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="server_config.is_local_only_access" hide-detail label="ローカルアクセスのみ許可" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="server_config.enable_tls" hide-detail label="TLS有効" />
                    </td>
                    <v-btn dark color="primary" @click="show_confirm_generate_tls_files_dialog">オレオレTLSファイル生成</v-btn>
                </tr>
                <tr>
                    <td>
                        アドレス
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.address"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        TLS CERTファイル
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.tls_cert_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        TLS KEYファイル
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.tls_key_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        ディレクトリを開くコマンド
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.open_directory_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        ファイルを開くコマンド
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.open_file_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        URLogタイムアウト
                    </td>
                    <td>
                        <v-text-field width="400" type="number" min="5"
                            v-model="server_config.urlog_timeout"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        URLog U/A
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.urlog_useragent"></v-text-field>
                    </td>
                </tr>
                <!--
                <tr>
                    <td>
                        アップロード容量月額制限
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.upload_size_limit_month"></v-text-field>
                    </td>
                </tr>
                -->
                <tr>
                    <td>
                        ユーザデータディレクトリ
                    </td>
                    <td>
                        <v-text-field width="400" v-model="server_config.user_data_directory"></v-text-field>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="update_server_config" color="primary">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <ConfirmGenerateTLSFilesDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="confirm_generate_tls_files_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages) => emits('received_messages', messages)" ref="create_account_dialog" />
        <ManageAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages) => emits('received_messages', messages)" ref="manage_account_dialog" />
        <NewDeviceNameDialog :gkill_api="gkill_api" :application_config="application_config"
            @setted_new_device_name="(device_name) => add_device(device_name)" ref="new_device_name_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { ServerConfig } from '@/classes/datas/config/server-config'
import { nextTick, ref, watch, type Ref } from 'vue'
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ManageAccountDialog from '../dialogs/manage-account-dialog.vue'
import type { ServerConfigViewEmits } from './server-config-view-emits'
import type { ServerConfigViewProps } from './server-config-view-props'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateServerConfigsRequest } from '@/classes/api/req_res/update-server-configs-request'
import NewDeviceNameDialog from '../dialogs/new-device-name-dialog.vue'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

const confirm_generate_tls_files_dialog = ref<InstanceType<typeof ConfirmGenerateTLSFilesDialog> | null>(null);
const manage_account_dialog = ref<InstanceType<typeof ManageAccountDialog> | null>(null);
const new_device_name_dialog = ref<InstanceType<typeof NewDeviceNameDialog> | null>(null);

const is_loading = ref(false)

const props = defineProps<ServerConfigViewProps>()
const emits = defineEmits<ServerConfigViewEmits>()

const cloned_server_configs: Ref<Array<ServerConfig>> = ref(props.server_configs.concat())
const server_config: Ref<ServerConfig> = ref(new ServerConfig())

const device: Ref<string> = ref("")
const devices: Ref<Array<string>> = ref(new Array<string>())

nextTick(() => {
    load_devices()
    device.value = props.server_configs.filter((server_cofnig) => server_cofnig.enable_this_device)[0].device
})

watch(() => props.server_configs, () => {
    cloned_server_configs.value = props.server_configs.concat()
    device.value = props.server_configs.filter((server_cofnig) => server_cofnig.enable_this_device)[0].device
    load_devices()
    load_current_server_config()
})

watch(() => device.value, () => {
    update_enable_device()
    load_current_server_config()
})

function update_enable_device(): void {
    for (let i = 0; i < cloned_server_configs.value.length; i++) {
        const server_config = cloned_server_configs.value[i]
        server_config.enable_this_device = server_config.device === device.value
    }
}

function add_device(device_name: string): void {
    for (let i = 0; i < cloned_server_configs.value.length; i++) {
        if (cloned_server_configs.value[i].device === device_name) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.device_is_aleady_exist
            error.error_message = "入力されたプロファイル名はすでに存在します"
            emits('received_errors', [error])
            return
        }
    }
    const new_server_config = new ServerConfig()
    new_server_config.is_local_only_access = server_config.value.is_local_only_access
    new_server_config.address = server_config.value.address
    new_server_config.enable_tls = server_config.value.enable_tls
    new_server_config.tls_cert_file = server_config.value.tls_cert_file
    new_server_config.tls_key_file = server_config.value.tls_key_file
    new_server_config.open_directory_command = server_config.value.open_directory_command
    new_server_config.open_file_command = server_config.value.open_file_command
    new_server_config.urlog_timeout = server_config.value.urlog_timeout
    new_server_config.urlog_useragent = server_config.value.urlog_useragent
    new_server_config.upload_size_limit_month = server_config.value.upload_size_limit_month
    new_server_config.user_data_directory = server_config.value.user_data_directory
    new_server_config.device = device_name
    cloned_server_configs.value.push(new_server_config)
    device.value = device_name
    load_devices()
}

function load_devices(): void {
    devices.value.splice(0)
    for (let i = 0; i < cloned_server_configs.value.length; i++) {
        devices.value.push(cloned_server_configs.value[i].device)
    }
}

async function load_current_server_config(): Promise<void> {
    let current_server_config: ServerConfig | null = null
    for (let i = 0; i < cloned_server_configs.value.length; i++) {
        if (cloned_server_configs.value[i].enable_this_device) {
            current_server_config = cloned_server_configs.value[i]
            break
        }
    }
    if (!current_server_config) {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.not_found_enable_device
        error.error_message = "有効なプロファイルが見つかりませんでした"
        emits('received_errors', [error])
        return
    }
    server_config.value = current_server_config
}

function show_manage_account_dialog(): void {
    manage_account_dialog.value?.show()
}

function show_confirm_generate_tls_files_dialog(): void {
    confirm_generate_tls_files_dialog.value?.show()
}

function show_new_device_name_dialog(): void {
    new_device_name_dialog.value?.show()
}

async function update_server_config(): Promise<void> {
    is_loading.value = true
    update_enable_device()

    const gkill_info_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(gkill_info_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }
    if (gkill_info_res.messages && gkill_info_res.messages.length !== 0) {
        emits('received_messages', gkill_info_res.messages)
    }

    const req = new UpdateServerConfigsRequest()
    req.server_configs = cloned_server_configs.value.concat()

    const res = await props.gkill_api.update_server_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    is_loading.value = false
    emits('requested_reload_server_config')
    emits('requested_close_dialog')
}

function delete_current_server_config(): void {
    let delete_target_server_config_index = -1
    for (let i = 0; i < cloned_server_configs.value.length; i++) {
        if (cloned_server_configs.value[i].device === device.value) {
            delete_target_server_config_index = i
            break
        }
    }
    if (delete_target_server_config_index === -1) {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.not_found_delete_target_device
        error.error_message = "削除対象のプロファイルが見つかりませんでした"
        emits('received_errors', [error])
        return
    }
    cloned_server_configs.value.splice(delete_target_server_config_index, 1)
    server_config.value = cloned_server_configs.value[0]
    device.value = server_config.value.device
    load_devices()
}
</script>
