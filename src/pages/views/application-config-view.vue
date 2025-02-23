<template>
    <v-card>
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>設定</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="reload_repositories()" color="'primary'">再読込</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="logout()" color="'primary'">ログアウト</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn v-if="cloned_application_config.account_is_admin" @click="show_server_config_dialog()"
                        color="'primary'">サーバ設定</v-btn>
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
                        <v-checkbox v-model="is_dark_theme" hide-detail label="ダークテーマ" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="rykv_hot_reload" hide-detail label="ホットリロード" />
                    </td>
                </tr>
                <tr>
                    <td>
                        URLogブックマークレット
                    </td>
                    <td>
                        <v-text-field width="400" v-model="urlog_bookmarklet" readonly
                            @focus="$event.target.select()"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="is_checked_use_rykv_period" hide-detail label="rykv表示日数" />
                    </td>
                    <td v-show="rykv_default_period !== -1">
                        <v-text-field type="number" min="-1" width="400" v-model="rykv_default_period" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="is_checked_use_mi_period" hide-detail label="mi表示日数" />
                    </td>
                    <td v-show="mi_default_period !== -1">
                        <v-text-field type="number" min="-1" width="400" v-model="mi_default_period" />
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
import { UpdateApplicationConfigRequest } from '@/classes/api/req_res/update-application-config-request'
import ServerConfigDialog from '../dialogs/server-config-dialog.vue'
import { LogoutRequest } from '@/classes/api/req_res/logout-request'
import router from '@/router'
import { ReloadRepositoriesRequest } from '@/classes/api/req_res/reload-repositories-request'
import { useTheme } from 'vuetify'

const theme = useTheme()

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

const is_loading = ref(false)

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
const rykv_image_list_column_number: Ref<number> = ref(cloned_application_config.value.rykv_image_list_column_number)
const enable_browser_cache: Ref<boolean> = ref(cloned_application_config.value.enable_browser_cache)
const rykv_hot_reload: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)
const mi_default_board: Ref<string> = ref(cloned_application_config.value.mi_default_board)
const mi_board_names: Ref<Array<string>> = ref(new Array())
const rykv_default_period: Ref<number> = ref(cloned_application_config.value.rykv_default_period)
const mi_default_period: Ref<number> = ref(cloned_application_config.value.mi_default_period)
const is_checked_use_rykv_period: Ref<boolean> = ref(cloned_application_config.value.rykv_default_period !== -1)
const is_checked_use_mi_period: Ref<boolean> = ref(cloned_application_config.value.mi_default_period !== -1)
const is_dark_theme: Ref<boolean> = ref(theme.global.name.value === 'gkill_dark_theme')

watch(() => is_checked_use_rykv_period.value, () => {
    if (is_checked_use_rykv_period.value) {
        rykv_default_period.value = 31
    } else {
        rykv_default_period.value = -1
    }
})

watch(() => is_checked_use_mi_period.value, () => {
    if (is_checked_use_mi_period.value) {
        mi_default_period.value = 31
    } else {
        mi_default_period.value = -1
    }
})

watch(() => is_dark_theme.value, () => {
    if (is_dark_theme.value) {
        theme.global.name.value = 'gkill_dark_theme'
    } else {
        theme.global.name.value = 'gkill_theme'
    }
})

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    google_map_api_key.value = cloned_application_config.value.google_map_api_key
    rykv_image_list_column_number.value = cloned_application_config.value.rykv_image_list_column_number
    enable_browser_cache.value = cloned_application_config.value.enable_browser_cache
    rykv_hot_reload.value = cloned_application_config.value.rykv_hot_reload
    mi_default_board.value = cloned_application_config.value.mi_default_board
    mi_board_names.value = new Array()
    rykv_default_period.value = cloned_application_config.value.rykv_default_period
    mi_default_period.value = cloned_application_config.value.mi_default_period
    load_mi_board_names()
}

async function load_mi_board_names(): Promise<void> {
    const req = new GetMiBoardRequest()

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
    application_config.rykv_image_list_column_number = parseInt(rykv_image_list_column_number.value.toString())
    application_config.enable_browser_cache = enable_browser_cache.value
    application_config.rykv_hot_reload = rykv_hot_reload.value
    application_config.mi_default_board = mi_default_board.value
    application_config.rykv_default_period = rykv_default_period.value
    application_config.mi_default_period = mi_default_period.value

    const req = new UpdateApplicationConfigRequest()
    req.application_config = application_config

    const res = await props.gkill_api.update_application_config(req)
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

async function logout(): Promise<void> {
    const req = new LogoutRequest()
    const res = await props.gkill_api.logout(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    await sleep(1500)

    deleteAllCookies()

    router.replace("/")
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

async function reload_repositories(): Promise<void> {
    is_loading.value = true
    const req = new ReloadRepositoriesRequest()
    const res = await props.gkill_api.reload_repositories(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    is_loading.value = false
}

const urlog_bookmarklet: Ref<string> = ref((`
javascript: (function () {
	function genURLog() {
		let description = '';
		let image_url = '';
		
		if (new URL(location.href).host == "www.youtube.com") {
			let youtubeDescriptionTag = document.querySelector('#description > yt-formatted-string');
			if (youtubeDescriptionTag !== null) {
				description = youtubeDescriptionTag.textContent;
			}
		}
		if (description == '') {
			let descriptionTag = document.querySelector("meta[name='description']");
			if (descriptionTag !== null) {
				description = descriptionTag.getAttribute('content');
			} else {
				descriptionTag = document.querySelector("meta[property='og:description']");
				if (descriptionTag !== null) {
					description = descriptionTag.getAttribute('content');
				}
			}
		}

		if (new URL(location.href).host == "www.amazon.co.jp" || new URL(location.href).host == "www.amazon.com") {
			let amazonImageTag = document.querySelector('#landingImage');
			if (amazonImageTag !== null) {
				image_url = amazonImageTag.getAttribute('src');
			}
		}
		if (image_url == '') {
			let imageOGTag = document.querySelector('meta[property="og:image"]');
			if (imageOGTag !== null) {
				image_url = imageOGTag.getAttribute('content');
			}
		}

		return {
			url: location.href,
			title: document.title,
			time: new Date().toISOString(),
			favicon_url: 'http://www.google.com/s2/favicons?domain=' + new URL(location.href).host,
			description: description,
			image_url: image_url,
			session_id: '`+ props.gkill_api.get_session_id() + `',
		};
	};
	function sendURLog() {
		let urlog = JSON.stringify(genURLog());
		fetch('`  + location.protocol + "//" + location.host + props.gkill_api.urlog_bookmarklet_address + `', {
			method: '`+ props.gkill_api.urlog_bookmarklet_method + `',
            mode: 'no-cors',
			headers: { 'Content-Type': 'application/json' },
				body: urlog
			}
		)
	};
	addEventListener('onload', sendURLog());
}());`).replace("\n", "").replace("\t", ""))

load_mi_board_names()

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

function deleteAllCookies() {
    const cookies = document.cookie.split(';')
    for (let i = 0; i < cookies.length; i++) {
        const cookie = cookies[i]
        const eqPos = cookie.indexOf('=')
        const name = eqPos > -1 ? cookie.substr(0, eqPos) : cookie
        document.cookie = name + '=;max-age=0'
    }
}
</script>
