<template>
    <v-card class="dnote_view">
        <h1><span>{{ start_date_str }}</span><span v-if="end_date_str !== '' && start_date_str != end_date_str">～</span><span
                v-if="end_date_str !== '' && start_date_str != end_date_str">{{ end_date_str }}</span></h1>
        <table>
            <tr>
                <td>
                    <div>覚醒：{{ calclutated_total_awake_time }}</div>
                    <div>睡眠：{{ calclutated_total_sleep_time }}</div>
                    <div>仕事：{{ calclutated_total_work_time }}</div>
                </td>
                <td>
                    <div>煙草：{{ calclutated_tabaco_record_count }} 本</div>
                    <div style="display: flex;">気分：
                        <LantanaFlowersView :gkill_api="gkill_api" :application_config="application_config"
                            :mood="calclated_average_lantana_mood" :editable="false" />
                    </div>
                    <div>収入： {{ calclutated_total_nlog_plus_amount }} 円</div>
                    <div>支出： {{ calclutated_total_nlog_minus_amount }} 円</div>
                    <div>コード：
                        <span class="git_commit_addition"> + {{ calclutated_total_git_addition_count }} 行</span>
                        <span class="git_commit_deletion"> - {{ calclutated_total_git_deletion_count }} 行</span>
                    </div>
                </td>
                <td>
                    <div>合計時間：{{ total_checked_time }}</div>
                    <div>合計収入：{{ total_checked_nlog_plus_amount }} 円</div>
                    <div>合計支出：{{ total_checked_nlog_minus_amount }} 円</div>
                </td>
            </tr>
            <tr>
                <td>
                    <h2>支出</h2>
                    <KyouListView :kyou_height="180" :width="200" :list_height="400"
                        :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="nlog_kyous"
                        :query="(() => { const query = new FindKyouQuery(); query.is_image_only = false; query.query_id = GkillAPI.get_instance().generate_uuid(); return query })()"
                        :last_added_tag="last_added_tag" :is_focused_list="false" :closable="false"
                        :show_checkbox="false" :show_footer="false" @scroll_list="() => { }"
                        @clicked_list_view="() => { }" @clicked_kyou="() => { }"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        @requested_reload_kyou="() => { }" @requested_reload_list="() => { }"
                        @requested_update_check_kyous="() => { }" @requested_change_focus_kyou="() => { }"
                        @requested_search="() => { }" ref="nlog_kyou_list_views"
                        @requested_change_is_image_only_view="() => { }" @requested_close_column="() => { }" />
                </td>
                <td>
                    <h2>場所（ {{ location_timeis_kmemo_kyous.length }} 件 ）</h2>
                    <KyouListView :kyou_height="180" :width="200" :list_height="400"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :matched_kyous="location_timeis_kmemo_kyous"
                        :query="(() => { const query = new FindKyouQuery(); query.is_image_only = false; query.query_id = GkillAPI.get_instance().generate_uuid(); return query })()"
                        :last_added_tag="last_added_tag" :is_focused_list="false" :closable="false"
                        @scroll_list="() => { }" :show_checkbox="false" :show_footer="false"
                        @clicked_list_view="() => { }" @clicked_kyou="() => { }"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        @requested_reload_kyou="() => { }" @requested_reload_list="() => { }"
                        @requested_update_check_kyous="() => { }" @requested_change_focus_kyou="() => { }"
                        @requested_search="() => { }" ref="location_kyou_list_views"
                        @requested_change_is_image_only_view="() => { }" @requested_close_column="() => { }" />
                </td>
                <td>
                    <h2>人（ {{ people_timeis_kmemo_kyous.length }} 件 ）</h2>
                    <KyouListView :kyou_height="180" :width="200" :list_height="400"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :matched_kyous="people_timeis_kmemo_kyous"
                        :query="(() => { const query = new FindKyouQuery(); query.is_image_only = false; query.query_id = GkillAPI.get_instance().generate_uuid(); return query })()"
                        :last_added_tag="last_added_tag" :is_focused_list="false" :closable="false"
                        @scroll_list="() => { }" :show_checkbox="false" :show_footer="false"
                        @clicked_list_view="() => { }" @clicked_kyou="() => { }"
                        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
                        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
                        @requested_reload_kyou="() => { }" @requested_reload_list="() => { }"
                        @requested_update_check_kyous="() => { }" @requested_change_focus_kyou="() => { }"
                        @requested_search="() => { }" ref="people_kyou_list_views"
                        @requested_change_is_image_only_view="() => { }" @requested_close_column="() => { }" />
                </td>
            </tr>
        </table>
    </v-card>
</template>
<script setup lang="ts">
import type { DnoteEmits } from './dnote-emits'
import type { DnoteProps } from './dnote-props'

import type { Kmemo } from '@/classes/datas/kmemo'
import type { Lantana } from '@/classes/datas/lantana'
import { TimeIs } from '@/classes/datas/time-is'
import { computed, nextTick, ref, watch, type Ref } from 'vue'

import DnoteLocationListView from './dnote-location-list-view.vue'
import DnoteNlogsListView from './dnote-nlogs-list-view.vue'
import DnotePeoplesListView from './dnote-peoples-list-view.vue'
import moment from 'moment'
import type { Kyou } from '@/classes/datas/kyou'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GkillAPI } from '@/classes/api/gkill-api'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { GkillError } from '@/classes/api/gkill-error'
import LantanaFlowersView from './lantana-flowers-view.vue'
import KyouListView from './kyou-list-view.vue'
import { deepEquals } from '@/classes/deep-equals'
import { ApplicationConfig } from '@/classes/datas/config/application-config'

const props = defineProps<DnoteProps>()
const emits = defineEmits<DnoteEmits>()
defineExpose({ recalc })

const is_loading = ref(false)
const start_date_str: Ref<string> = computed(() => !cloned_query.value.calendar_start_date ? "" : (moment(cloned_query.value.calendar_start_date ? cloned_query.value.calendar_start_date : moment().toDate()).format("YYYY-MM-DD")))
const end_date_str: Ref<string> = computed(() => !cloned_query.value.calendar_end_date ? "" : (moment(cloned_query.value.calendar_end_date ? cloned_query.value.calendar_end_date : moment().toDate()).format("YYYY-MM-DD")))
const date_kmemo0000_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const awake_timeis_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const sleep_timeis_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const work_timeis_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const location_timeis_kmemo_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const nlog_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const people_timeis_kmemo_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const tabaco_kmemo_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const lantana_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const git_commit_log_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const calclutated_total_awake_time: Ref<string> = ref("")
const calclutated_total_sleep_time: Ref<string> = ref("")
const calclutated_total_work_time: Ref<string> = ref("")
const calclutated_tabaco_record_count: Ref<Number> = ref(0)
const calclated_average_lantana_mood: Ref<Number> = ref(0)
const calclutated_total_git_addition_count: Ref<Number> = ref(0)
const calclutated_total_git_deletion_count: Ref<Number> = ref(0)
const calclutated_total_nlog_plus_amount: Ref<Number> = ref(0)
const calclutated_total_nlog_minus_amount: Ref<Number> = ref(0)

const total_checked_time: Ref<string> = ref("")
const total_checked_nlog_plus_amount: Ref<Number> = ref(0)
const total_checked_nlog_minus_amount: Ref<Number> = ref(0)

const abort_controller: Ref<AbortController> = ref(new AbortController())
const cloned_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const cloned_application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())

watch(() => props.application_config, () => load_application_config())

async function recalc(): Promise<void> {
    is_loading.value = true
    abort_controller.value = new AbortController()
    await load_application_config()
    await load_query()
    const wait_promises = new Array<Promise<any>>()
    wait_promises.push(calculate_dnote())
    wait_promises.push(calclate_checked_aggregate())
    Promise.all(wait_promises).then(() => is_loading.value = false)
}

async function load_query(): Promise<void> {
    cloned_query.value = props.query.clone()
}

async function load_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    await cloned_application_config.value.parse_template_and_struct()
}

async function calculate_dnote(): Promise<void> {
    abort_controller.value.abort()
    abort_controller.value = new AbortController()
    extruct_location_kyous()
    extruct_people_kyous()
    extruct_nlog_kyous()
    calc_total_awake_time() // 時間合算
    calc_total_sleep_time() //時間合算
    calc_total_work_time() //時間合算
    calc_total_tabaco_record_count() //条件付き件数
    calc_average_lantana_mood() // 平均
    calc_total_git_addition_deletion_count() //合算
    calc_total_plus_minus_nlogs()
}

async function calclate_checked_aggregate(): Promise<void> {
    calc_checked_nlog()
    calc_checked_timeis()
}

async function extruct_location_kyous(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    location_timeis_kmemo_kyous.value.splice(0)

    // timeisとkmemoのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_extruct_location_kyous = cloned_query.value.clone()
    query_for_extruct_location_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_location_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_kmemo_rep = false
        let match_timeis_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_kmemo_rep = rep_struct.rep_name.toLowerCase().startsWith("kmemo")
                match_timeis_rep = rep_struct.rep_name.toLowerCase().startsWith("timeis")
                break
            }
        }
        if (exist_in_sidebar_query && (match_kmemo_rep || match_timeis_rep)) {
            query_for_extruct_location_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_location_kyous.tags = ["ろ"]

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_location_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    location_timeis_kmemo_kyous.value = res.kyous
}

async function extruct_people_kyous(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    people_timeis_kmemo_kyous.value.splice(0)

    // timeisとkmemoのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_extruct_people_kyous = cloned_query.value.clone()
    query_for_extruct_people_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_people_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_kmemo_rep = false
        let match_timeis_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_kmemo_rep = rep_struct.rep_name.toLowerCase().startsWith("kmemo")
                match_timeis_rep = rep_struct.rep_name.toLowerCase().startsWith("timeis")
                break
            }
        }
        if (exist_in_sidebar_query && (match_kmemo_rep || match_timeis_rep)) {
            query_for_extruct_people_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_people_kyous.tags = ["あ", "通話"]
    query_for_extruct_people_kyous.tags_and = false

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_people_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    people_timeis_kmemo_kyous.value = res.kyous
}

async function extruct_nlog_kyous(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    nlog_kyous.value.splice(0)

    // nlogのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_nlog_kyous = cloned_query.value.clone()
    query_for_nlog_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_nlog_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_nlog_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_nlog_rep = rep_struct.rep_name.toLowerCase().startsWith("nlog")
                break
            }
        }
        if (exist_in_sidebar_query && (match_nlog_rep)) {
            query_for_nlog_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_nlog_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    nlog_kyous.value = res.kyous
}

async function calc_total_awake_time(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    awake_timeis_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は覚醒
    const query_for_extruct_awake_kyous = cloned_query.value.clone()
    query_for_extruct_awake_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_awake_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_timeis_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_timeis_rep = rep_struct.rep_name.toLowerCase().startsWith("timeis")
                break
            }
        }
        if (exist_in_sidebar_query && (match_timeis_rep)) {
            query_for_extruct_awake_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_awake_kyous.tags = ["ぢ"]
    query_for_extruct_awake_kyous.use_words = true
    query_for_extruct_awake_kyous.keywords = "覚醒"
    query_for_extruct_awake_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_awake_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    awake_timeis_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < awake_timeis_kyous.value.length; i++) {
        const kyou = awake_timeis_kyous.value[i]
        wait_promises.push(kyou.load_typed_datas())
    }
    await Promise.all(wait_promises)

    let total_diff_milli_second = 0
    for (let i = 0; i < awake_timeis_kyous.value.length; i++) {
        const kyou = awake_timeis_kyous.value[i]
        const start_time = kyou.typed_timeis?.start_time
        const end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())

        const diff = moment.duration(moment(start_time).diff(moment(end_time))).asMilliseconds()
        total_diff_milli_second += Math.abs(diff.valueOf())
    }
    calclutated_total_awake_time.value = format_duration(total_diff_milli_second)
}

async function calc_total_sleep_time(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    sleep_timeis_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は睡眠
    const query_for_extruct_sleep_kyous = cloned_query.value.clone()
    query_for_extruct_sleep_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_sleep_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_timeis_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_timeis_rep = rep_struct.rep_name.toLowerCase().startsWith("timeis")
                break
            }
        }
        if (exist_in_sidebar_query && (match_timeis_rep)) {
            query_for_extruct_sleep_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_sleep_kyous.tags = ["ぢ"]
    query_for_extruct_sleep_kyous.use_words = true
    query_for_extruct_sleep_kyous.keywords = "睡眠"
    query_for_extruct_sleep_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_sleep_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    sleep_timeis_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < sleep_timeis_kyous.value.length; i++) {
        const kyou = sleep_timeis_kyous.value[i]
        wait_promises.push(kyou.load_typed_datas())
    }
    await Promise.all(wait_promises)

    let total_diff_milli_second = 0
    for (let i = 0; i < sleep_timeis_kyous.value.length; i++) {
        const kyou = sleep_timeis_kyous.value[i]
        const start_time = kyou.typed_timeis?.start_time
        const end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())

        const diff = moment.duration(moment(start_time).diff(moment(end_time))).asMilliseconds()
        total_diff_milli_second += Math.abs(diff.valueOf())
    }
    calclutated_total_sleep_time.value = format_duration(total_diff_milli_second)
}

async function calc_total_work_time(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    work_timeis_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_work_kyous = cloned_query.value.clone()
    query_for_extruct_work_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_work_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_timeis_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_timeis_rep = rep_struct.rep_name.toLowerCase().startsWith("timeis")
                break
            }
        }
        if (exist_in_sidebar_query && (match_timeis_rep)) {
            query_for_extruct_work_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_work_kyous.use_words = true
    query_for_extruct_work_kyous.keywords = "仕事"
    query_for_extruct_work_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_work_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    work_timeis_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < work_timeis_kyous.value.length; i++) {
        const kyou = work_timeis_kyous.value[i]
        wait_promises.push(kyou.load_typed_datas())
    }
    await Promise.all(wait_promises)

    let total_diff_milli_second = 0
    for (let i = 0; i < work_timeis_kyous.value.length; i++) {
        const kyou = work_timeis_kyous.value[i]
        const start_time = kyou.typed_timeis?.start_time
        const end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())

        const diff = moment.duration(moment(start_time).diff(moment(end_time))).asMilliseconds()
        total_diff_milli_second += Math.abs(diff.valueOf())
    }
    calclutated_total_work_time.value = format_duration(total_diff_milli_second)
}

async function calc_total_tabaco_record_count(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    tabaco_kmemo_kyous.value.splice(0)

    // kmemoのRepだけを検索対象とする
    // 対象タグは煙草
    const query_for_extruct_tabaco_kyous = cloned_query.value.clone()
    query_for_extruct_tabaco_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_tabaco_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_kmemo_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_kmemo_rep = rep_struct.rep_name.toLowerCase().startsWith("kmemo")
                break
            }
        }
        if (exist_in_sidebar_query && (match_kmemo_rep)) {
            query_for_extruct_tabaco_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)
    query_for_extruct_tabaco_kyous.tags = ["煙草"]

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_tabaco_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    tabaco_kmemo_kyous.value.push(...res.kyous)
    calclutated_tabaco_record_count.value = tabaco_kmemo_kyous.value.length
}

async function calc_average_lantana_mood(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    lantana_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_lantana_kyous = cloned_query.value.clone()
    query_for_extruct_lantana_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_lantana_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_lantana_rep = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_lantana_rep = rep_struct.rep_name.toLowerCase().startsWith("lantana")
                break
            }
        }
        if (exist_in_sidebar_query && (match_lantana_rep)) {
            query_for_extruct_lantana_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_lantana_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    lantana_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < lantana_kyous.value.length; i++) {
        const kyou = lantana_kyous.value[i]
        wait_promises.push(kyou.load_typed_lantana())
    }
    await Promise.all(wait_promises)

    let total_mood = 0
    let total_count = 0
    let average_mood = 0
    for (let i = 0; i < lantana_kyous.value.length; i++) {
        const kyou = lantana_kyous.value[i]
        const mood = kyou.typed_lantana?.mood
        if (mood) {
            total_mood += mood.valueOf()
            total_count++
        }
    }
    average_mood = total_mood / total_count
    calclated_average_lantana_mood.value = average_mood
}

async function calc_total_git_addition_deletion_count(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    git_commit_log_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_work_kyous = cloned_query.value.clone()
    query_for_extruct_work_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_work_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_git = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_git = rep_struct.rep_name.toLowerCase().startsWith("git")
                break
            }
        }
        if (exist_in_sidebar_query && (match_git)) {
            query_for_extruct_work_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_work_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    git_commit_log_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < git_commit_log_kyous.value.length; i++) {
        const kyou = git_commit_log_kyous.value[i]
        wait_promises.push(kyou.load_typed_git_commit_log())
    }
    await Promise.all(wait_promises)

    let total_addition = 0
    let total_deletion = 0
    for (let i = 0; i < git_commit_log_kyous.value.length; i++) {
        const kyou = git_commit_log_kyous.value[i]
        if (kyou.typed_git_commit_log) {
            total_addition += kyou.typed_git_commit_log.addition
            total_deletion += kyou.typed_git_commit_log.deletion
        }
    }
    calclutated_total_git_addition_count.value = total_addition
    calclutated_total_git_deletion_count.value = total_deletion
}

async function calc_total_plus_minus_nlogs(): Promise<void> {
    const sidebar_query = cloned_query.value.clone()
    nlog_kyous.value.splice(0)

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_work_kyous = cloned_query.value.clone()
    query_for_extruct_work_kyous.query_id = GkillAPI.get_instance().generate_uuid()
    query_for_extruct_work_kyous.reps.splice(0)
    let walk = (rep_struct: RepStructElementData) => { }
    walk = (rep_struct: RepStructElementData) => {
        let exist_in_sidebar_query = false
        let match_git = false
        for (let i = 0; i < sidebar_query.reps.length; i++) {
            if (sidebar_query.reps[i] === rep_struct.rep_name) {
                exist_in_sidebar_query = true
                match_git = rep_struct.rep_name.toLowerCase().startsWith("git")
                break
            }
        }
        if (exist_in_sidebar_query && (match_git)) {
            query_for_extruct_work_kyous.reps.push(rep_struct.rep_name)
        }
        if (rep_struct.children && rep_struct.children.length !== 0) {
            rep_struct.children.forEach(child_rep => walk(child_rep));
        }
    }
    walk(cloned_application_config.value.parsed_rep_struct)

    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_work_kyous

    req.query.parse_words_and_not_words()
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    nlog_kyous.value.push(...res.kyous)

    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < nlog_kyous.value.length; i++) {
        const kyou = nlog_kyous.value[i]
        wait_promises.push(kyou.load_typed_nlog())
    }
    await Promise.all(wait_promises)

    let total_plus_nlog = 0
    let total_minus_nlog = 0
    for (let i = 0; i < nlog_kyous.value.length; i++) {
        const kyou = nlog_kyous.value[i]
        if (kyou.typed_nlog && kyou.typed_nlog.amount) {
            if (kyou.typed_nlog.amount.valueOf() > 0) {
                total_plus_nlog += kyou.typed_nlog.amount.valueOf()
            } else {
                total_minus_nlog += kyou.typed_nlog.amount.valueOf()
            }
        }
    }
    calclutated_total_nlog_plus_amount.value = total_plus_nlog
    calclutated_total_nlog_minus_amount.value = total_minus_nlog
}

function format_duration(duration_milli_second: number): string {
    let diff_str = ""
    const offset_in_locale_milli_second = new Date().getTimezoneOffset().valueOf() * 60000
    const diff = duration_milli_second
    const diff_date = moment(diff + offset_in_locale_milli_second).toDate()
    if (diff_date.getFullYear() - 1970 !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getFullYear() - 1970 + "年"
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + "ヶ月"
    }
    if ((diff_date.getDate() - 1) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getDate() - 1) + "日"
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getHours()) + "時間"
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + "分"
    }
    if (diff_str === "") {
        diff_str += diff_date.getSeconds() + "秒"
    }
    return diff_str
}

async function calc_checked_timeis(): Promise<void> {
    const checked_timeis_kyous = new Array<Kyou>()
    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < props.checked_kyous.length; i++) {
        const kyou = props.checked_kyous[i]
        if (kyou.data_type.toLowerCase().startsWith("timeis")) {
            checked_timeis_kyous.push(kyou)
            wait_promises.push(kyou.load_typed_timeis())
        }
    }
    await Promise.all(wait_promises)

    let total_diff_milli_second = 0
    for (let i = 0; i < checked_timeis_kyous.length; i++) {
        const kyou = checked_timeis_kyous[i]
        const start_time = kyou.typed_timeis?.start_time
        const end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())

        const diff = moment.duration(moment(start_time).diff(moment(end_time))).asMilliseconds()
        total_diff_milli_second += Math.abs(diff.valueOf())
    }
    total_checked_time.value = format_duration(total_diff_milli_second)
}

async function calc_checked_nlog(): Promise<void> {
    const checked_nlog_kyous = new Array<Kyou>()
    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < props.checked_kyous.length; i++) {
        const kyou = props.checked_kyous[i]
        if (kyou.data_type.toLowerCase().startsWith("nlog")) {
            checked_nlog_kyous.push(kyou)
            wait_promises.push(kyou.load_typed_nlog())
        }
    }
    await Promise.all(wait_promises)

    let total_plus_nlog = 0
    let total_minus_nlog = 0
    for (let i = 0; i < checked_nlog_kyous.length; i++) {
        const kyou = checked_nlog_kyous[i]
        if (kyou.typed_nlog && kyou.typed_nlog.amount) {
            if (kyou.typed_nlog.amount.valueOf() > 0) {
                total_plus_nlog += kyou.typed_nlog.amount.valueOf()
            } else {
                total_minus_nlog += kyou.typed_nlog.amount.valueOf()
            }
        }
    }
    total_checked_nlog_plus_amount.value = total_plus_nlog
    total_checked_nlog_minus_amount.value = total_minus_nlog
}
</script>

<style lang="css">
.git_commit_log_message {
    white-space: pre-line;
}

.git_commit_addition {
    color: limegreen;
}

.git_commit_deletion {
    color: crimson;
}

.dnote_view {
    width: 625px;
    max-width: 625px;
    min-width: 625px;
}

.dnote_view .lantana_icon {
    position: relative;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}

.dnote_view .lantana_icon_fill {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.dnote_view .lantana_icon_harf_left {
    position: absolute;
    left: 0px;
    width: 10px !important;
    height: 20px !important;
    max-width: 10px !important;
    min-width: 10px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    object-fit: cover;
    object-position: 0 0;
    display: inline-block;
    z-index: 10;
}

.dnote_view .lantana_icon_harf_right {
    position: absolute;
    left: 0px;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 9;
}

.dnote_view .lantana_icon_none {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.dnote_view .gray {
    filter: grayscale(100%);
}

.dnote_view .lantana_icon_tr {
    width: calc(20px * 5);
    max-width: calc(20px * 5);
    min-width: calc(20px * 5);
}

.dnote_view .lantana_icon_td {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}
</style>
