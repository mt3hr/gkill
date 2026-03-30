<template>
    <div class="mi_view_wrap" ref="mi_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_SIDEBAR')">
                <template v-slot:activator="{ props }">
                    <v-app-bar-nav-icon v-bind="props" @click.stop="toggleDrawer" :disabled="!inited" />
                </template>
            </v-tooltip>
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("MI_APP_TITLE") }}
                    </span>
                    <v-menu activator="parent">
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
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_CALENDAR')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_kyou_count_calendar = !is_show_kyou_count_calendar">
                        <v-icon>mdi-calendar</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>

            <v-divider vertical />
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-cog" :disabled="!application_config.is_loaded"
                        @click="emits('requested_show_application_config_dialog')" />
                </template>
            </v-tooltip>
        </v-app-bar>
        <v-navigation-drawer v-model="drawer" app :height="app_content_height" :mobile="drawer_mode_is_mobile"
            :touchless="!inited" :width="318">
            <MiQueryEditorSidebar v-show="inited" class="mi_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */"
                @requested_search="onSidebarRequestedSearch"
                @updated_query="onSidebarUpdatedQuery"
                @inited="onSidebarInited"
                @request_open_focus_board="(board_name: string) => open_or_focus_board(board_name)"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main" :class="(drawer_mode_is_mobile) ? 'scroll_snap_container' : ''">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="mi_view_table" v-show="inited">
                <tr>
                    <td valign="top" v-for="query, index in querys" :key="query.query_id"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <v-card dropzone="true" @dragenter.prevent.stop="() => { }"
                            @drop.prevent.stop="(e: DragEvent) => on_drop_board_task(e, query)"
                            @dragover.prevent.stop="(e: DragEvent) => on_dragover_board_task(e, query)">
                            <v-card-title v-if="query.use_mi_board_name">{{ query.mi_board_name }}</v-card-title>
                            <v-card-title v-if="!query.use_mi_board_name">{{ i18n.global.t("MI_ALL_TITLE")
                            }}</v-card-title>
                            <KyouListView :kyou_height="56 + 35" :width="400" :draggable="true"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous_list[index]" :query="query"
                                :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                                :is_readonly_mi_check="false" :show_checkbox="false" :show_footer="true"
                                :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false"
                                :show_rep_name="false" :force_show_latest_kyou_info="true" :show_content_only="false"
                                :show_timeis_plaing_end_button="false"
                                v-on="crudRelayHandlers"
                                @scroll_list="(position: number) => onColumnScrollList(index, position)"
                                @clicked_list_view="() => onColumnClickedListView(index)"
                                @clicked_kyou="(kyou: Kyou) => onColumnClickedKyou(index, kyou)"
                                @focused_kyou="(kyou: Kyou) => onColumnClickedKyou(index, kyou)"
                                @requested_change_focus_kyou="(is_focus: boolean) => onColumnRequestedChangeFocusKyou(index, is_focus)"
                                @requested_search="() => onColumnRequestedSearch(index)"
                                @requested_change_is_image_only_view="(is_image_only: boolean) => onColumnRequestedChangeImageOnlyView(index, is_image_only)"
                                @requested_close_column="close_list_view(index)"
                                @requested_reload_kyou="(kyou: Kyou) => reload_kyou(kyou)"
                                @requested_reload_list="() => reload_list(index)"
                                @requested_update_check_kyous="(kyous: Kyou[], checked: boolean) => update_check_kyous(kyous, checked)"
                                @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload)"
                                ref="kyou_list_views" />
                        </v-card>
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
                                :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true" :height="'unset'"
                                :is_readonly_mi_check="false" :width="'unset'"
                                :enable_context_menu="enable_context_menu" :show_update_time="false"
                                :show_related_time="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                :enable_dialog="enable_dialog" :show_attached_timeis="true" class="kyou_detail_view"
                                :show_attached_tags="false" :show_attached_texts="false"
                                :show_attached_notifications="false"
                                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...rykvDialogHandler }" />
                        </div>
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="true"
                            @requested_focus_time="(date: Date) => onRequestedFocusTime(date)" />
                    </td>
                </tr>
            </table>
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="crudRelayHandlers"
                ref="add_kc_dialog" />
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                @focused_kyou="(kyou: Kyou) => { focused_kyou = kyou as Kyou }"
                @clicked_kyou="(kyou: Kyou) => { focused_kyou = kyou as Kyou }"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers }"
                ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                v-on="crudRelayHandlers"
                ref="upload_file_dialog" />
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @closed="(id: string) => close_rykv_dialog(id)"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...rykvDialogHandler }" />
            <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
                <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" v-long-press="() => show_kftl_dialog()" icon="mdi-plus" variant="text"
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
            <HelpDialog screen_name="mi" ref="help_dialog" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import { i18n } from '@/i18n'
import MiQueryEditorSidebar from './mi-query-editor-sidebar.vue'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
import type { miViewEmits } from './mi-view-emits'
import type { miViewProps } from './mi-view-props'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import RykvDialogHost from './rykv-dialog-host.vue'
import HelpDialog from '../dialogs/help-dialog.vue'
import { useMiView } from '@/classes/use-mi-view'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const props = defineProps<miViewProps>()
const emits = defineEmits<miViewEmits>()

const {
    // Template refs
    mi_root,
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
    is_show_kyou_detail_view,
    is_show_kyou_count_calendar,
    drawer,
    drawer_mode_is_mobile,
    is_loading,
    inited,

    // Computed
    kyou_list_view_height,
    add_kyou_menu_style,
    page_list,

    // Template event handlers
    toggleDrawer,
    navigateToPage,
    onSidebarRequestedSearch,
    onSidebarUpdatedQuery,
    onSidebarInited,
    onColumnScrollList,
    onColumnClickedListView,
    onColumnClickedKyou,
    onColumnRequestedChangeFocusKyou,
    onColumnRequestedSearch,
    onColumnRequestedChangeImageOnlyView,
    onRequestedFocusTime,
    on_drop_board_task,
    on_dragover_board_task,
    close_list_view,
    open_or_focus_board,
    open_rykv_dialog,
    close_rykv_dialog,
    reload_kyou,
    reload_list,
    update_check_kyous,

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
    rykvDialogHandler,
} = useMiView({ props, emits })
</script>
<style lang="css" scoped>
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: calc(v-bind('app_content_height.toString().concat("px")'));
    width: 408px;
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
