<template>
    <v-sheet>

        <div class="d-flex align-center justify-space-between px-4 py-2">
            <v-btn icon @click="date = add_months(date, -1)">
                <VIcon icon="mdi-chevron-left" />
            </v-btn>
            <span class="calendar_date text-subtitle-1 font-weight-medium">
                {{ date.getFullYear().toString() + "/" + ("0" + (date.getMonth() + 1).toString()).slice(-2) }}
            </span>
            <v-btn icon @click="date = add_months(date, 1)">
                <VIcon icon="mdi-chevron-right" />
            </v-btn>
        </div>

        <v-calendar :width="350" :model-value="date"
            @update:model-value="(...updated_date: any[]) => { date = new Date(updated_date[0]) }"
            ref="kyou_counter_calendar" :events="events" @wheel.prevent.stop="on_wheel">
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
import { i18n } from '@/i18n'
import { VCalendar } from 'vuetify/components';
import type { KyouCountCalendarEmits } from './kyou-count-calendar-emits'
import type { KyouCountCalendarProps } from './kyou-count-calendar-props'
import { computed, ref, watch, nextTick, type Ref } from 'vue';
import moment from 'moment';
const kyou_counter_calendar = ref<InstanceType<typeof VCalendar> | null>(null)

const props = defineProps<KyouCountCalendarProps>()
const emits = defineEmits<KyouCountCalendarEmits>()

const date = ref(new Date(Date.now()))
const slider_model: Ref<number> = ref(props.for_mi ? 0 : 86399)
const events: Ref<Array<any>> = ref(new Array<any>())

watch(() => date.value, () => {
    nextTick(() => {
        set_handler_on_calendar_date_texts()
    })
})

watch(props.kyous, () => {
    update_events()
})
watch(() => slider_model.value, () => {
    clicked_date(date.value)
})

update_events()

function update_events(): void {
    events.value.splice(0)
    const date_event_map: Map<string, Number> = new Map<string, Number>()
    for (let i = 0; i < props.kyous.length; i++) {
        const kyou = props.kyous[i]
        const date_str = moment(kyou.related_time).format("yyyy-MM-DD")
        const count = date_event_map.get(date_str)?.valueOf()
        if (count) {
            date_event_map.set(date_str, count + 1)
        } else {
            date_event_map.set(date_str, 1)
        }
    }

    date_event_map.forEach((count: Number, date_str: string): void => {
        events.value.push({
            title: count.toString(),
            start: moment(date_str).toDate(),
            end: moment(date_str).add(1, 'day').add(-1, 'milliseconds').toDate(),
        })
    })
}

function on_wheel(e: any) {
    if (0 < e.deltaY) {
        date.value = add_months(date.value, 1)
    } else {
        date.value = add_months(date.value, -1)
    }
}
function clicked_date(date: Date): void {
    emits('requested_focus_time', moment(moment(date).format("yyyy-MM-DD") + " " + time.value).toDate())
}
const time = computed(() => {
    return ('00' + Math.floor(slider_model.value / 3600).toString()).slice(-2) + ":" +
        ('00' + (Math.floor(slider_model.value / 60) % 60).toString()).slice(-2) + ":" +
        ('00' + Math.floor(slider_model.value % 60).toString()).slice(-2)
})

function set_handler_on_calendar_date_texts(): void {
    const calendar_date_text_selector = ".v-calendar-weekly__day"
    document.querySelectorAll(calendar_date_text_selector).forEach((element) => {
        element.addEventListener('click', (() => {
            if (!element.textContent || element.textContent.trim() === "") {
                return
            }
            let year = date.value.getFullYear().toString()
            let month = (date.value.getMonth() + 1).toString()
            let day = (element as any).innerText.toString().split("\n")[0].split(" ").slice(-1)[0].replaceAll(i18n.global.t("DAY_TITLE"), "")
            clicked_date(moment(year + "-" + month + "-" + day).toDate())
        }))
    })

}

nextTick(() => {
    set_handler_on_calendar_date_texts()
})

function add_months(date: Date, diff: number) {
    const added_date = new Date(date)
    added_date.setMonth(added_date.getMonth() + diff)
    return added_date
}
</script>

<style lang="css">
.v-calendar .v-event {
    text-align: center;
}

.v-calendar.v-calendar-events .v-calendar-weekly__day {
    width: unset !important
}

.v-calendar-weekly.v-calendar.v-calendar-events {
    height: 430px !important;
}

.calendar_date {
    font-size: 26px;
}
</style>