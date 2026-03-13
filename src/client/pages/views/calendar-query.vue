<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="query.use_calendar" @change="clicked_use_calendar_checkbox"
                :label="i18n.global.t('CALENDAR_QUERY_TITLE')" hide-details class="pb-0 mb-0" />
        </v-col>
        <v-spacer class="pa-0 ma-0" />
        <v-col cols="auto" class="pb-0 mb-0 pr-0">
            <v-btn dark color="secondary" @click="clicked_clear_calendar_button" hide-details>{{
                i18n.global.t("CLEAR_TITLE") }}</v-btn>
        </v-col>
    </v-row>
    <VDatePicker v-show="query.use_calendar" class="calendar_query_date_picker" :max-width="300" :model-value="dates"
        :multible="true" :color="'primary'" :multiple="'range'" :year="calendar_year" :month="calendar_month"
        @wheel.prevent.stop="(e: any) => on_wheel(e)" @update:model-value="clicked_date" ref="calendar" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { CalendarQueryEmits } from './calendar-query-emits'
import type { CalendarQueryProps } from './calendar-query-props'
import { defineEmits, defineProps } from 'vue'
import { VDatePicker } from 'vuetify/components';
import { useCalendarQuery } from '@/classes/use-calendar-query'
const calendar = ref<InstanceType<typeof VDatePicker> | null>(null)

const props = defineProps<CalendarQueryProps>()
const emits = defineEmits<CalendarQueryEmits>()

import { ref } from 'vue'

const {
    query,
    calendar_year,
    calendar_month,
    dates,
    clicked_date,
    on_wheel,
    clicked_clear_calendar_button,
    clicked_use_calendar_checkbox,
    get_use_calendar,
    get_start_date,
    get_end_date,
} = useCalendarQuery({ props, emits })

defineExpose({ get_use_calendar, get_start_date, get_end_date })
</script>
