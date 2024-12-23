<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="query.use_calendar" @change="clicked_use_calendar_checkbox" label="日付" hide-details
                class="pb-0 mb-0" />
        </v-col>
        <v-spacer class="pa-0 ma-0" />
        <v-col cols="auto" class="pa-0 ma-0">
            <v-btn @click="clicked_clear_calendar_button" hide-details class="pb-0 mb-0">クリア</v-btn>
        </v-col>
    </v-row>
    <VDatePicker v-show="query.use_calendar" class="calendar_query_date_picker" :max-width="300" :model-value="dates"
        :multible="true" :color="'primary'" :multiple="'range'" @wheel.prevent.stop="(e: any) => on_wheel(e)"
        @update:model-value="clicked_date" />
</template>
<script lang="ts" setup>
import moment from 'moment';
import type { CalendarQueryEmits } from './calendar-query-emits'
import type { CalendarQueryProps } from './calendar-query-props'
import { computed, ref, type Ref, defineEmits, defineProps, watch } from 'vue'
import { VDatePicker } from 'vuetify/components';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';

const props = defineProps<CalendarQueryProps>()
const emits = defineEmits<CalendarQueryEmits>()

const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const dates: Ref<Array<Date>> = ref([])
defineExpose({ get_use_calendar, get_start_date, get_end_date })

watch(() => props.application_config, async () => {
    emits('inited')
})

watch(() => props.find_kyou_query, () => {
    query.value = props.find_kyou_query.clone()

    const start_date = moment(props.find_kyou_query.calendar_start_date)
    const end_date = moment(props.find_kyou_query.calendar_end_date)
    const date_list = Array<Date>()
    if (query.value.calendar_start_date && query.value.calendar_end_date) {
        for (let date = start_date; date.unix() <= end_date.unix(); date = date.add(1, 'day')) {
            date_list.push(date.toDate())
        }
    } else {
        if (query.value.calendar_start_date) {
            date_list.push(start_date.toDate())
        }
        if (query.value.calendar_end_date) {
            date_list.push(start_date.toDate())
        }
    }
    dates.value = []
    dates.value = date_list
})

// 日付がクリックされた時、日時を更新してclicked_timeをemitする
function clicked_date(recved_dates: any): void {
    dates.value = recved_dates
    if (dates.value) {
        emits('request_update_dates', moment(dates.value[0]).toDate(), moment(dates.value[dates.value.length - 1]).toDate())
    } else {
        emits('request_update_dates', null, null)
    }
}
// カレンダーでホイールが転がされた時、下ならカレンダーを次の年月へ、上ならカレンダーを前の年月へ
function on_wheel(e: any) {
    if (0 < e.deltaY) {
        document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(2)").forEach((el, key, parent) => { (el as any).click() })
    } else {
        document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(1)").forEach((el, key, parent) => { (el as any).click() })
    }
}

function clicked_clear_calendar_button(): void {
    emits('request_clear_calendar_query')
}

function clicked_use_calendar_checkbox(): void {
    emits('request_update_use_calendar_query', query.value.use_calendar)
}

function get_use_calendar(): boolean {
    return query.value.use_calendar
}
function get_start_date(): Date | null {
    if (dates.value.length >= 1) {
        return dates.value[0]
    }
    return null
}
function get_end_date(): Date | null {
    if (dates.value.length >= 1) {
        return dates.value[dates.value.length-1]
    }
    return null
}
</script>
<style lang="css">
div.v-sheet.v-picker.v-date-picker.v-date-picker--year>div:nth-child(1),
div.v-sheet.v-picker.v-date-picker.v-date-picker--month>div:nth-child(1),
div.v-sheet.v-picker.v-date-picker.v-date-picker--months>div:nth-child(1) {
    display: none;
}
</style>
