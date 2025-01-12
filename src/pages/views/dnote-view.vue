<template>
    <div>
        <v-card class="dnote_view">
            <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
            <h1><span>{{ start_date_str }}</span><span
                    v-if="end_date_str !== '' && start_date_str != end_date_str">～</span><span
                    v-if="end_date_str !== '' && start_date_str != end_date_str">{{ end_date_str }}</span><span
                    v-if="start_date_str === '' && !(end_date_str !== '' && start_date_str != end_date_str)">全期間</span>
            </h1>
            <table>
                <tr>
                    <td>
                        <div>
                            <span>
                                覚醒：
                            </span>
                            <span v-if="calclutated_total_awake_time !== ''">
                                {{ calclutated_total_awake_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                睡眠：
                            </span>
                            <span v-if="calclutated_total_sleep_time !== ''">
                                {{ calclutated_total_sleep_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                仕事：
                            </span>
                            <span v-if="calclutated_total_work_time !== ''">
                                {{ calclutated_total_work_time }}
                            </span>
                        </div>
                    </td>
                    <td>
                        <div>
                            <span>
                                煙草：
                            </span>
                            <span v-if="calclutated_tabaco_record_count !== -1">
                                <span>
                                    {{ calclutated_tabaco_record_count }}
                                </span>
                                <span>
                                    本
                                </span>
                            </span>
                        </div>
                        <div style="display: flex;">
                            <span>
                                気分：
                            </span>
                            <LantanaFlowersView v-if="calclated_average_lantana_mood !== -1" :gkill_api="gkill_api"
                                :application_config="application_config" :mood="calclated_average_lantana_mood"
                                :editable="false" />
                        </div>
                        <div>
                            <span>
                                収入：
                            </span>
                            <span v-if="calclutated_total_nlog_plus_amount !== -1">
                                <span class="amount_plus">
                                    {{ calclutated_total_nlog_plus_amount }}
                                </span>
                                <span>
                                    円
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                支出：
                            </span>
                            <span v-if="calclutated_total_nlog_minus_amount !== -1">
                                <span class="amount_minus">
                                    {{ calclutated_total_nlog_minus_amount }}
                                </span>
                                <span>
                                    円
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                コード：
                            </span>
                            <span v-if="calclutated_total_git_addition_count !== -1">
                                <span class="git_commit_addition">
                                    <span>
                                        + {{ calclutated_total_git_addition_count }}
                                    </span>
                                </span>
                                <span>
                                    行
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                コード：
                            </span>
                            <span v-if="calclutated_total_git_deletion_count !== -1">
                                <span class="git_commit_deletion">
                                    <span>
                                        - {{ calclutated_total_git_deletion_count }}
                                    </span>
                                </span>
                                <span>
                                    行
                                </span>
                            </span>
                        </div>
                    </td>
                    <td>
                        <div>
                            <span>
                                合計時間：
                            </span>
                            <span v-if="total_checked_time !== ''">
                                {{ total_checked_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                合計収入：
                            </span>
                            <span v-if="total_checked_nlog_plus_amount !== -1">
                                <span class="amount_plus">
                                    {{ total_checked_nlog_plus_amount }}
                                </span>
                                <span>
                                    円
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                合計支出：
                            </span>
                            <span v-if="total_checked_nlog_minus_amount !== -1">
                                <span class="amount_minus">
                                    {{ total_checked_nlog_minus_amount }}
                                </span>
                                <span>
                                    円
                                </span>
                            </span>
                        </div>
                    </td>
                </tr>
                <tr>
                    <td>
                        <h2>収支</h2>
                        <AggregateAmountListView :application_config="application_config" :gkill_api="gkill_api"
                            :last_added_tag="last_added_tag" :aggregate_ammounts="aggregate_amounts" />
                    </td>
                    <td>
                        <h2>場所（{{ aggregate_locations.length }}件）</h2>
                        <AggregateLocationListView :application_config="application_config" :gkill_api="gkill_api"
                            :last_added_tag="last_added_tag" :aggregate_locations="aggregate_locations" />
                    </td>
                    <td>
                        <h2>人（{{ aggregate_peoples.length }}件）</h2>
                        <AggregatePeopleListView :application_config="application_config" :gkill_api="gkill_api"
                            :last_added_tag="last_added_tag" :aggregate_peoples="aggregate_peoples" />
                    </td>
                </tr>
            </table>
        </v-card>
    </div>
</template>
<script setup lang="ts">
import type { DnoteEmits } from './dnote-emits'
import type { DnoteViewProps } from './dnote-view-props'

import { computed, nextTick, ref, type Ref } from 'vue'

import moment from 'moment'
import type { Kyou } from '@/classes/datas/kyou'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { aggregate_locations_from_kyous, AggregateLocation } from '@/classes/api/dnote/aggregate-location'
import { aggregate_peoples_from_kyous, AggregatePeople } from '@/classes/api/dnote/aggregate-people'
import { aggregate_amounts_from_kyous, AggregateAmount } from '@/classes/api/dnote/aggregate-amount'
import AggregateAmountListView from './aggregate-amount-list-view.vue'
import AggregateLocationListView from './aggregate-location-list-view.vue'
import AggregatePeopleListView from './aggregate-people-list-view.vue'

const props = defineProps<DnoteViewProps>()
const emits = defineEmits<DnoteEmits>()
defineExpose({ recalc_all, recalc_checked_aggregate, abort })

const is_loading = ref(false)
const start_date_str: Ref<string> = computed(() => !cloned_query.value.calendar_start_date ? "" : (moment(cloned_query.value.calendar_start_date ? cloned_query.value.calendar_start_date : moment().toDate()).format("YYYY-MM-DD")))
const end_date_str: Ref<string> = computed(() => !cloned_query.value.calendar_end_date ? "" : (moment(cloned_query.value.calendar_end_date ? cloned_query.value.calendar_end_date : moment().toDate()).format("YYYY-MM-DD")))
// const date_kmemo0000_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
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
const calclutated_tabaco_record_count: Ref<Number> = ref(-1)
const calclated_average_lantana_mood: Ref<Number> = ref(-1)
const calclutated_total_git_addition_count: Ref<Number> = ref(-1)
const calclutated_total_git_deletion_count: Ref<Number> = ref(-1)
const calclutated_total_nlog_plus_amount: Ref<Number> = ref(-1)
const calclutated_total_nlog_minus_amount: Ref<Number> = ref(-1)

const aggregate_amounts: Ref<Array<AggregateAmount>> = ref(new Array<AggregateAmount>())
const aggregate_locations: Ref<Array<AggregateLocation>> = ref(new Array<AggregateLocation>())
const aggregate_peoples: Ref<Array<AggregatePeople>> = ref(new Array<AggregatePeople>())

const total_checked_time: Ref<string> = ref("")
const total_checked_nlog_plus_amount: Ref<Number> = ref(-1)
const total_checked_nlog_minus_amount: Ref<Number> = ref(-1)

const abort_controller: Ref<AbortController> = ref(new AbortController())
const cloned_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

async function recalc_all(): Promise<void> {
    nextTick(() => { ((async () => is_loading.value = true)()); })
    abort_controller.value = new AbortController()
    const wait_promises = new Array<Promise<any>>()
    wait_promises.push(calculate_dnote())
    wait_promises.push(recalc_checked_aggregate())
    Promise.all(wait_promises).then(() => is_loading.value = false)
}

async function abort(): Promise<void> {
    abort_controller.value.abort()
}

async function load_query(): Promise<void> {
    cloned_query.value = props.query.clone()
}

async function calculate_dnote(): Promise<void> {
    location_timeis_kmemo_kyous.value.splice(0)
    people_timeis_kmemo_kyous.value.splice(0)
    awake_timeis_kyous.value.splice(0)
    sleep_timeis_kyous.value.splice(0)
    work_timeis_kyous.value.splice(0)
    tabaco_kmemo_kyous.value.splice(0)
    lantana_kyous.value.splice(0)
    git_commit_log_kyous.value.splice(0)
    nlog_kyous.value.splice(0)
    await load_query()
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
}

async function recalc_checked_aggregate(): Promise<void> {
    calc_checked_nlog()
    calc_checked_timeis()
}

async function extruct_location_kyous(): Promise<void> {
    // timeisとkmemoのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_extruct_location_kyous = cloned_query.value.clone()
    query_for_extruct_location_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_location_kyous.use_rep_types = true
    query_for_extruct_location_kyous.rep_types = ["kmemo", "timeis"]
    query_for_extruct_location_kyous.tags = ["ろ"]

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_location_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    location_timeis_kmemo_kyous.value = res.kyous

    const aggregated_locations = await aggregate_locations_from_kyous(location_timeis_kmemo_kyous.value)
    aggregated_locations.sort((a, b) => b.duration_milli_second - a.duration_milli_second)
    aggregate_locations.value = aggregated_locations
}

async function extruct_people_kyous(): Promise<void> {
    // timeisとkmemoのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_extruct_people_kyous = cloned_query.value.clone()
    query_for_extruct_people_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_people_kyous.use_rep_types = true
    query_for_extruct_people_kyous.rep_types = ["timeis", "kmemo"]
    query_for_extruct_people_kyous.tags = ["あ", "通話"]
    query_for_extruct_people_kyous.tags_and = false

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_people_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    people_timeis_kmemo_kyous.value = res.kyous
    const aggregated_peoples = await aggregate_peoples_from_kyous(people_timeis_kmemo_kyous.value)
    aggregated_peoples.sort((a, b) => b.duration_milli_second - a.duration_milli_second)
    aggregate_peoples.value = aggregated_peoples
}

async function extruct_nlog_kyous(): Promise<void> {
    calclutated_total_nlog_plus_amount.value = -1
    calclutated_total_nlog_minus_amount.value = -1

    // nlogのRepだけを検索対象とする
    // それ以外はサイドバー条件を継承する
    const query_for_nlog_kyous = cloned_query.value.clone()
    query_for_nlog_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_nlog_kyous.use_rep_types = true
    query_for_nlog_kyous.rep_types = ["nlog"]
    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_nlog_kyous
    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    nlog_kyous.value = res.kyous

    for (let i = 0; i < nlog_kyous.value.length; i++) {
        const kyou = nlog_kyous.value[i]
        await kyou.load_typed_nlog()
    }

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

    const aggregate_nlogs = await aggregate_amounts_from_kyous(nlog_kyous.value)
    aggregate_nlogs.sort((a, b) => Math.abs(b.amount) - Math.abs(a.amount))
    aggregate_amounts.value = aggregate_nlogs

    calclutated_total_nlog_plus_amount.value = total_plus_nlog
    calclutated_total_nlog_minus_amount.value = total_minus_nlog
}

async function calc_total_awake_time(): Promise<void> {
    calclutated_total_awake_time.value = ""

    // timeisのRepだけを検索対象とする
    // 検索条件は覚醒
    const query_for_extruct_awake_kyous = cloned_query.value.clone()
    query_for_extruct_awake_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_awake_kyous.use_rep_types = true
    query_for_extruct_awake_kyous.rep_types = ["timeis"]
    query_for_extruct_awake_kyous.tags = ["ぢ"]
    query_for_extruct_awake_kyous.use_words = true
    query_for_extruct_awake_kyous.keywords = "覚醒"
    query_for_extruct_awake_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_awake_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    awake_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < awake_timeis_kyous.value.length; i++) {
        const kyou = awake_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

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
    calclutated_total_sleep_time.value = ""

    // timeisのRepだけを検索対象とする
    // 検索条件は睡眠
    const query_for_extruct_sleep_kyous = cloned_query.value.clone()
    query_for_extruct_sleep_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_sleep_kyous.use_rep_types = true
    query_for_extruct_sleep_kyous.rep_types = ["timeis"]
    query_for_extruct_sleep_kyous.tags = ["ぢ"]
    query_for_extruct_sleep_kyous.use_words = true
    query_for_extruct_sleep_kyous.keywords = "睡眠"
    query_for_extruct_sleep_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_sleep_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    sleep_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < sleep_timeis_kyous.value.length; i++) {
        const kyou = sleep_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

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
    calclutated_total_work_time.value = ""

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_work_kyous = cloned_query.value.clone()
    query_for_extruct_work_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_work_kyous.use_rep_types = true
    query_for_extruct_work_kyous.rep_types = ["timeis"]
    query_for_extruct_work_kyous.use_words = true
    query_for_extruct_work_kyous.keywords = "仕事"
    query_for_extruct_work_kyous.words_and = true

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_work_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    work_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < work_timeis_kyous.value.length; i++) {
        const kyou = work_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

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
    calclutated_tabaco_record_count.value = -1

    // kmemoのRepだけを検索対象とする
    // 対象タグは煙草
    const query_for_extruct_tabaco_kyous = cloned_query.value.clone()
    query_for_extruct_tabaco_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_tabaco_kyous.use_rep_types = true
    query_for_extruct_tabaco_kyous.rep_types = ["kmemo"]
    query_for_extruct_tabaco_kyous.tags = ["煙草"]

    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_tabaco_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
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
    calclated_average_lantana_mood.value = -1

    // timeisのRepだけを検索対象とする
    // 検索条件は仕事
    const query_for_extruct_lantana_kyous = cloned_query.value.clone()
    query_for_extruct_lantana_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_lantana_kyous.use_rep_types = true
    query_for_extruct_lantana_kyous.rep_types = ["lantana"]
    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_lantana_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    lantana_kyous.value.push(...res.kyous)

    for (let i = 0; i < lantana_kyous.value.length; i++) {
        const kyou = lantana_kyous.value[i]
        await kyou.load_typed_lantana()
    }

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
    calclutated_total_git_addition_count.value = -1
    calclutated_total_git_deletion_count.value = -1

    const query_for_extruct_git_commit_log_kyous = cloned_query.value.clone()
    query_for_extruct_git_commit_log_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_git_commit_log_kyous.use_rep_types = true
    query_for_extruct_git_commit_log_kyous.rep_types = ["git_commit_log"]
    const req = new GetKyousRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_git_commit_log_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    git_commit_log_kyous.value.push(...res.kyous)

    for (let i = 0; i < git_commit_log_kyous.value.length; i++) {
        const kyou = git_commit_log_kyous.value[i]
        await kyou.load_typed_git_commit_log()
    }

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
    total_checked_time.value = ""
    const checked_timeis_kyous = new Array<Kyou>()
    for (let i = 0; i < props.checked_kyous.length; i++) {
        const kyou = props.checked_kyous[i]
        if (kyou.data_type.toLowerCase().startsWith("timeis")) {
            checked_timeis_kyous.push(kyou)
            await kyou.load_typed_timeis()
        }
    }

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
    total_checked_nlog_plus_amount.value = -1
    total_checked_nlog_minus_amount.value = -1
    const checked_nlog_kyous = new Array<Kyou>()
    for (let i = 0; i < props.checked_kyous.length; i++) {
        const kyou = props.checked_kyous[i]
        if (kyou.data_type.toLowerCase().startsWith("nlog")) {
            checked_nlog_kyous.push(kyou)
            await kyou.load_typed_nlog()
        }
    }

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

.amount_plus {
    color: limegreen;
}

.amount_minus {
    color: crimson;
}

.dnote_view {
    position: relative;
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
