<template>

</template>
<script lang="ts" setup>
import { computed } from 'vue'
import type { AggregateViewEmits } from './aggregate-view-emits'
import type { AggregateViewProps } from './aggregate-view-props'
import type { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'

const props = defineProps<AggregateViewProps>()
const emits = defineEmits<AggregateViewEmits>()
const kyous_count = computed(() => props.checked_kyous.length)
const nlogs_total_amount = computed(() => {
    let total_amount: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "nlog") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_nlog()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        if (kyou.typed_nlog) {
            total_amount = total_amount.valueOf() + kyou.typed_nlog.amount.valueOf()
        }
    })
    return total_amount
})
const nlogs_total_plus_amount = computed(() => {
    let plus_amount: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "nlog") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_nlog()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        if (kyou.typed_nlog) {
            if (kyou.typed_nlog.amount.valueOf() <= 0) {
                return
            }
            plus_amount = plus_amount.valueOf() + kyou.typed_nlog.amount.valueOf()
        }
    })
    return plus_amount
})
const nlogs_total_minus_amount = computed(() => {
    let minus_amount: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "nlog") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_nlog()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        if (kyou.typed_nlog) {
            if (kyou.typed_nlog.amount.valueOf() > 0) {
                return
            }
            minus_amount = minus_amount.valueOf() + kyou.typed_nlog.amount.valueOf()
        }
    })
    return minus_amount
})
const timeis_total_time_milli_second = computed(() => {
    let millisecond: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "timeis") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_timeis()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        if (kyou.typed_timeis) {
            const start_time = moment(kyou.typed_timeis.start_time)
            const end_time = kyou.typed_timeis.end_time ? moment(kyou.typed_timeis.end_time) : moment()
            const diff = start_time.diff(end_time)
            millisecond = diff //TODO あってる？
        }
    })
    return millisecond
})
const git_total_file_count = computed(() => {
    let total_file_count: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "git") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_git_commit_log()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        //TODO total_file_count = diff
    })
    return total_file_count
})
const git_total_add_row = computed(() => {
    let add_row: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "git") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_git_commit_log()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        //TODO add_row = diff
    })
    return add_row
})
const git_total_remove_row = computed(() => {
    let remove_row: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "git") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_git_commit_log()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        //TODO remove_row = diff
    })
    return remove_row
})
const lantanas_average_mood = computed(() => {
    let sum_mood: Number = 0
    let mood_count: Number = 0
    props.checked_kyous.forEach(async (kyou) => {
        if (kyou.data_type !== "lantana") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_lantana()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        mood_count = mood_count.valueOf() + 1
        if (kyou.typed_lantana) {
            sum_mood = sum_mood.valueOf() + kyou.typed_lantana.mood.valueOf()
        }
    })
    return sum_mood.valueOf() / mood_count.valueOf()
})
async function update_aggregate_view(): Promise<void> {
    throw new Error('Not implemented')
}
</script>
