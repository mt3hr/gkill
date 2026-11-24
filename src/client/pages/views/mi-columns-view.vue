<template>
    <!-- mi（タスク画面）の「列」。**モードを知らない。**
         普段使う mi（mi-view）と HTML 出力（exported-mi-view）の2つのラッパがこれをホストし、
         何ができるかを props へリテラルで渡す。ラッパの持ち物（アプリバーの題名と画面切替メニュー・
         編集系のダイアログ群・FAB・ヘルプ・設定）はスロットで差し込まれる。
         data-gkill-view-ready は E2E の準備完了待ちが見る。真偽値をそのまま bind すると
         false で属性ごと消えるので、文字列にして渡す -->
    <div class="mi_view_wrap" ref="mi_root" :data-gkill-view-ready="is_view_ready ? 'true' : 'false'">
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
        <v-navigation-drawer v-model="drawer" app :height="app_content_height" :mobile="drawer_mode_is_mobile"
            :touchless="!inited" :width="318">
            <MiQueryEditorSidebar v-show="inited" class="mi_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */"
                @requested_search="onSidebarRequestedSearch"
                @requested_export_kyou_html="emits('requested_export_kyou_html')"
                @updated_query="onSidebarUpdatedQuery"
                @request_open_focus_board="(board_name: string) => open_or_focus_board(board_name)"
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
            <table class="mi_view_table" v-show="inited">
                <tbody>
                    <tr>
                        <td valign="top" v-for="query, index in querys" :key="query.query_id"
                            :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <v-card dropzone="true" @dragenter.prevent.stop="() => { }"
                                @drop.prevent.stop="(e: DragEvent) => onDropBoardTask(e, query)"
                                @dragover.prevent.stop="(e: DragEvent) => onDragoverBoardTask(e, query)">
                                <v-card-title class="mi_board_column_title" v-if="query.mi_board_name !== null">{{
                                    query.mi_board_name }}</v-card-title>
                                <v-card-title class="mi_board_column_title" v-if="query.mi_board_name === null">{{
                                    i18n.global.t("MI_ALL_TITLE") }}</v-card-title>
                                <KyouListView :kyou_height="56 + 35" :width="400"
                                    :draggable="enable_board_drag_drop"
                                    :list_height="kyou_list_view_height.valueOf() - MI_BOARD_TITLE_HEIGHT"
                                    :application_config="application_config" :gkill_api="gkill_api"
                                    :matched_kyous="match_kyous_list[index]" :query="query"
                                    :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                                    :enable_context_menu="enable_context_menu" :enable_dialog="true"
                                    :is_readonly_mi_check="is_readonly_mi_check" :show_checkbox="false"
                                    :show_footer="show_column_footer"
                                    :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false"
                                    :show_rep_name="false" :force_show_latest_kyou_info="true" :show_content_only="false"
                                    :show_timeis_playing_end_button="false"
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
                                    :show_timeis_elapsed_time="true"
                                    :show_timeis_playing_end_button="show_timeis_playing_end_button" :height="'unset'"
                                    :is_readonly_mi_check="is_readonly_mi_check" :width="'unset'"
                                    :enable_context_menu="enable_context_menu" :show_update_time="false"
                                    :show_related_time="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                    :enable_dialog="true" :show_attached_timeis="true" class="kyou_detail_view"
                                    :show_attached_tags="true" :show_attached_texts="true"
                                    :show_attached_notifications="true"
                                    v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...rykvDialogRelayHandlers }" />
                            </div>
                        </td>
                        <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                            <MiKyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                                :gkill_api="gkill_api" :kyous="focused_kyous_list" :mi_sort_type="focused_query.mi_sort_type"
                                :is_active="is_show_kyou_count_calendar"
                                @requested_focus_time="(date: Date) => onRequestedFocusTime(date)" />
                        </td>
                    </tr>
                </tbody>
            </table>
            <!-- 編集系のダイアログ群はラッパが差し込む。**中継束をスロット props で渡すこと** ――
                 保存した記録は列へ差し込む（再検索しない）ので、ダイアログの emit は
                 列の中継束へ流れていなければならない。
                 **sub_view_focus_relay も渡す** —— mi の打刻追加ダイアログだけが
                 ダイアログの中のフォーカス移動を上げてくる（rykv の同じダイアログは上げない）。
                 出力したファイルのラッパはこのスロットへ何も入れない -->
            <slot name="dialogs" :crud_relay="crudRelayHandlers"
                :all_columns_request_relay="allColumnsRequestHandlers"
                :sub_view_focus_relay="subViewFocusHandlers" :querys="querys"
                :focused_column_index="focused_column_index" />
            <!-- 詳細（ダブルクリックで開く KyouDialog）は**出力 HTML でも使える**。
                 出力は「読むために渡すファイル」なので、記録を開けないと用途を果たさない。
                 残りはコンテキストメニュー由来なので enable_context_menu=false で閉じたままになる -->
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="enable_context_menu" :enable_dialog="true"
                @closed="(id: string) => close_rykv_dialog(id)"
                v-on="{ ...crudRelayHandlers, ...allColumnsRequestHandlers, ...subViewFocusHandlers, ...rykvDialogRelayHandlers }" />
            <slot name="fab" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import { i18n } from '@/i18n'
import MiQueryEditorSidebar from './mi-query-editor-sidebar.vue'
import type { Kyou } from '@/classes/datas/kyou'
import MiKyouCountCalendar from './mi-kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import RykvDialogHost from './rykv-dialog-host.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { MiColumnsViewEmits } from './mi-columns-view-emits'
import type { MiColumnsViewProps } from './mi-columns-view-props'
import { useMiColumns } from '@/classes/use-mi-columns'
import { MI_BOARD_TITLE_HEIGHT } from '@/classes/mi-board-column-layout'

const props = defineProps<MiColumnsViewProps>()
const emits = defineEmits<MiColumnsViewEmits>()

const {
    // Template refs
    mi_root,
    query_editor_sidebar,
    kyou_list_views,

    // State
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
    overlay_target_min_width_px,
    inited,

    // Computed
    kyou_list_view_height,
    is_view_ready,

    // Template event handlers
    toggle_drawer,
    onSidebarRequestedSearch,
    onSidebarUpdatedQuery,
    onColumnScrollList,
    onColumnClickedListView,
    onColumnClickedKyou,
    onColumnRequestedChangeFocusKyou,
    onColumnRequestedSearch,
    onColumnRequestedChangeImageOnlyView,
    onRequestedFocusTime,
    onDropBoardTask,
    onDragoverBoardTask,
    close_list_view,
    open_or_focus_board,
    open_rykv_dialog,
    close_rykv_dialog,
    reload_kyou,
    update_check_kyous,

    // Event relay objects
    crudRelayHandlers,
    allColumnsRequestHandlers,
    rykvDialogRelayHandlers,
    subViewFocusHandlers,
    onColumnRequestedReloadList,
} = useMiColumns({ props, emits })
</script>
<style lang="css" scoped>
/* 列の高さ計算 (list_height = app_content_height - MI_BOARD_TITLE_HEIGHT) が
   Vuetify の既定サイズに依存しないよう、見出しの高さをここで固定する。
   mi-board-column-layout.ts の MI_BOARD_TITLE_HEIGHT と必ず同じ値にすること */
.mi_board_column_title {
    height: 44px;
}

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
    min-width: v-bind(overlay_target_min_width_px);
}
</style>
