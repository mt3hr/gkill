<template>
    <div>
        <v-card class="dnote_view">
            <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
            <h1><span>{{ start_date_str }}</span><span
                    v-if="end_date_str !== '' && start_date_str != end_date_str">～</span><span
                    v-if="end_date_str !== '' && start_date_str != end_date_str">{{ end_date_str }}</span><span
                    v-if="start_date_str === '' && !(end_date_str !== '' && start_date_str != end_date_str)">{{
                        $t("DNOTE_WHOLE_PERIOD_TITLE") }}</span>
            </h1>
            <table>
                <tr>
                    <td>
                        <div>
                            <span>
                                {{ $t("DNOTE_AWAKE_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_awake_time !== ''">
                                {{ calclutated_total_awake_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_SLEEP_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_sleep_time !== ''">
                                {{ calclutated_total_sleep_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_WORK_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_work_time !== ''">
                                {{ calclutated_total_work_time }}
                            </span>
                        </div>
                    </td>
                    <td>
                        <div>
                            <span>
                                {{ $t("DNOTE_TABACO_TITLE") }}：
                            </span>
                            <span v-if="calclutated_tabaco_record_count !== -1">
                                <span>
                                    {{ calclutated_tabaco_record_count }}
                                </span>
                                <span>
                                    {{ $t("DNOTE_TABACO_COUNT") }}
                                </span>
                            </span>
                        </div>
                        <div style="display: flex;">
                            <span>
                                {{ $t("DNOTE_LANTANA_MOOD_TITLE") }}：
                            </span>
                            <LantanaFlowersView v-if="calclated_average_lantana_mood !== -1" :gkill_api="gkill_api"
                                :application_config="application_config" :mood="calclated_average_lantana_mood"
                                :editable="false" />
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_INCOME_AMOUNT_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_nlog_plus_amount !== -1">
                                <span class="amount_plus">
                                    {{ calclutated_total_nlog_plus_amount }}
                                </span>
                                <span>
                                    {{ $t("DNOTE_NLOG_YEN") }}
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_EXPENSE_AMOUNT_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_nlog_minus_amount !== -1">
                                <span class="amount_minus">
                                    {{ calclutated_total_nlog_minus_amount }}
                                </span>
                                <span>
                                    {{ $t("DNOTE_NLOG_YEN") }}
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_GIT_CODE_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_git_addition_count !== -1">
                                <span class="git_commit_addition">
                                    <span>
                                        + {{ calclutated_total_git_addition_count }}
                                    </span>
                                </span>
                                <span>
                                    {{ $t("DNOTE_GIT_CODE_COUNT") }}
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_GIT_CODE_TITLE") }}：
                            </span>
                            <span v-if="calclutated_total_git_deletion_count !== -1">
                                <span class="git_commit_deletion">
                                    <span>
                                        - {{ calclutated_total_git_deletion_count }}
                                    </span>
                                </span>
                                <span>
                                    {{ $t("DNOTE_GIT_CODE_COUNT") }}
                                </span>
                            </span>
                        </div>
                    </td>
                    <td>
                        <div>
                            <span>
                                {{ $t("DNOTE_TOTAL_TIME_TITLE") }}：
                            </span>
                            <span v-if="total_checked_time !== ''">
                                {{ total_checked_time }}
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_TOTAL_INCOME_AMOUNT_TITLE") }}：
                            </span>
                            <span v-if="total_checked_nlog_plus_amount !== -1">
                                <span class="amount_plus">
                                    {{ total_checked_nlog_plus_amount }}
                                </span>
                                <span>
                                    {{ $t("DNOTE_NLOG_YEN") }}
                                </span>
                            </span>
                        </div>
                        <div>
                            <span>
                                {{ $t("DNOTE_TOTAL_EXPENSE_AMOUNT_TITLE") }}：
                            </span>
                            <span v-if="total_checked_nlog_minus_amount !== -1">
                                <span class="amount_minus">
                                    {{ total_checked_nlog_minus_amount }}
                                </span>
                                <span>
                                    {{ $t("DNOTE_NLOG_YEN") }}
                                </span>
                            </span>
                        </div>
                    </td>
                </tr>
                <tr>
                    <td>
                        <h2>
                            {{ $t("DNOTE_AGGREGATE_NLOG_TITLE") }}
                        </h2>
                        <AggregateAmountListView :application_config="application_config" :gkill_api="gkill_api"
                            :last_added_tag="last_added_tag" :aggregate_ammounts="aggregate_amounts" />
                    </td>
                    <td>
                        <h2> {{ $t("DNOTE_AGGREGATE_LOCATION_TITLE") }} （{{ aggregate_locations.length
                            }}{{ $t("DNOTE_AGGREGATE_LOCATION_COUNT") }}）</h2>
                        <AggregateLocationListView :application_config="application_config" :gkill_api="gkill_api"
                            :last_added_tag="last_added_tag" :aggregate_locations="aggregate_locations" />
                    </td>
                    <td>
                        <h2>{{ $t("DNOTE_AGGREGATE_PEOPLE_TITLE") }}（{{ aggregate_peoples.length
                            }}{{ $t("DNOTE_AGGREGATE_PEOPLE_COUNT") }}）</h2>
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
    abort_controller.value = new AbortController()
    calculate_dnote()
    recalc_checked_aggregate()
}

async function abort(): Promise<void> {
    abort_controller.value.abort()
}

async function load_query(): Promise<void> {
    cloned_query.value = props.query.clone()
}

async function calculate_dnote(): Promise<void> {
    try {
        is_loading.value = true
        calclutated_total_awake_time.value = ""
        calclutated_total_sleep_time.value = ""
        calclutated_total_work_time.value = ""
        calclutated_tabaco_record_count.value = 0
        calclated_average_lantana_mood.value = 0
        calclutated_total_git_addition_count.value = 0
        calclutated_total_git_deletion_count.value = 0
        calclutated_total_nlog_plus_amount.value = 0
        calclutated_total_nlog_minus_amount.value = 0
        aggregate_amounts.value.splice(0)
        aggregate_locations.value.splice(0)
        aggregate_peoples.value.splice(0)
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

        // Promise.allをするとChromeでエラーが出るため全部await
        await calc_average_lantana_mood() // 平均
        await calc_total_git_addition_deletion_count() //合算
        await extruct_location_kyous()
        await extruct_people_kyous()
        await extruct_nlog_kyous()
        await calc_total_awake_time() // 時間合算
        await calc_total_sleep_time() //時間合算
        await calc_total_work_time() //時間合算
        await calc_total_tabaco_record_count() //条件付き件数
    } finally {
        is_loading.value = false
    }
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
    query_for_extruct_location_kyous.for_dnote_timeis_plaing_between_start_time_and_end_time = true

    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_location_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    location_timeis_kmemo_kyous.value = res.kyous

    const start_time = props.query.calendar_start_date ? props.query.calendar_start_date : new Date(0)
    const end_time = props.query.calendar_end_date ? props.query.calendar_end_date : new Date(Number.MAX_VALUE)

    const aggregated_locations = await aggregate_locations_from_kyous(location_timeis_kmemo_kyous.value, abort_controller.value, start_time, end_time)
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
    query_for_extruct_people_kyous.for_dnote_timeis_plaing_between_start_time_and_end_time = true

    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_people_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }

    const start_time = props.query.calendar_start_date ? props.query.calendar_start_date : new Date(0)
    const end_time = props.query.calendar_end_date ? props.query.calendar_end_date : new Date(Number.MAX_VALUE)

    people_timeis_kmemo_kyous.value = res.kyous
    const aggregated_peoples = await aggregate_peoples_from_kyous(people_timeis_kmemo_kyous.value, abort_controller.value, start_time, end_time)
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
    req.abort_controller = abort_controller.value
    req.query = query_for_nlog_kyous
    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
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

    const aggregate_nlogs = await aggregate_amounts_from_kyous(nlog_kyous.value, abort_controller.value)
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
    query_for_extruct_awake_kyous.for_dnote_timeis_plaing_between_start_time_and_end_time = true

    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_awake_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    awake_timeis_kyous.value.splice(0)
    awake_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < awake_timeis_kyous.value.length; i++) {
        const kyou = awake_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

    let total_diff_milli_second = 0
    for (let i = 0; i < awake_timeis_kyous.value.length; i++) {
        const kyou = awake_timeis_kyous.value[i]

        let start_time = kyou.typed_timeis!.start_time
        start_time = start_time.getTime() <= props.query.calendar_start_date!.getTime() ? props.query.calendar_start_date! : start_time

        let end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
        end_time = end_time.getTime() >= props.query.calendar_end_date!.getTime() ? props.query.calendar_end_date! : end_time

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
    query_for_extruct_sleep_kyous.for_dnote_timeis_plaing_between_start_time_and_end_time = true

    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_sleep_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    sleep_timeis_kyous.value.splice(0)
    sleep_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < sleep_timeis_kyous.value.length; i++) {
        const kyou = sleep_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

    let total_diff_milli_second = 0
    for (let i = 0; i < sleep_timeis_kyous.value.length; i++) {
        const kyou = sleep_timeis_kyous.value[i]

        let start_time = kyou.typed_timeis!.start_time
        start_time = start_time.getTime() <= props.query.calendar_start_date!.getTime() ? props.query.calendar_start_date! : start_time

        let end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
        end_time = end_time.getTime() >= props.query.calendar_end_date!.getTime() ? props.query.calendar_end_date! : end_time

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
    query_for_extruct_work_kyous.for_dnote_timeis_plaing_between_start_time_and_end_time = true

    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_work_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    work_timeis_kyous.value.splice(0)
    work_timeis_kyous.value.push(...res.kyous)

    for (let i = 0; i < work_timeis_kyous.value.length; i++) {
        const kyou = work_timeis_kyous.value[i]
        await kyou.load_typed_datas()
    }

    let total_diff_milli_second = 0
    for (let i = 0; i < work_timeis_kyous.value.length; i++) {
        const kyou = work_timeis_kyous.value[i]

        let start_time = kyou.typed_timeis!.start_time
        start_time = start_time.getTime() <= props.query.calendar_start_date!.getTime() ? props.query.calendar_start_date! : start_time

        let end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
        end_time = end_time.getTime() >= props.query.calendar_end_date!.getTime() ? props.query.calendar_end_date! : end_time

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
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_tabaco_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    tabaco_kmemo_kyous.value.push(...res.kyous)
    calclutated_tabaco_record_count.value = tabaco_kmemo_kyous.value.length
}

async function calc_average_lantana_mood(): Promise<void> {
    calclated_average_lantana_mood.value = -1

    const query_for_extruct_lantana_kyous = cloned_query.value.clone()
    query_for_extruct_lantana_kyous.query_id = props.gkill_api.generate_uuid()
    query_for_extruct_lantana_kyous.use_rep_types = true
    query_for_extruct_lantana_kyous.rep_types = ["lantana"]
    const req = new GetKyousRequest()
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_lantana_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
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
    req.abort_controller = abort_controller.value
    req.query = query_for_extruct_git_commit_log_kyous

    req.query.parse_words_and_not_words()
    const res = await props.gkill_api.get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
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

        let start_time = kyou.typed_timeis!.start_time
        start_time = start_time.getTime() <= props.query.calendar_start_date!.getTime() ? props.query.calendar_start_date! : start_time

        let end_time = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
        end_time = end_time.getTime() >= props.query.calendar_end_date!.getTime() ? props.query.calendar_end_date! : end_time

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
