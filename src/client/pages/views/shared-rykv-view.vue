<template>
    <!-- 共有された rykv 画面。**通常の rykv（rykv-view）とは別のビュー。**
         共有は「渡された検索結果を1列で見せるだけ」なので、列の機構も検索もサイドバーも持たない。
         読み取り専用であることは、子へ渡す値をリテラルで書くことで表す
         （モードのフラグをこのビューに持ち込まない。shared-mi-view と同じ作法） -->
    <div class="rykv_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-toolbar-title>
                <div>{{ share_title }}</div>
            </v-toolbar-title>
            <v-spacer />
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_DETAIL_VIEW')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_kyou_detail_view = !is_show_kyou_detail_view">
                        <v-icon>mdi-file-document</v-icon>
                    </v-btn>
                </template>
            </v-tooltip>
            <!-- 地図は共有でも出す（GPSLog を共有できる）。集計（Dnote）は出さない -->
            <v-tooltip :text="i18n.global.t('TOOLTIP_TOGGLE_MAP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon @click="is_show_gps_log_map = !is_show_gps_log_map">
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
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="rykv_view_table" v-show="!is_loading">
                <tbody>
                    <tr>
                        <td valign="top">
                            <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous" :query="default_query"
                                :is_focused_list="true" :closable="false"
                                :enable_context_menu="false" :enable_dialog="false"
                                :is_readonly_mi_check="true" :show_checkbox="true" :show_footer="false"
                                :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true"
                                :show_rep_name="false" :force_show_latest_kyou_info="true"
                                :show_content_only="false" :show_timeis_playing_end_button="false"
                                v-on="crudRelayHandlers"
                                @clicked_kyou="(kyou: Kyou) => onClickedKyou(kyou)"
                                @focused_kyou="(kyou: Kyou) => onClickedKyou(kyou)"
                                ref="kyou_list_view" />
                        </td>
                        <td valign="top" v-if="is_show_kyou_detail_view" class="rykv_kyou_detail_view_wrap">
                            <div class="kyou_detail_view dummy">
                                <KyouView v-if="focused_kyou" :is_image_request_to_thumb_size="false"
                                    :application_config="application_config" :gkill_api="gkill_api"
                                    :highlight_targets="[]" :is_image_view="false" :kyou="focused_kyou"
                                    :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                                    :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                    :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
                                    :show_timeis_playing_end_button="false" :height="'auto'"
                                    :is_readonly_mi_check="true" :width="'auto'"
                                    :enable_context_menu="false" :enable_dialog="false"
                                    :show_attached_timeis="true" :show_rep_name="true"
                                    :force_show_latest_kyou_info="true" class="kyou_detail_view"
                                    :show_update_time="false" :show_related_time="true" :show_attached_tags="true"
                                    :show_attached_texts="true" :show_attached_notifications="true"
                                    v-on="{ ...crudRelayHandlers, ...subViewFocusHandlers }" />
                            </div>
                            <div class="ryuu_view dummy">
                                <RyuuView v-if="focused_kyou" :application_config="application_config"
                                    :gkill_api="gkill_api" :target_kyou="focused_kyou" :editable="false"
                                    :find_kyou_query_default="default_query" :matched_kyous="match_kyous"
                                    v-on="{ ...crudRelayHandlers, ...subViewFocusHandlers }" />
                            </div>
                        </td>
                        <td valign="top">
                            <GPSLogMap v-show="is_show_gps_log_map" :application_config="application_config"
                                :gkill_api="gkill_api" :start_time="gps_log_map_start_time"
                                :end_time="gps_log_map_end_time" :marker_time="gps_log_map_marker_time"
                                :app_content_height="app_content_height"
                                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                                @requested_focus_time="(date: Date) => onGpsLogMapRequestedFocusTime(date)"
                                @requested_change_map_date="(date: Date) => focus_gps_log_map_on(date)" />
                        </td>
                        <td valign="top">
                            <KyouCountCalendar v-show="is_show_kyou_count_calendar"
                                :application_config="application_config" :gkill_api="gkill_api" :kyous="match_kyous"
                                :for_mi="false" :is_active="is_show_kyou_count_calendar"
                                @requested_focus_time="(date: Date) => onRequestedFocusTime(date)" />
                        </td>
                    </tr>
                </tbody>
            </table>
            <!-- 詳細ダイアログのホスト。共有では開く導線が無い（enable_dialog も
                 enable_context_menu も false）が、shared-mi-view と同じく常時 mount しておく -->
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="false" :enable_dialog="false"
                v-on="rykvDialogHandlers" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { SharedRykvViewProps } from './shared-rykv-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import { i18n } from '@/i18n'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import GPSLogMap from './gps-log-map.vue'
import RyuuView from './ryuu-view.vue'
import RykvDialogHost from './rykv-dialog-host.vue'
import { useSharedRykvView } from '@/classes/use-shared-rykv-view'

const props = defineProps<SharedRykvViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    kyou_list_view,

    // State
    match_kyous,
    focused_kyou,
    is_loading,
    is_show_kyou_detail_view,
    is_show_kyou_count_calendar,
    is_show_gps_log_map,
    gps_log_map_start_time,
    gps_log_map_end_time,
    gps_log_map_marker_time,
    opened_dialogs,
    default_query,

    // Computed
    kyou_list_view_height,

    // Business logic
    onClickedKyou,
    onRequestedFocusTime,
    onGpsLogMapRequestedFocusTime,
    focus_gps_log_map_on,

    // Event relay objects
    crudRelayHandlers,
    subViewFocusHandlers,
    rykvDialogHandlers,
} = useSharedRykvView({ props, emits })
</script>
<style lang="css" scoped>
/* ビューポート単位（100vh / 100vw）とメディアクエリは使わない。
   このビューは単独ページ専用だが、rykv-view と同じ作法に揃えておく */
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: v-bind('app_content_height.toString().concat("px")');
    width: 400px;
    min-width: 400px;
    max-width: v-bind('app_content_width.toString().concat("px")');
}

.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? app_content_width.toString().concat('px') : '0px'");
}

.rykv_kyou_detail_view_wrap {
    background-color: rgb(var(--v-theme-background));
}
</style>
