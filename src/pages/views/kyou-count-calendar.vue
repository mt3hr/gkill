<template>
    <v-sheet>
        <v-calendar :width="350" :v-slot:event="events" :weeks-in-month="'static'"
            :v-model="new Date(Date.now())" :allowed-dates="(date: unknown): boolean => { return true }"
            ref="kyou_counter_calendar" :color="'primary'" :events="events" type="day" @wheel.prevent.stop="on_wheel">
            <template v-slot:event="{ day, allDay, event }">
                <p class="kyou_counter" @click="clicked_date(event.start as Date)">{{ event.title }}</p>
            </template>
        </v-calendar>
        <v-slider min="0" max="86399" v-model="slider_model" :label="time" />
    </v-sheet>
</template>
<script lang="ts" setup>
import { VCalendar } from 'vuetify/labs/components';
import type { KyouCountCalendarEmits } from './kyou-count-calendar-emits'
import type { KyouCountCalendarProps } from './kyou-count-calendar-props'
import { computed, ref, watch, nextTick, type Ref } from 'vue';
import moment from 'moment';
const kyou_counter_calendar = ref<InstanceType<typeof VCalendar> | null>(null)

const props = defineProps<KyouCountCalendarProps>()
const emits = defineEmits<KyouCountCalendarEmits>()

const dates: Ref<Array<Date>> = ref([moment('2024-11-10 00:00:00').toDate()])
const slider_model: Ref<number> = ref(86399)
const events: Ref<Array<any>> = ref(new Array<any>())

watch(props.kyous, () => {
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

    const event_list = new Array<any>()
    date_event_map.forEach((count: Number, date_str: string): void => {
        event_list.push({
            title: count.toString(),
            start: date_str,
            end: date_str,
        })
    })
    events.value = event_list
})

function on_wheel(e: any) {
    if (0 < e.deltaY) {
        next()
    } else {
        prev()
    }
}
function prev() {
    document.querySelectorAll("div.v-calendar.v-calendar-monthly > div:nth-child(1) > div > button:nth-child(2)").forEach((element, key, parent) => { (element as any).click() })
}
function next() {
    document.querySelectorAll("div.v-calendar.v-calendar-monthly > div:nth-child(1) > div > button:nth-child(3)").forEach((element, key, parent) => { (element as any).click() })
}
function clicked_date(date: Date): void {
    emits('requested_focus_time', moment(moment(date).format("yyyy-MM-DD") + " " + time.value).toDate())
}
const time = computed(() => {
    return ('00' + Math.floor(slider_model.value / 3600).toString()).slice(-2) + ":" +
        ('00' + (Math.floor(slider_model.value / 60) % 60).toString()).slice(-2) + ":" +
        ('00' + Math.floor(slider_model.value % 60).toString()).slice(-2)
})
nextTick(() => {
    document.querySelectorAll("div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div > div.v-calendar-weekly__day-label > button > span.v-btn__content").forEach((element, key, parent) => {
        element.addEventListener('click', (() => {
            let year = 0
            let month = 0
            kyou_counter_calendar.value?.daysInMonth.forEach(date => {
                if (moment((date.date as Date).toString()).date() === 14) {
                    year = date.year
                    month = date.month
                }
            })
            clicked_date(moment(year + "-" + month + "-" + element.textContent).toDate())
        }))
    })
})
</script>

<style lang="css">
.v-calendar-month__days>.v-calendar-month__day {
    min-height: 60px;
}

.v-calendar-weekly__day-alldayevents-container {
    display: none;
}

.kyou_counter {
    text-align: center;
    color: white;
    font-weight: bold;
    background-color: #2672ed;
    border-radius: 20px;
}
</style>