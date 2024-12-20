<template>
    <v-sheet>
        <v-calendar :width="350" :weeks-in-month="'static'" :v-model="new Date(Date.now())" ref="kyou_counter_calendar"
            :events="events" @wheel.prevent.stop="on_wheel">
            <template v-slot:event="{ event }">
                <div class="kyou_count">
                    {{ event["title"] }}
                </div>
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

const slider_model: Ref<number> = ref(86399)
const events: Ref<Array<any>> = ref(new Array<any>())

watch(props.kyous, () => {
    update_events()
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
    const calendar_year_month_selector = "div.v-calendar.v-calendar-monthly > div:nth-child(1) > div > div"
    const calendar_date_cell_selectors = [
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+2):nth-child(-n+8)",
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+10):nth-child(-n+16)",
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+18):nth-child(-n+24)",
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+26):nth-child(-n+32)",
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+34):nth-child(-n+40)",
        "div.v-calendar.v-calendar-monthly > div.v-calendar__container.days__7 > div.v-calendar-month__days.days-with-weeknumbers__7.v-calendar-month__weeknumbers > div:nth-child(n+42):nth-child(-n+48)",
    ]
    const calendar_date_text_selector = "div.v-calendar-weekly__day-label > button > span.v-btn__content"
    document.querySelectorAll(calendar_year_month_selector).forEach(year_month_element => {
        calendar_date_cell_selectors.forEach(date_cell_selector => {
            document.querySelectorAll(date_cell_selector).forEach((element, key, parent) => {
                element.addEventListener('click', (() => {
                    if (!element.textContent || element.textContent.trim() === "") {
                        return
                    }
                    let year = year_month_element.textContent?.substring(0, 4)
                    let month = year_month_element.textContent?.substring(5, 7).replace("月", "")
                    let date: string | null = ""
                    element.querySelectorAll(calendar_date_text_selector).forEach(date_text_element => date = ("0" + date_text_element.textContent).slice(-2))
                    clicked_date(moment(year + "-" + month + "-" + date).toDate())
                }))
            })
        })
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

.kyou_count {
    color: white;
    background-color: rgb(var(--v-theme-primary));
    border-radius: 20px;
    width: 100%;
    font-weight: bold;
    text-align: center;
    font-size: 85%;
}

.v-calendar-header__title {
    font-size: 20px !important;
}
</style>