<template>
    <v-card>
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("APPLICATION_CONFIG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" v-if="cloned_application_config.account_is_admin"
                        @click="show_server_config_dialog()">{{ i18n.global.t("SERVER_CONFIG_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="reload_repositories()">{{ i18n.global.t("RELOAD_TITLE")
                        }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="logout()" color="primary">{{ i18n.global.t("LOGOUT_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        <v-checkbox v-model="use_dark_theme" hide-detail :label="i18n.global.t('DARK_THEME_TITLE')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="rykv_hot_reload" hide-detail :label="i18n.global.t('HOT_RELOAD_TITLE')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="show_tags_in_list" hide-detail
                            :label="i18n.global.t('SHOW_TAGS_IN_LIST')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="is_show_share_footer" hide-detail
                            :label="i18n.global.t('SHOW_SHARE_FOOTER')" />
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-checkbox v-model="is_checked_use_rykv_period" hide-detail
                            :label="i18n.global.t('RYKV_DEFAULT_PERIOD_TITLE')" />
                    </td>
                    <td v-show="rykv_default_period !== -1">
                        <v-text-field type="number" min="-1" width="400" v-model="rykv_default_period" />
                    </td>
                </tr>

                <tr>
                    <td>
                        {{ i18n.global.t("DEFAULT_VIEW_TITLE") }}
                    </td>
                    <td>
                        <v-row class="pa-0 ma-0">
                            <v-col class="pa-0 ma-0">
                                <v-select class="select" v-model="default_page" :items="pages" item-title="app_name"
                                    item-value="page_name" />
                            </v-col>
                        </v-row>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("RYKV_IMAGE_LIST_COLUMN_NUMBER_TITLE") }}
                    </td>
                    <td>
                        <v-text-field type="number" min="1" max="10" width="400"
                            v-model="rykv_image_list_column_number" />
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("MI_DEFAULT_BOARD_NAME_TITLE") }}
                    </td>
                    <td>
                        <v-row class="pa-0 ma-0">
                            <v-col class="pa-0 ma-0">
                                <v-select class="select" v-model="mi_default_board" :items="mi_board_names" />
                            </v-col>
                            <v-col class="pa-0 ma-0 pt-2">
                                <v-btn color="primary" @click="show_new_board_name_dialog()" icon="mdi-plus" dark
                                    size="small"></v-btn>
                            </v-col>
                        </v-row>
                    </td>
                </tr>

                <tr>
                    <td>
                        {{ i18n.global.t("URLOG_BOOKMARKLET_ADDRESS_TITLE") }}
                    </td>
                    <td>
                        <v-text-field width="400" v-model="urlog_bookmarklet" readonly
                            @focus="$event.target.select()"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("GOOGLE_MAP_API_KEY_TITLE") }}
                    </td>
                    <td>
                        <v-text-field width="400" v-model="google_map_api_key"></v-text-field>
                    </td>
                </tr>
            </table>
            <table>
                <tr>
                    <td>
                        <v-btn dark color="primary" @click="show_edit_tag_dialog">{{
                            i18n.global.t("EDIT_TAG_STRUCT_TITLE")
                            }}</v-btn>
                        <v-btn dark color="primary" @click="show_edit_rep_dialog">{{
                            i18n.global.t("EDIT_REP_STRUCT_TITLE")
                            }}</v-btn>
                        <v-btn dark color="primary" @click="show_edit_device_dialog">{{
                            i18n.global.t("EDIT_DEVICE_STRUCT_TITLE")
                            }}</v-btn>
                        <v-btn dark color="primary" @click="show_edit_rep_type_dialog">{{
                            i18n.global.t("EDIT_REP_TYPE_STRUCT_TITLE") }}</v-btn>
                    </td>
                </tr>
                <tr>
                    <td>
                        <v-btn dark color="primary" @click="show_edit_kftl_template_dialog">{{
                            i18n.global.t("EDIT_KFTL_TEMPLATE_STRUCT_TITLE") }}</v-btn>
                        <v-btn dark color="primary" @click="show_edit_dnote_dialog">{{
                            i18n.global.t("EDIT_DNOTE_TITLE") }}</v-btn>
                        <v-btn dark color="primary" @click="show_edit_ryuu_dialog">{{
                            i18n.global.t("EDIT_RYUU_TITLE") }}</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="update_application_config" color="primary">{{ i18n.global.t("APPLY_TITLE")
                        }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
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
        <EditDnoteDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            ref="edit_dnote_dialog" />
        <EditRyuuDialog v-model="cloned_application_config" :app_content_height="app_content_height"
            :app_content_width="app_content_width" :application_config="cloned_application_config"
            :gkill_api="gkill_api" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            ref="edit_ryuu_dialog" />
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
import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'

import EditDeviceStructDialog from '../dialogs/edit-device-struct-dialog.vue'
import EditKFTLTemplateDialog from '../dialogs/edit-kftl-template-struct-dialog.vue'
import EditRepStructDialog from '../dialogs/edit-rep-struct-dialog.vue'
import EditRepTypeStructDialog from '../dialogs/edit-rep-type-struct-dialog.vue'
import EditTagStructDialog from '../dialogs/edit-tag-struct-dialog.vue'
import EditDnoteDialog from '../dialogs/edit-dnote-dialog.vue'
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
import EditRyuuDialog from '../dialogs/edit-ryuu-dialog.vue'
import delete_gkill_kyou_cache, { delete_gkill_config_cache } from '@/classes/delete-gkill-cache'

const theme = useTheme()

const new_board_name_dialog = ref<InstanceType<typeof NewBoardNameDialog> | null>(null);
const edit_device_struct_dialog = ref<InstanceType<typeof EditDeviceStructDialog> | null>(null);
const edit_rep_struct_dialog = ref<InstanceType<typeof EditRepStructDialog> | null>(null);
const edit_rep_type_struct_dialog = ref<InstanceType<typeof EditRepTypeStructDialog> | null>(null);
const edit_tag_struct_dialog = ref<InstanceType<typeof EditTagStructDialog> | null>(null);
const edit_kftl_template_dialog = ref<InstanceType<typeof EditKFTLTemplateDialog> | null>(null);
const edit_dnote_dialog = ref<InstanceType<typeof EditDnoteDialog> | null>(null);
const edit_ryuu_dialog = ref<InstanceType<typeof EditRyuuDialog> | null>(null);
const server_config_dialog = ref<InstanceType<typeof ServerConfigDialog> | null>(null);
const pages = ref([
    { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
    { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
    { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
    { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
    { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
])

const props = defineProps<ApplicationConfigViewProps>()
const emits = defineEmits<ApplicationConfigViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, async () => {
    cloned_application_config.value = props.application_config.clone()
})

const is_loading = ref(false)

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
const rykv_image_list_column_number: Ref<number> = ref(cloned_application_config.value.rykv_image_list_column_number)
const rykv_hot_reload: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)
const show_tags_in_list: Ref<boolean> = ref(cloned_application_config.value.show_tags_in_list)
const mi_default_board: Ref<string> = ref(cloned_application_config.value.mi_default_board)
const mi_board_names: Ref<Array<string>> = ref(new Array())
const rykv_default_period: Ref<number> = ref(cloned_application_config.value.rykv_default_period)
const mi_default_period: Ref<number> = ref(cloned_application_config.value.mi_default_period)
const is_checked_use_rykv_period: Ref<boolean> = ref(cloned_application_config.value.rykv_default_period !== -1)
const is_checked_use_mi_period: Ref<boolean> = ref(cloned_application_config.value.mi_default_period !== -1)
const use_dark_theme: Ref<boolean> = ref(theme.global.name.value === 'gkill_dark_theme')
const is_show_share_footer: Ref<boolean> = ref(cloned_application_config.value.is_show_share_footer)
const default_page: Ref<string> = ref(cloned_application_config.value.default_page)

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

watch(() => use_dark_theme.value, () => {
    if (use_dark_theme.value) {
        theme.global.name.value = 'gkill_dark_theme'
    } else {
        theme.global.name.value = 'gkill_theme'
    }
})

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    google_map_api_key.value = cloned_application_config.value.google_map_api_key
    rykv_image_list_column_number.value = cloned_application_config.value.rykv_image_list_column_number
    rykv_hot_reload.value = cloned_application_config.value.rykv_hot_reload
    show_tags_in_list.value = cloned_application_config.value.show_tags_in_list
    mi_default_board.value = cloned_application_config.value.mi_default_board
    mi_board_names.value = new Array()
    rykv_default_period.value = cloned_application_config.value.rykv_default_period
    mi_default_period.value = cloned_application_config.value.mi_default_period
    is_show_share_footer.value = cloned_application_config.value.is_show_share_footer
    default_page.value = cloned_application_config.value.default_page
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
    application_config.rykv_hot_reload = rykv_hot_reload.value
    application_config.mi_default_board = mi_default_board.value
    application_config.rykv_default_period = rykv_default_period.value
    application_config.show_tags_in_list = show_tags_in_list.value
    application_config.mi_default_period = mi_default_period.value
    application_config.use_dark_theme = use_dark_theme.value
    application_config.is_show_share_footer = is_show_share_footer.value
    application_config.default_page = default_page.value
    application_config.ryuu_json_data = cloned_application_config.value.ryuu_json_data

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
    props.gkill_api.set_default_page_to_cookie(application_config.default_page)
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

    props.gkill_api.set_session_id("")

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
function show_edit_dnote_dialog() {
    edit_dnote_dialog.value?.show()
}
function show_edit_ryuu_dialog() {
    edit_ryuu_dialog.value?.show()
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
    await delete_gkill_config_cache()
    await delete_gkill_kyou_cache(null)
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
</script>
