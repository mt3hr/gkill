<template>
    <v-sheet>

        <div class="d-flex align-center justify-space-between px-4 py-2">
            <v-btn icon @click="date = add_months(date, -1)">
                <VIcon icon="mdi-chevron-left" />
            </v-btn>
            <span class="calendar_date text-subtitle-1 font-weight-medium">
                {{ calendar_year_month }}
            </span>
            <v-btn icon @click="date = add_months(date, 1)">
                <VIcon icon="mdi-chevron-right" />
            </v-btn>
        </div>

        <v-calendar :width="350" :model-value="date"
            @update:model-value="(updated_date: string) => { date = new Date(updated_date) }"
            ref="kyou_counter_calendar" :events="events" @wheel.prevent.stop="on_wheel">
            <template v-slot:event="{ event }">
                <div class="kyou_count">
                    {{ event["title"] }}
                </div>
            </template>
        </v-calendar>
    </v-sheet>
</template>
<script lang="ts" setup>
import { VCalendar } from 'vuetify/components';
import type { MiKyouCountCalendarEmits } from './mi-kyou-count-calendar-emits'
import type { MiKyouCountCalendarProps } from './mi-kyou-count-calendar-props'
import { useMiKyouCountCalendar } from '@/classes/use-mi-kyou-count-calendar'

const props = defineProps<MiKyouCountCalendarProps>()
const emits = defineEmits<MiKyouCountCalendarEmits>()

const {
    // Template refs
    kyou_counter_calendar,

    // State
    date,
    events,

    // Computed
    calendar_year_month,

    // Business logic
    add_months,
    on_wheel,
} = useMiKyouCountCalendar({ props, emits })
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
</style>
