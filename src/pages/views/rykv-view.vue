<template>
    <KyouListView v-for="query, index in querys" :application_config="application_config" :gkill_api="gkill_api"
        :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="reload_list(index)"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <RykvQueryEditorSideBar :application_config="application_config" :gkill_api="gkill_api"
        :query="querys[focused_column_index]" @request_search="search(focused_column_index)"
        @updated_query="(new_query) => querys.splice(focused_column_index, 1, new_query)" />
    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :is_image_view="false" :kyou="focused_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
        :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <KyouCountCalendar :application_config="application_config" :gkill_api="gkill_api" :kyous="focused_column_kyous"
        @requested_focus_time="(time) => focused_time = time" />
    <GPSLogMap :application_config="application_config" :gkill_api="gkill_api" :start_date="focused_time"
        :end_date="focused_time" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_focus_time="(time) => focused_time = time" />
    <Dnote :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config="application_config" :gkill_api="gkill_api" :query="querys[focused_column_index]"
        :last_added_tag="last_added_tag" @received_messages="(messages) => emits('received_messages', messages)"
        @received_errors="(errors) => emits('received_errors', errors)" />
    <AggregateView :application_config="application_config" :gkill_api="gkill_api"
        :checked_kyous="focused_column_checked_kyous"
        @received_messages="(messages) => emits('received_messages', messages)"
        @received_errors="(errors) => emits('received_errors', errors)" />
    <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :last_added_tag="last_added_tag" :kyou="new Kyou()"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :last_added_tag="last_added_tag" :kyou="new Kyou()"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :last_added_tag="last_added_tag" :kyou="new Kyou()"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <StartTimeIsDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :last_added_tag="last_added_tag" :kyou="new Kyou()"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
        :app_content_width="app_content_width" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
</template>
<script setup lang="ts">

import { type Ref, ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import AggregateView from './aggregate-view.vue'
import Dnote from './dnote.vue'
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue'
import GPSLogMap from './gps-log-map.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import StartTimeIsDialog from '../dialogs/start-time-is-dialog.vue'
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import type { rykvViewEmits } from './rykv-view-emits'
import type { rykvViewProps } from './rykv-view-props'

const querys: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>())
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const focused_column_index: Ref<number> = ref(0)
const focused_kyou: Ref<Kyou> = ref(new Kyou())
const focused_list_views_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_time: Ref<Date> = ref(new Date())
const focused_column_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_column_checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const is_show_kyou_detail_view: Ref<boolean> = ref(false)
const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
const is_show_gps_log_map: Ref<boolean> = ref(false)
const last_added_tag: Ref<string> = ref("")
const props = defineProps<rykvViewProps>()
const emits = defineEmits<rykvViewEmits>()

async function add_list_view(): Promise<void> {
    throw new Error('Not implemented')
}
async function close_list_view(query_index: Number): Promise<void> {
    throw new Error('Not implemented')
}
async function update_queries(query_index: Number, by_user: boolean): Promise<void> {
    throw new Error('Not implemented')
}
async function update_kyous(column_index: Number, kyous: Array<Kyou>): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_list(column_index: number): Promise<void> {
    throw new Error('Not implemented')
}

async function update_check_kyous(kyous: Array<Kyou>, is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

async function search(column_index: number): Promise<void> {
    throw new Error('Not implemented')
}
</script>
