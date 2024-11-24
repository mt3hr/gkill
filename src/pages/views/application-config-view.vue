<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>設定</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn v-if="is_admin" @click="show_server_config_dialog()" color="'primary'">サーバ設定</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        GoogleMapAPIキー
                    </td>
                    <td>
                        <v-text-field width="400" v-model="google_map_api_key"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        rykv画像ビューア列数
                    </td>
                    <td>
                        <v-text-field type="number" min="1" max="10" width="400"
                            v-model="rykv_image_list_column_number" />
                    </td>
                </tr>
                <tr>
                    <td>
                        miデフォルト板名
                    </td>
                    <td>
                        <v-row class="pa-0 ma-0">
                            <v-col class="pa-0 ma-0">
                                <v-select class="select" v-model="mi_default_board" :items="mi_board_names" />
                            </v-col>
                            <v-col class="pa-0 ma-0">
                                <v-btn color="primary" @click="show_new_board_name_dialog()" icon="mdi-plus" dark
                                    size="small"></v-btn>
                            </v-col>
                        </v-row>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="enable_browser_cache" hide-detail label="ブラウザキャッシュ有効" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="rykv_hot_reload" hide-detail label="rykvホットリロード" />
                    </td>
                </tr>

            </table>
            <table>
                <tr>
                    <td>
                        <v-btn @click="show_edit_tag_dialog" color="primary">タグ編集</v-btn>
                        <v-btn @click="show_edit_rep_dialog" color="primary">Rep編集</v-btn>
                        <v-btn @click="show_edit_device_dialog" color="primary">Device編集</v-btn>
                        <v-btn @click="show_edit_rep_type_dialog" color="primary">RepType編集</v-btn>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-btn @click="show_edit_kftl_template_dialog" color="primary">KFTLテンプレート編集</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="update_application_config">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <EditDeviceStructDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_device_struct_element="async () => emits('requested_reload_application_config')"
            ref="edit_device_struct_dialog" />
        <EditKFTLTemplateDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="() => emits('requested_reload_application_config')"
            ref="edit_kftl_template_dialog" />
        <EditRepStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config')"
            ref="edit_rep_struct_dialog" />
        <EditRepTypeStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config')"
            ref="edit_rep_type_struct_dialog" />
        <EditTagStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            ref="edit_tag_struct_dialog" />
        <NewBoardNameDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
        <ServerConfigDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="server_config_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'

import EditDeviceStructDialog from '../dialogs/edit-device-struct-dialog.vue'
import EditKFTLTemplateDialog from '../dialogs/edit-kftl-template-struct-dialog.vue'
import EditRepStructDialog from '../dialogs/edit-rep-struct-dialog.vue'
import EditRepTypeStructDialog from '../dialogs/edit-rep-type-struct-dialog.vue'
import EditTagStructDialog from '../dialogs/edit-tag-struct-dialog.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'

import type { ApplicationConfigViewEmits } from './application-config-view-emits'
import type { ApplicationConfigViewProps } from './application-config-view-props'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { GkillAPI } from '@/classes/api/gkill-api'
import { UpdateApplicationConfigRequest } from '@/classes/api/req_res/update-application-config-request'
import { ServerConfig } from '@/classes/datas/config/server-config'
import ServerConfigDialog from '../dialogs/server-config-dialog.vue'

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);
const edit_device_struct_dialog = ref<InstanceType<typeof EditDeviceStructDialog> | null>(null);
const edit_rep_struct_dialog = ref<InstanceType<typeof EditRepStructDialog> | null>(null);
const edit_rep_type_struct_dialog = ref<InstanceType<typeof EditRepTypeStructDialog> | null>(null);
const edit_tag_struct_dialog = ref<InstanceType<typeof EditTagStructDialog> | null>(null);
const edit_kftl_template_dialog = ref<InstanceType<typeof EditKFTLTemplateDialog> | null>(null);
const server_config_dialog = ref<InstanceType<typeof ServerConfigDialog> | null>(null);

const props = defineProps<ApplicationConfigViewProps>()
const emits = defineEmits<ApplicationConfigViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
    cloned_application_config.value.parse_template_and_struct()
})

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
const is_admin: Ref<boolean> = ref(cloned_application_config.value.account_is_admin)

const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
const rykv_image_list_column_number: Ref<Number> = ref(cloned_application_config.value.rykv_image_list_column_number)
const enable_browser_cache: Ref<boolean> = ref(cloned_application_config.value.enable_browser_cache)
const rykv_hot_reload: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)
const mi_default_board: Ref<string> = ref(cloned_application_config.value.mi_default_board)
const mi_board_names: Ref<Array<string>> = ref(new Array())

async function update_is_admin(): Promise<void> {
}

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    google_map_api_key.value = cloned_application_config.value.google_map_api_key
    rykv_image_list_column_number.value = cloned_application_config.value.rykv_image_list_column_number
    enable_browser_cache.value = cloned_application_config.value.enable_browser_cache
    rykv_hot_reload.value = cloned_application_config.value.rykv_hot_reload
    mi_default_board.value = cloned_application_config.value.mi_default_board
    mi_board_names.value = new Array()
    load_mi_board_names()
}

async function load_mi_board_names(): Promise<void> {
    const req = new GetMiBoardRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()

    const res = await props.gkill_api.get_mi_board_list(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        // emits('received_messages', res.messages)
    }
    mi_board_names.value = res.boards
}

async function update_application_config(): Promise<void> {
    const application_config = new ApplicationConfig()
    application_config.google_map_api_key = google_map_api_key.value
    application_config.rykv_image_list_column_number = rykv_image_list_column_number.value
    application_config.enable_browser_cache = enable_browser_cache.value
    application_config.rykv_hot_reload = rykv_hot_reload.value
    application_config.mi_default_board = mi_default_board.value

    const req = new UpdateApplicationConfigRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.application_config = application_config

    const res = await GkillAPI.get_instance().update_application_config(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_reload_application_config')
    emits('requested_close_dialog')
}

function show_edit_device_dialog() {
    edit_device_struct_dialog.value?.show()
}
function show_edit_rep_dialog() {
    edit_rep_struct_dialog.value?.show()
}
function show_edit_tag_dialog() {
    edit_tag_struct_dialog.value?.show()
}
function show_edit_rep_type_dialog() {
    edit_rep_type_struct_dialog.value?.show()
}
function show_edit_kftl_template_dialog() {
    edit_kftl_template_dialog.value?.show()
}
function show_new_board_name_dialog(): void {
    new_board_name_dialog.value?.show()
}
function update_board_name(board_name: string): void {
    mi_board_names.value.push(board_name)
    mi_default_board.value = board_name
}
function show_server_config_dialog(): void {
    server_config_dialog.value?.show()
}
//TODO コメントアウト解除 load_mi_board_names()
is_admin.value = true //TODO けして
</script>
