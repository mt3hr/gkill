<template>
    <!-- rykv の「列」。**モードを知らない。**
         通常（rykv-view）と HTML 出力（exported-rykv-view）の2つのラッパがこれをホストし、
         何ができるかを props へリテラルで渡す。ラッパの持ち物（アプリバーの中身・
         編集系のダイアログ群・FAB・ヘルプ）はスロットで差し込まれる。
         data-gkill-view-ready は E2E の準備完了待ちが見る。真偽値をそのまま bind すると
         false で属性ごと消えるので、文字列にして渡す -->
    <div class="rykv_view_wrap" ref="rykv_root" :data-gkill-view-ready="is_view_ready ? 'true' : 'false'">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_SIDEBAR')">
                <template v-slot:activator="{ props }">
                    <v-app-bar-nav-icon v-bind="props" v-if="is_sidebar_available" @click.stop="toggle_drawer"
                        :disabled="!inited" />
                </template>
            </v-tooltip>
            <v-toolbar-title>
                <slot name="app_bar_title" />
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
                    <v-btn v-bind="props" v-if="is_dnote_available" icon
                        @click="toggle_dnote">
                        <v-icon>mdi-file-chart-outline</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_MAP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" v-if="is_map_available" icon @click="is_show_gps_log_map = !is_show_gps_log_map">
                        <v-icon>mdi-map</v-icon>
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
            <slot name="app_bar_actions" />
        </v-app-bar>
        <!-- サイドバーを含めない出力でも **mount したままにする**。
             init() はサイドバーの template ref が埋まっていないと列を作らずに戻るので、
             v-if で消すと列が1本も作られず読み込み中オーバーレイが消えない（ADR-0415）。
             落とすのは開く導線（アプリバーのアイコン）だけ -->
        <v-navigation-drawer v-model="drawer" app :height="app_content_height"
            :touchless="!inited" :mobile="drawer_mode_is_mobile" :width="318">
            <RykvQueryEditorSideBar v-show="inited" class="rykv_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */"
                @requested_search="onSidebarRequestedSearch"
                @requested_export_kyou_html="emits('requested_export_kyou_html')"
                @updated_query="onSidebarUpdatedQuery"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main" :class="(drawer_mode_is_mobile) ? 'scroll_snap_container' : ''">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <!-- 設定が取れないと is_loaded が立たず初期化が一度も走らない。
                         スピナーのままにすると永久に固まるので、再試行の導線を出す -->
                    <div v-if="application_config_load_failed" class="text-center">
                        <div class="mb-2">{{ i18n.global.t('FAILED_GET_APPLICATION_CONFIG_MESSAGE') }}</div>
                        <v-btn color="primary" @click="emits('requested_reload_application_config')">
                            {{ i18n.global.t('RELOAD_TITLE') }}
                        </v-btn>
                    </div>
                    <v-progress-circular v-else indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="rykv_view_table" v-show="inited">
                <tbody>
                    <tr>
                        <td valign="top" v-for="query, index in querys" :key="query.query_id"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous_list[index]" :query="query"
                                :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                                :enable_context_menu="enable_context_menu" :enable_dialog="true"
                                :is_readonly_mi_check="is_readonly_mi_check" :show_checkbox="true"
                                :show_footer="show_column_footer"
                                :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true"
                                :show_rep_name="show_rep_name" :force_show_latest_kyou_info="true"
                                :show_content_only="false" :show_timeis_playing_end_button="false"
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
                                @requested_reload_list="() => onColumnRequestedReloadList(index)"
                                @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload)"
                                ref="kyou_list_views" />
                        </td>
                        <td valign="top" v-if="enable_add_column"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <v-btn class="rykv_add_column_button rounded-sm mx-auto" :height="app_content_height.valueOf()"
                                :width="30" :color="'primary'" @click="onAddColumnClick" icon="mdi-plus" variant="text"
                                :style="{ background: 'rgb(var(--v-theme-background))' }" />
                        </td>
                        <td valign="top" v-if="is_show_kyou_detail_view"
                            class="rykv_kyou_detail_view_wrap"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <div class="kyou_detail_view dummy" ref="kyou_detail_view_element">
                                <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                    :is_image_request_to_thumb_size="false" :application_config="application_config"
                                    :gkill_api="gkill_api" :highlight_targets="[]" :is_image_view="false"
                                    :kyou="focused_kyou" :show_checkbox="false"
                                    :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                    :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                    :show_timeis_elapsed_time="true"
                                    :show_timeis_playing_end_button="show_timeis_playing_end_button"
                                    :height="'auto'" :is_readonly_mi_check="is_readonly_mi_check" :width="'auto'"
                                    :enable_context_menu="enable_context_menu" :enable_dialog="true"
                                    :show_attached_timeis="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                    class="kyou_detail_view" :show_update_time="false" :show_related_time="true"
                                    :show_attached_tags="true" :show_attached_texts="true"
                                    :show_attached_notifications="true"
                                    v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...rykvDialogRelayHandlers }" />
                            </div>
                            <div class="ryuu_view dummy">
                                <RyuuView v-if="focused_kyou && default_query" :application_config="application_config"
                                    :gkill_api="gkill_api" :target_kyou="focused_kyou" :editable="false"
                                    :find_kyou_query_default="default_query"
                                    :matched_kyous="match_kyous_list[focused_column_index] ?? null"
                                    v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogRelayHandlers }" />
                            </div>
                        </td>
                        <td valign="top" v-if="is_show_dnote && is_dnote_available"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <Dnote class="rykv_dnote_wrap" :app_content_height="app_content_height"
                                :app_content_width="app_content_width" :application_config="application_config"
                                :gkill_api="gkill_api" :query="focused_query" :checked_kyous="focused_column_checked_kyous"
                                :editable="false" :fill_height="true"
                                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogRelayHandlers }"
                                ref="dnote_view" />
                        </td>
                        <td valign="top" v-if="is_map_available"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <GPSLogMap v-show="is_show_gps_log_map" :application_config="application_config"
                                :gkill_api="gkill_api" :start_time="gps_log_map_start_time" :end_time="gps_log_map_end_time"
                                :marker_time="gps_log_map_marker_time" :app_content_height="app_content_height"
                                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                                @requested_focus_time="(date: Date) => onGpsLogMapRequestedFocusTime(date)"
                                @requested_change_map_date="(date: Date) => onGpsLogMapRequestedChangeDate(date)" />
                        </td>
                        <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                                :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="false"
                                :is_active="is_show_kyou_count_calendar"
                                @requested_focus_time="(date: Date) => onRequestedFocusTime(date)" />
                        </td>
                    </tr>
                </tbody>
            </table>
            <!-- 編集系のダイアログ群はラッパが差し込む。**中継束をスロット props で渡すこと** ――
                 保存した記録は列へ差し込む（再検索しない）ので、ダイアログの emit は
                 列の中継束へ流れていなければならない。
                 出力したファイルのラッパはこのスロットへ何も入れない -->
            <slot name="dialogs" :crud_relay="crudRelayHandlers"
                :all_columns_request_relay="allColumnsRequestHandlers" :querys="querys"
                :focused_column_index="focused_column_index" />
            <!-- 詳細（ダブルクリックで開く KyouDialog）は**出力 HTML でも使える**。
                 出力は「読むために渡すファイル」なので、記録を開けないと用途を果たさない。
                 ホストする27種のうち詳細以外の26種は全部コンテキストメニュー由来で、
                 そちらは enable_context_menu=false で閉じたままになる。
                 enable_dialog はダイアログの中の KyouView へも渡る。false にすると
                 開いた詳細の中からさらに詳細を開けない（関連する記録を辿れない） -->
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="enable_context_menu"
                :enable_dialog="true" @closed="(id: string) => close_rykv_dialog(id)"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogRelayHandlers }" />
            <slot name="fab" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import { i18n } from '@/i18n'
import type { Kyou } from '@/classes/datas/kyou'
import Dnote from './dnote-view.vue'
import GPSLogMap from './gps-log-map.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue'
import RyuuView from './ryuu-view.vue'
import RykvDialogHost from './rykv-dialog-host.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RykvColumnsViewEmits } from './rykv-columns-view-emits'
import type { RykvColumnsViewProps } from './rykv-columns-view-props'
import { useRykvColumns } from '@/classes/use-rykv-columns'

const props = defineProps<RykvColumnsViewProps>()
const emits = defineEmits<RykvColumnsViewEmits>()

const {
    // Template refs
    rykv_root,
    query_editor_sidebar,
    dnote_view,
    kyou_list_views,
    kyou_detail_view_element,

    // State
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
    ryuu_pane_height_px,
    ryuu_pane_width_px,
    kyou_detail_view_height_px,
    kyou_detail_view_base_width_px,
    app_content_width_px,
    overlay_target_min_width_px,

    // Computed
    kyou_list_view_height,
    is_view_ready,

    // Template event handlers
    toggle_drawer,
    toggle_dnote,
    onSidebarRequestedSearch,
    onSidebarUpdatedQuery,
    onColumnScrollList,
    onColumnClickedListView,
    onColumnClickedKyou,
    onColumnRequestedChangeFocusKyou,
    onColumnRequestedSearch,
    onColumnRequestedChangeImageOnlyView,
    onColumnRequestedReloadList,
    onRequestedFocusTime,
    onGpsLogMapRequestedFocusTime,
    onGpsLogMapRequestedChangeDate,
    onAddColumnClick,
    close_list_view,
    open_rykv_dialog,
    close_rykv_dialog,
    reload_kyou,

    // Event relay objects
    crudRelayHandlers,
    allColumnsRequestHandlers,
    subViewFocusHandlers,
    rykvDialogRelayHandlers,
} = useRykvColumns({ props, emits })

</script>
<style lang="css" scoped>
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: v-bind(kyou_detail_view_height_px);
    width: v-bind(kyou_detail_view_base_width_px);
    min-width: 400px;
    max-width: v-bind(app_content_width_px);
}

/* 確定幅を与える。中身のKyouの型によらず、外側のtd列がryuuの中身のmin-contentで広がるのを防ぐ */
.ryuu_view.dummy {
    overflow-x: hidden;
    overflow-y: auto;
    height: v-bind(ryuu_pane_height_px);
    width: v-bind(ryuu_pane_width_px);
    max-width: v-bind(ryuu_pane_width_px);
}

.ryuu_view.dummy :deep(.ryuu_views) {
    min-width: 0;
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
    min-width: v-bind(overlay_target_min_width_px);
}
.rykv_kyou_detail_view_wrap {
    background-color: rgb(var(--v-theme-background));
}
</style>
