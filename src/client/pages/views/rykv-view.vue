<template>
    <div class="rykv_view_wrap" ref="rykv_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_SIDEBAR')">
                <template v-slot:activator="{ props }">
                    <v-app-bar-nav-icon v-bind="props" v-if="!is_shared_rykv_view" @click.stop="toggleDrawer"
                        :disabled="!inited" />
                </template>
            </v-tooltip>
            <v-toolbar-title>
                <div>
                    <span v-if="!is_shared_rykv_view">
                        {{ i18n.global.t("RYKV_APP_TITLE") }}
                    </span>
                    <span v-if="is_shared_rykv_view">
                        {{ share_title }}
                    </span>
                    <v-menu v-if="!is_shared_rykv_view" activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in page_list">
                                <v-list-item-title @click="navigateToPage(page.page_name)">
                                    {{ page.app_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_DETAIL_VIEW')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_kyou_detail_view = !is_show_kyou_detail_view">
                        <v-icon>mdi-file-document</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_DNOTE')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" v-if="!is_shared_rykv_view" icon
                        @click="toggleDnote">
                        <v-icon>mdi-file-chart-outline</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_CALENDAR')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_kyou_count_calendar = !is_show_kyou_count_calendar">
                        <v-icon>mdi-calendar</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_MAP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_gps_log_map = !is_show_gps_log_map">
                        <v-icon>mdi-map</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-divider vertical />
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" v-if="!is_shared_rykv_view" @click="help_dialog?.show()" />
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" v-if="!is_shared_rykv_view" icon="mdi-cog" :disabled="!application_config.is_loaded"
                        @click="emits('requested_show_application_config_dialog')" />
                </template>
            </v-tooltip>
        </v-app-bar>
        <v-navigation-drawer v-if="!is_shared_rykv_view" v-model="drawer" app :height="app_content_height"
            :touchless="!inited" :mobile="drawer_mode_is_mobile" :width="318">
            <RykvQueryEditorSideBar v-show="inited" class="rykv_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */"
                @requested_search="onSidebarRequestedSearch"
                @updated_query="onSidebarUpdatedQuery"
                @inited="onSidebarInited"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main" :class="(drawer_mode_is_mobile) ? 'scroll_snap_container' : ''">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="rykv_view_table" v-show="inited">
                <tr>
                    <td valign="top" v-for="query, index in querys" :key="query.query_id"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                            :application_config="application_config" :gkill_api="gkill_api"
                            :matched_kyous="match_kyous_list[index]" :query="query"
                            :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                            :enable_context_menu="!is_shared_rykv_view" :enable_dialog="!is_shared_rykv_view"
                            :is_readonly_mi_check="false" :show_checkbox="true" :show_footer="!is_shared_rykv_view"
                            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true"
                            :show_rep_name="!is_shared_rykv_view" :force_show_latest_kyou_info="true"
                            :show_content_only="false" :show_timeis_plaing_end_button="false"
                            v-on="crudRelayHandlers"
                            @scroll_list="(...args: any[]) => onColumnScrollList(index, args[0] as number)"
                            @clicked_list_view="() => onColumnClickedListView(index)"
                            @clicked_kyou="(...args: any[]) => onColumnClickedKyou(index, args[0] as Kyou)"
                            @focused_kyou="(...args: any[]) => onColumnClickedKyou(index, args[0] as Kyou)"
                            @requested_change_focus_kyou="(...args: any[]) => onColumnRequestedChangeFocusKyou(index, args[0] as boolean)"
                            @requested_search="() => onColumnRequestedSearch(index)"
                            @requested_change_is_image_only_view="(...args: any[]) => onColumnRequestedChangeImageOnlyView(index, args[0] as boolean)"
                            @requested_close_column="close_list_view(index)"
                            @requested_reload_kyou="(...args: any[]) => reload_kyou(args[0] as Kyou)"
                            @requested_reload_list="() => onColumnRequestedReloadList(index)"
                            @requested_open_rykv_dialog="(...args: any[]) => open_rykv_dialog(args[0], args[1], args[2])"
                            ref="kyou_list_views" />
                    </td>
                    <td valign="top" v-if="!is_shared_rykv_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <v-btn class="rykv_add_column_button rounded-sm mx-auto" :height="app_content_height.valueOf()"
                            :width="30" :color="'primary'" @click="onAddColumnClick" icon="mdi-plus" variant="text"
                            :style="{ background: 'rgb(var(--v-theme-background))' }" />
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                :is_image_request_to_thumb_size="false" :application_config="application_config"
                                :gkill_api="gkill_api" :highlight_targets="[]" :is_image_view="false"
                                :kyou="focused_kyou" :show_checkbox="false"
                                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="!is_shared_rykv_view"
                                :height="'auto'" :is_readonly_mi_check="is_shared_rykv_view" :width="'auto'"
                                :enable_context_menu="!is_shared_rykv_view" :enable_dialog="!is_shared_rykv_view"
                                :show_attached_timeis="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                class="kyou_detail_view" :show_update_time="false" :show_related_time="true"
                                :show_attached_tags="true" :show_attached_texts="true"
                                :show_attached_notifications="true"
                                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...rykvDialogHandler }" />
                        </div>
                        <div class="ryuu_view dummy">
                            <RyuuListView v-if="focused_kyou && default_query" :application_config="application_config"
                                :gkill_api="gkill_api" :target_kyou="focused_kyou" :editable="false"
                                :find_kyou_query_default="default_query"
                                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers }" />
                        </div>
                    </td>
                    <td valign="top" v-if="is_show_dnote && !is_shared_rykv_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <Dnote class="rykv_dnote_wrap" :app_content_height="app_content_height"
                            :app_content_width="app_content_width" :application_config="application_config"
                            :gkill_api="gkill_api" :query="focused_query" :checked_kyous="focused_column_checked_kyous"
                            :editable="false"
                            v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogHandler }"
                            ref="dnote_view" />
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="false"
                            @requested_focus_time="(...args: any[]) => onRequestedFocusTime(args[0] as Date)" />
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <GPSLogMap v-show="is_show_gps_log_map" :application_config="application_config"
                            :gkill_api="gkill_api" :start_date="gps_log_map_start_time" :end_date="gps_log_map_end_time"
                            :marker_time="gps_log_map_marker_time"
                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                            @requested_focus_time="(...args: any[]) => onGpsLogMapRequestedFocusTime(args[0] as Date)" />
                    </td>
                </tr>
            </table>
            <AddKCDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="crudRelayHandlers"
                ref="add_kc_dialog" />
            <AddTimeisDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_timeis_dialog" />
            <AddLantanaDialog v-if="!is_shared_rykv_view" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_lantana_dialog" />
            <AddUrlogDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_urlog_dialog" />
            <AddMiDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_mi_dialog" />
            <AddNlogDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_nlog_dialog" />
            <kftlDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="kftl_dialog" />
            <mkflDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="mkfl_dialog" />
            <UploadFileDialog v-if="!is_shared_rykv_view" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
                v-on="{ ...crudRelayHandlers, ...rykvDialogHandler }"
                ref="upload_file_dialog" />
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogHandler }" />
            <v-avatar v-if="!is_shared_rykv_view" :style="floatingActionButtonStyle()" color="primary"
                class="position-fixed">
                <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" v-long-press="() => show_kftl_dialog()" variant="text"
                            v-bind="props" />
                    </template>
                    <v-list>
                        <v-list-item @click="show_kftl_dialog()">
                            <v-list-item-title>{{ i18n.global.t("KFTL_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_mkfl_dialog()">
                            <v-list-item-title>{{ i18n.global.t("MKFL_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_add_kc_dialog()">
                            <v-list-item-title>{{ i18n.global.t("KC_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_urlog_dialog()">
                            <v-list-item-title>{{ i18n.global.t("URLOG_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_timeis_dialog()">
                            <v-list-item-title>{{ i18n.global.t("TIMEIS_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_mi_dialog()">
                            <v-list-item-title>{{ i18n.global.t("MI_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_nlog_dialog()">
                            <v-list-item-title>{{ i18n.global.t("NLOG_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_lantana_dialog()">
                            <v-list-item-title>{{ i18n.global.t("LANTANA_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_upload_file_dialog()">
                            <v-list-item-title>{{ i18n.global.t("UPLOAD_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
            <HelpDialog screen_name="rykv" ref="help_dialog" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { i18n } from '@/i18n'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import Dnote from './dnote-view.vue'
import GPSLogMap from './gps-log-map.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
import type { rykvViewEmits } from './rykv-view-emits'
import type { rykvViewProps } from './rykv-view-props'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import RyuuListView from './ryuu-list-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import RykvDialogHost from './rykv-dialog-host.vue'
import HelpDialog from '../dialogs/help-dialog.vue'
import { useRykvView } from '@/classes/use-rykv-view'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const props = defineProps<rykvViewProps>()
const emits = defineEmits<rykvViewEmits>()

const {
    // Template refs
    rykv_root,
    query_editor_sidebar,
    add_mi_dialog,
    add_nlog_dialog,
    add_lantana_dialog,
    add_timeis_dialog,
    add_urlog_dialog,
    kftl_dialog,
    add_kc_dialog,
    mkfl_dialog,
    upload_file_dialog,
    dnote_view,
    kyou_list_views,

    // State
    enable_context_menu,
    enable_dialog,
    opened_dialogs,
    querys,
    match_kyous_list,
    focused_query,
    focused_column_index,
    focused_kyous_list,
    focused_kyou,
    focused_column_checked_kyous,
    gps_log_map_start_time,
    gps_log_map_end_time,
    gps_log_map_marker_time,
    is_show_kyou_detail_view,
    is_show_kyou_count_calendar,
    is_show_gps_log_map,
    is_show_dnote,
    drawer,
    drawer_mode_is_mobile,
    default_query,
    is_loading,
    inited,

    // Computed
    kyou_list_view_height,
    add_kyou_menu_style,
    page_list,

    // Template event handlers
    toggleDrawer,
    navigateToPage,
    toggleDnote,
    onSidebarRequestedSearch,
    onSidebarUpdatedQuery,
    onSidebarInited,
    onColumnScrollList,
    onColumnClickedListView,
    onColumnClickedKyou,
    onColumnRequestedChangeFocusKyou,
    onColumnRequestedSearch,
    onColumnRequestedChangeImageOnlyView,
    onColumnRequestedReloadList,
    onRequestedFocusTime,
    onGpsLogMapRequestedFocusTime,
    onAddColumnClick,
    close_list_view,
    open_rykv_dialog,
    close_rykv_dialog,
    reload_kyou,

    // Dialog show methods
    show_kftl_dialog,
    show_mkfl_dialog,
    show_add_kc_dialog,
    show_urlog_dialog,
    show_timeis_dialog,
    show_mi_dialog,
    show_nlog_dialog,
    show_lantana_dialog,
    show_upload_file_dialog,
    floatingActionButtonStyle,

    // Event relay objects
    crudRelayHandlers,
    allColumnsRequestHandlers,
    subViewFocusHandlers,
    rykvDialogHandler,
} = useRykvView({ props, emits })

import { computed } from 'vue'
const is_ryuu_empty = computed(() => {
    const data = props.application_config.ryuu_json_data
    if (!data || !Array.isArray(data) || data.length === 0) return true
    return data.every((item: any) => {
        if (Array.isArray(item)) return item.length === 0
        if (item && typeof item === 'object' && 'queries' in item) {
            return !item.queries || !Array.isArray(item.queries) || item.queries.length === 0
        }
        return false
    })
})
</script>
<style lang="css" scoped>
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: calc(v-bind('app_content_height.toString().concat("px")') - v-bind('is_ryuu_empty ? "0px" : "100vh * 0.2"'));
    width: 400px;
    min-width: 0;
    max-width: 100vw;
}

@media (max-width: 600px) {
    .kyou_detail_view.dummy {
        width: 100vw;
    }
}

.ryuu_view.dummy {
    overflow-x: hidden;
    overflow-y: auto;
    height: calc(100vh * 0.2);
    height: calc(v-bind('is_ryuu_empty ? "0px" : "100vh * 0.2"'));
}

.scroll_snap_container {
    scroll-snap-type: x proximity;
    width: 100%;
}

.scroll_snap_area {
    scroll-snap-align: start;
    width: 100%;
}
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
