<template>
    <div class="mi_view_wrap">
        <v-app-bar :height="app_title_bar_height" class="app_bar" color="primary" app flat>
            <v-toolbar-title> {{ share_title }} </v-toolbar-title>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="mi_view_table" v-show="!is_loading">
                <tr>
                    <td valign="top">
                        <v-card>
                            <v-card-title>{{ share_title }}</v-card-title>
                            <KyouListView :kyou_height="56 + 35" :width="400" :show_timeis_plaing_end_button="false"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous" :query="new FindKyouQuery()"
                                :is_focused_list="true" :closable="false" :is_readonly_mi_check="true"
                                :show_checkbox="false" :show_footer="false" :enable_context_menu="false"
                                :enable_dialog="false" :show_content_only="false"
                                :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false"
                                :show_rep_name="false" :force_show_latest_kyou_info="true"
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @focused_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                                @clicked_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                                v-on="crudRelayHandlers"
                                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])"
                                ref="kyou_list_view" />
                        </v-card>
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view">
                        <table>
                            <tr>
                                <td valign="top">
                                    <KyouCountCalendar v-show="is_show_kyou_count_calendar"
                                        :application_config="application_config" :gkill_api="gkill_api"
                                        :kyous="match_kyous" :for_mi="true" class="kyou_list_calendar_in_share_mi_view"
                                        @requested_focus_time="(...time: any[]) => { focused_time = time[0] as Date }" />
                                </td>
                            </tr>
                            <tr>
                                <td valign="top" v-if="is_show_kyou_detail_view">
                                    <div class="kyou_detail_view dummy">
                                        <KyouView v-if="focused_kyou && is_show_kyou_detail_view" :is_image_request_to_thumb_size="false"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="focused_kyou"
                                            :show_checkbox="false" :show_content_only="false"
                                            :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                            :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                            :show_attached_timeis="true" :show_timeis_elapsed_time="false"
                                            :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()"
                                            :is_readonly_mi_check="true" :width="400" :enable_context_menu="false"
                                            :show_rep_name="false" :force_show_latest_kyou_info="true"
                                            :enable_dialog="false" :show_update_time="false" :show_related_time="true"
                                            class="kyou_detail_view" :show_attached_tags="true"
                                            :show_attached_texts="true" :show_attached_notifications="true"
                                            v-on="{ ...crudRelayHandlers, ...rykvDialogHandler }" />
                                    </div>
                                </td>
                            </tr>
                        </table>
                    </td>
                </tr>
            </table>
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="false" :enable_dialog="false"
                @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
                @focused_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                @clicked_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                v-on="{ ...crudRelayHandlers, ...rykvDialogHandler }"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { SharedMiViewProps } from './shared-mi-view-props'

import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import KyouCountCalendar from './kyou-count-calendar.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from './kyou-view-emits'
import RykvDialogHost from './rykv-dialog-host.vue'
import { useSharedMiView } from '@/classes/use-shared-mi-view'

const props = defineProps<SharedMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    kyou_list_view,

    // State
    match_kyous,
    focused_time,
    share_title,
    is_loading,
    is_show_kyou_detail_view,
    is_show_kyou_count_calendar,
    focused_kyou,
    opened_dialogs,

    // Computed
    kyou_list_view_height,

    // Business logic
    reload_kyou,
    onDeletedKyou,
    open_rykv_dialog,
    close_rykv_dialog,

    // Event relay objects
    crudRelayHandlers,
    rykvDialogHandler,
} = useSharedMiView({ props, emits })
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
<style lang="css">
.mi_view_wrap .v-calendar-weekly__head {
    width: unset !important;
}
</style>
