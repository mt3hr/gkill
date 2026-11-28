<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-checkbox v-model="use_calendar" @change="clicked_use_calendar_checkbox"
                :label="i18n.global.t('CALENDAR_QUERY_TITLE')" hide-details class="pb-0 mb-0" />
        </v-col>
        <v-spacer class="pa-0 ma-0" />
        <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
            <v-btn dark color="secondary" @click="clicked_clear_calendar_button" hide-details>{{
                i18n.global.t("CLEAR_TITLE") }}</v-btn>
        </v-col>
    </v-row>
    <VDatePicker v-show="use_calendar" class="calendar_query_date_picker" :max-width="312" :model-value="dates"
        :multible="true" :color="'primary'" :multiple="'range'" :year="calendar_year" :month="calendar_month"
        @wheel.prevent.stop="(e: WheelEvent) => onWheel(e)" @update:model-value="clicked_date" ref="calendar" />
    <!-- 開始と終了をそれぞれ1行にまとめる（日付の隣にその時刻）。
         日付欄はカレンダーの選択の写し。打ち直すとカレンダーもその月へ動く。
         空にすると範囲ごと落ちる（dates は「終了だけ」を表せない）。
         打ちかけの年を弾く規則は use-calendar-query.ts にある。
         時刻は空のままなら従来どおり日の境界（00:00:00.000 / 23:59:59.999）。
         空で「未指定」を表すのが唯一の手段なので clearable を外さないこと -->
    <v-row class="pa-0 ma-0 calendar_query_start_row" v-show="use_calendar">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-text-field :model-value="start_date_string" @update:model-value="updated_start_date" type="date"
                hide-details persistent-placeholder class="calendar_query_start_date_field"
                :label="i18n.global.t('CALENDAR_QUERY_START_DATE_TITLE')" min-width="160" />
        </v-col>
        <v-col cols="auto" class="pa-0 ma-0">
            <v-menu v-model="show_start_time_menu" :close-on-content-click="false" transition="scale-transition"
                offset-y min-width="auto">
                <template #activator="{ props: menu_props }">
                    <v-text-field v-model="start_time_string" hide-details clearable
                        class="calendar_query_start_time_field"
                        :label="i18n.global.t('CALENDAR_QUERY_START_TIME_TITLE')" readonly min-width="120"
                        v-bind="menu_props" />
                </template>
                <v-time-picker v-model="start_time_string" format="24hr"
                    @update:minute="show_start_time_menu = false" />
            </v-menu>
        </v-col>
    </v-row>
    <v-row class="pa-0 ma-0 calendar_query_end_row" v-show="use_calendar">
        <v-col cols="auto" class="pa-0 ma-0">
            <v-text-field :model-value="end_date_string" @update:model-value="updated_end_date" type="date"
                hide-details persistent-placeholder class="calendar_query_end_date_field"
                :label="i18n.global.t('CALENDAR_QUERY_END_DATE_TITLE')" min-width="160" />
        </v-col>
        <v-col cols="auto" class="pa-0 ma-0">
            <v-menu v-model="show_end_time_menu" :close-on-content-click="false" transition="scale-transition"
                offset-y min-width="auto">
                <template #activator="{ props: menu_props }">
                    <v-text-field v-model="end_time_string" hide-details clearable
                        class="calendar_query_end_time_field"
                        :label="i18n.global.t('CALENDAR_QUERY_END_TIME_TITLE')" readonly min-width="120"
                        v-bind="menu_props" />
                </template>
                <v-time-picker v-model="end_time_string" format="24hr" @update:minute="show_end_time_menu = false" />
            </v-menu>
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { CalendarQueryEmits } from './calendar-query-emits'
import type { CalendarQueryProps } from './calendar-query-props'
import { VDatePicker } from 'vuetify/components';
import { useCalendarQuery } from '@/classes/use-calendar-query'
import { ref } from 'vue'

const calendar = ref<InstanceType<typeof VDatePicker> | null>(null)

const props = defineProps<CalendarQueryProps>()
const emits = defineEmits<CalendarQueryEmits>()


const {
    use_calendar,
    calendar_year,
    calendar_month,
    dates,
    clicked_date,
    onWheel,
    clicked_clear_calendar_button,
    clicked_use_calendar_checkbox,
    start_time_string,
    end_time_string,
    show_start_time_menu,
    show_end_time_menu,
    start_date_string,
    end_date_string,
    updated_start_date,
    updated_end_date,
    get_use_calendar,
    get_start_time,
    get_end_time,
} = useCalendarQuery({ props, emits })

defineExpose({ get_use_calendar, get_start_time, get_end_time })
</script>
<style scoped>
/* レンジ背景のopacityを濃くする（デフォルト 0.1 → 0.5） */
:deep(.v-date-picker-month__range-bg--range) {
    opacity: 0.5 !important;
}
</style>
