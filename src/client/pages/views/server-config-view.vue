<template>
    <v-card>
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("SERVER_CONFIG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_manage_account_dialog">{{
                        i18n.global.t("MANAGE_ACCOUNT_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        <v-select v-model="device" :items="devices" :label="i18n.global.t('PROFILE_TITLE')" />
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
                                        @click="delete_current_server_config()" dark>{{ i18n.global.t("DELETE_TITLE")
                                        }}</v-btn>
                                </td>
                            </tr>
                        </table>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="server_config.is_local_only_access" hide-detail
                            :label="i18n.global.t('USE_LOCAL_ACCSESS_ONLY_TITLE')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="server_config.use_gkill_notification" hide-detail
                            :label="i18n.global.t('USE_GKILL_NOTIFICATION_TITLE')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="server_config.enable_tls" hide-detail
                            :label="i18n.global.t('ENABLE_TLS_TITLE')" />
                    </td>
                    <v-btn dark color="primary" @click="show_confirm_generate_tls_files_dialog">{{
                        i18n.global.t("GENERATE_OREORE_TLS_TITLE") }}</v-btn>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("ADDRESS_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.address"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("TLS_CERT_FILE_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.tls_cert_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("TLS_KEY_FILE_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.tls_key_file"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("OPEN_DIRECTORY_COMMAND_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.open_directory_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("OPEN_FILE_COMMAND_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.open_file_command"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("URLOG_TIMEOUT_TITLE") }}
                    </td>
                    <td>
                        <v-text-field type="number" min="5" v-model="server_config.urlog_timeout"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("URLOG_USERAGENT_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.urlog_useragent"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("USER_DATA_DIRECTORY_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="server_config.user_data_directory"></v-text-field>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="update_server_config" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="onRequestedCloseDialog">{{
                        i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <ConfirmGenerateTLSFilesDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            ref="confirm_generate_tls_files_dialog" />
        <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            ref="create_account_dialog" />
        <ManageAccountDialog :application_config="application_config" :gkill_api="gkill_api"
            :server_configs="server_configs"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @requested_reload_server_config="() => emits('requested_reload_server_config')"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            ref="manage_account_dialog" />
        <NewDeviceNameDialog :gkill_api="gkill_api" :application_config="application_config"
            @setted_new_device_name="(device_name: string) => onSettedNewDeviceName(device_name)"
            ref="new_device_name_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ManageAccountDialog from '../dialogs/manage-account-dialog.vue'
import type { ServerConfigViewEmits } from './server-config-view-emits'
import type { ServerConfigViewProps } from './server-config-view-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import NewDeviceNameDialog from '../dialogs/new-device-name-dialog.vue'
import { useServerConfigView } from '@/classes/use-server-config-view'

const props = defineProps<ServerConfigViewProps>()
const emits = defineEmits<ServerConfigViewEmits>()

const {
    // Template refs
    confirm_generate_tls_files_dialog,
    manage_account_dialog,
    new_device_name_dialog,

    // State
    is_loading,
    cloned_server_configs,
    server_config,
    device,
    devices,

    // Template event handlers
    show_manage_account_dialog,
    show_confirm_generate_tls_files_dialog,
    show_new_device_name_dialog,
    update_server_config,
    delete_current_server_config,
    onRequestedCloseDialog,
    onSettedNewDeviceName,
} = useServerConfigView({ props, emits })
</script>
