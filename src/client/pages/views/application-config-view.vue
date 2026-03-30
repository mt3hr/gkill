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
                    <v-btn dark color="primary" @click="reload_repositories(false)"
                        v-long-press="() => reload_repositories(true)">{{ i18n.global.t("RELOAD_TITLE")
                        }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="() => logout(false)" v-long-press="() => logout(true)">{{
                        i18n.global.t("LOGOUT_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr>
                    <td>
                        {{ i18n.global.t("LOCALE_TITLE") }}
                    </td>
                    <td>
                        <v-row class="pa-0 ma-0">
                            <v-col class="pa-0 ma-0">
                                <v-select class="select" v-model="locale_name" :items="[
                                    { locale_title: i18n.global.t('LOCALE_JA_TITLE'), locale_name: 'ja' },
                                    { locale_title: i18n.global.t('LOCALE_EN_TITLE'), locale_name: 'en' },
                                    { locale_title: i18n.global.t('LOCALE_ZH_TITLE'), locale_name: 'zh' },
                                    { locale_title: i18n.global.t('LOCALE_KO_TITLE'), locale_name: 'ko' },
                                    { locale_title: i18n.global.t('LOCALE_ES_TITLE'), locale_name: 'es' },
                                    { locale_title: i18n.global.t('LOCALE_FR_TITLE'), locale_name: 'fr' },
                                    { locale_title: i18n.global.t('LOCALE_DE_TITLE'), locale_name: 'de' },
                                ]" item-title="locale_title" item-value="locale_name" />
                            </v-col>
                        </v-row>
                    </td>
                </tr>
            </table>
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
            </table>
            <table>
                <tr>
                    <td>
                        <v-checkbox v-model="is_checked_use_rykv_period" hide-detail
                            :label="i18n.global.t('RYKV_DEFAULT_PERIOD_TITLE')" />
                    </td>
                    <td v-show="rykv_default_period !== -1">
                        <v-text-field type="number" min="-1" v-model="rykv_default_period" />
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
                        <v-text-field type="number" min="1" max="10" v-model="rykv_image_list_column_number" />
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
                        <v-text-field v-model="urlog_bookmarklet" readonly
                            @focus="$event.target.select()"></v-text-field>
                    </td>
                </tr>
                <tr>
                    <td>
                        {{ i18n.global.t("GOOGLE_MAP_API_KEY_TITLE") }}
                    </td>
                    <td>
                        <v-text-field v-model="google_map_api_key"></v-text-field>
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
            <a href="https://github.com/mt3hr/gkill" style="color:inherit;text-decoration:none;" target="_blank">
                <p class="gkill_version_info">gkill v-{{ application_config.version }} ({{
                    application_config.build_time.toLocaleString()
                    }})</p>
                <p class="gkill_version_info">{{ application_config.commit_hash }}</p>
            </a>
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
        <EditDeviceStructDialog :application_config="cloned_application_config" :folder_name="''" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_device_struct="(data: DeviceStructElementData) => onRequestedApplyDeviceStruct(data)"
            ref="edit_device_struct_dialog" />
        <EditKFTLTemplateDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_kftl_template_struct="(data: KFTLTemplateStructElementData) => onRequestedApplyKftlTemplateStruct(data)"
            @requested_reload_application_config="() => () => { }" ref="edit_kftl_template_dialog" />
        <EditRepStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_rep_struct="(data: RepStructElementData) => onRequestedApplyRepStruct(data)"
            @requested_reload_application_config="() => () => { }"
            ref="edit_rep_struct_dialog" />
        <EditRepTypeStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_rep_type_struct="(data: RepTypeStructElementData) => onRequestedApplyRepTypeStruct(data)"
            @requested_reload_application_config="() => () => { }"
            ref="edit_rep_type_struct_dialog" />
        <EditTagStructDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_tag_struct="(data: TagStructElementData) => onRequestedApplyTagStruct(data)"
            @requested_reload_application_config="() => { }" ref="edit_tag_struct_dialog" />
        <EditDnoteDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_dnote="(data: DnoteData) => onRequestedApplyDnote(data)"
            @requested_reload_application_config="() => { }" ref="edit_dnote_dialog" />
        <EditRyuuDialog v-model="cloned_application_config" :app_content_height="app_content_height"
            :app_content_width="app_content_width" :application_config="cloned_application_config"
            :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_apply_ryuu_struct="(data: RyuuData) => onRequestedApplyRyuuStruct(data)"
            @requested_reload_application_config="() => { }" ref="edit_ryuu_dialog" />
        <NewBoardNameDialog :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
        <ServerConfigDialog :application_config="cloned_application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="server_config_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'

import EditDeviceStructDialog from '../dialogs/edit-device-struct-dialog.vue'
import EditKFTLTemplateDialog from '../dialogs/edit-kftl-template-struct-dialog.vue'
import EditRepStructDialog from '../dialogs/edit-rep-struct-dialog.vue'
import EditRepTypeStructDialog from '../dialogs/edit-rep-type-struct-dialog.vue'
import EditTagStructDialog from '../dialogs/edit-tag-struct-dialog.vue'
import EditDnoteDialog from '../dialogs/edit-dnote-dialog.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import ServerConfigDialog from '../dialogs/server-config-dialog.vue'
import EditRyuuDialog from '../dialogs/edit-ryuu-dialog.vue'

import type { ApplicationConfigViewEmits } from './application-config-view-emits'
import type { ApplicationConfigViewProps } from './application-config-view-props'
import { useApplicationConfigView } from '@/classes/use-application-config-view'
import type { DeviceStructElementData } from "@/classes/datas/config/device-struct-element-data"
import type { KFTLTemplateStructElementData } from "@/classes/datas/config/kftl-template-struct-element-data"
import type { RepStructElementData } from "@/classes/datas/config/rep-struct-element-data"
import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data"
import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"
type DnoteData = Record<string, unknown>
type RyuuData = Record<string, unknown>

const props = defineProps<ApplicationConfigViewProps>()
const emits = defineEmits<ApplicationConfigViewEmits>()

const {
    // Template refs
    new_board_name_dialog,
    edit_device_struct_dialog,
    edit_rep_struct_dialog,
    edit_rep_type_struct_dialog,
    edit_tag_struct_dialog,
    edit_kftl_template_dialog,
    edit_dnote_dialog,
    edit_ryuu_dialog,
    server_config_dialog,

    // State
    is_loading,
    cloned_application_config,
    locale_name,
    google_map_api_key,
    rykv_image_list_column_number,
    rykv_hot_reload,
    show_tags_in_list,
    mi_default_board,
    mi_board_names,
    rykv_default_period,
    is_checked_use_rykv_period,
    use_dark_theme,
    is_show_share_footer,
    default_page,
    pages,

    // Computed
    urlog_bookmarklet,

    // Business logic
    reload_cloned_application_config,
    update_application_config,
    logout,
    reload_repositories,

    // Dialog show methods
    show_edit_device_dialog,
    show_edit_rep_dialog,
    show_edit_tag_dialog,
    show_edit_rep_type_dialog,
    show_edit_kftl_template_dialog,
    show_edit_dnote_dialog,
    show_edit_ryuu_dialog,
    show_new_board_name_dialog,
    show_server_config_dialog,

    // Event handlers
    update_board_name,
    onRequestedApplyDeviceStruct,
    onRequestedApplyKftlTemplateStruct,
    onRequestedApplyRepStruct,
    onRequestedApplyRepTypeStruct,
    onRequestedApplyTagStruct,
    onRequestedApplyDnote,
    onRequestedApplyRyuuStruct,

    // Event relay objects
    errorMessageRelayHandlers,
} = useApplicationConfigView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.gkill_version_info {
    text-align: right;
    font-size: x-small;
}
</style>
