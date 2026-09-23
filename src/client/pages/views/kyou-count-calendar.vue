<template>
    <v-sheet>

        <div class="d-flex align-center justify-space-between px-4 py-2">
            <v-btn icon @click="date = add_months(date, -1)">
                <VIcon icon="mdi-chevron-left" />
            </v-btn>
            <v-menu v-model="is_show_date_picker" :close-on-content-click="false" location="bottom"
                :z-index="3000">
                <template v-slot:activator="{ props: dateMenuProps }">
                    <v-btn variant="text" class="calendar_date calendar_date_button text-none" v-bind="dateMenuProps">
                        {{ date.getFullYear().toString() + "/" + ("0" + (date.getMonth() + 1).toString()).slice(-2) }}
                    </v-btn>
                </template>
                <v-date-picker v-model="date_picker_model" />
            </v-menu>
            <v-btn icon @click="date = add_months(date, 1)">
                <VIcon icon="mdi-chevron-right" />
            </v-btn>
        </div>

        <v-calendar :width="350" :model-value="date"
            @update:model-value="(updated_date: string) => { date = new Date(updated_date) }"
            ref="kyou_counter_calendar" :events="events" @wheel.prevent.stop="onWheel">
            <template v-slot:event="{ event }">
                <div class="kyou_count">
                    {{ event["title"] }}
                </div>
            </template>
        </v-calendar>
        <v-slider v-show="!props.for_mi" min="0" max="86399" v-model="slider_model" :label="time" />
    </v-sheet>
</template>
<script lang="ts" setup>
import { VCalendar } from 'vuetify/components';
import type { KyouCountCalendarEmits } from './kyou-count-calendar-emits'
import type { KyouCountCalendarProps } from './kyou-count-calendar-props'
import { useKyouCountCalendar } from '@/classes/use-kyou-count-calendar'

const props = defineProps<KyouCountCalendarProps>()
const emits = defineEmits<KyouCountCalendarEmits>()

const {
    // Template refs
    kyou_counter_calendar,

    // State
    date,
    slider_model,
    events,
    time,
    is_show_date_picker,
    date_picker_model,

    // Business logic
    add_months,
    onWheel,
} = useKyouCountCalendar({ props, emits })
</script>

<style lang="css">
.v-calendar .v-event {
    text-align: center;
}

.v-calendar.v-calendar-events .v-calendar-weekly__day {
    width: unset !important
}

.v-calendar-weekly.v-calendar {
    height: 430px !important;
}

.v-calendar-weekly__head {
    width: 300px;
}

.calendar_date {
    font-size: 26px;
}

/* 年月表示は日付ピッカーを開くボタン。v-btn の既定の文字サイズ・高さに負けないよう2クラスで指定する */
.v-btn.calendar_date_button {
    font-size: 26px;
    height: auto;
    letter-spacing: normal;
}
</style>
