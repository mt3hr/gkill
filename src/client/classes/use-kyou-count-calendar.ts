import { i18n } from '@/i18n'
import { computed, ref, watch, nextTick, type Ref } from 'vue'
import type { KyouCountCalendarProps } from '@/pages/views/kyou-count-calendar-props'
import type { KyouCountCalendarEmits } from '@/pages/views/kyou-count-calendar-emits'
import moment from 'moment'
import type { ComponentRef } from '@/classes/component-ref'

export function useKyouCountCalendar(options: {
    props: KyouCountCalendarProps,
    emits: KyouCountCalendarEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_counter_calendar = ref<ComponentRef | null>(null)

    // ── State refs ──
    const date = ref(new Date(Date.now()))
    const slider_model: Ref<number> = ref(props.for_mi ? 0 : 86399)
    const events: Ref<Array<Record<string, unknown>>> = ref(new Array<Record<string, unknown>>())

    // ── Computed ──
    const time = computed(() => {
        return ('00' + Math.floor(slider_model.value / 3600).toString()).slice(-2) + ":" +
            ('00' + (Math.floor(slider_model.value / 60) % 60).toString()).slice(-2) + ":" +
            ('00' + Math.floor(slider_model.value % 60).toString()).slice(-2)
    })

    // ── Watchers ──
    watch(() => date.value, () => {
        nextTick(() => {
            set_handler_on_calendar_date_texts()
        })
    })

    watch(() => props.kyous, () => {
        update_events()
    })

    watch(() => slider_model.value, () => {
        clicked_date(date.value)
    })

    // ── Business logic ──
    function update_events(): void {
        events.value.splice(0)
        if (!props.kyous) {
            return
        }
        const date_event_map: Map<string, number> = new Map<string, number>()
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

        date_event_map.forEach((count: number, date_str: string): void => {
            events.value.push({
                title: count.toString(),
                start: moment(date_str).toDate(),
                end: moment(date_str).add(1, 'day').add(-1, 'milliseconds').toDate(),
            })
        })
    }

    function on_wheel(e: WheelEvent) {
        if (0 < e.deltaY) {
            date.value = add_months(date.value, 1)
        } else {
            date.value = add_months(date.value, -1)
        }
    }

    function clicked_date(date: Date): void {
        emits('requested_focus_time', moment(moment(date).format("yyyy-MM-DD") + " " + time.value).toDate())
    }

    function set_handler_on_calendar_date_texts(): void {
        const calendar_date_text_selector = ".v-calendar-weekly__day"
        document.querySelectorAll(calendar_date_text_selector).forEach((element) => {
            element.addEventListener('click', (() => {
                if (!element.textContent || element.textContent.trim() === "") {
                    return
                }
                const year = date.value.getFullYear().toString()
                const month = (date.value.getMonth() + 1).toString()
                const day = (element as HTMLElement).innerText.toString().split("\n")[0].split(" ").slice(-1)[0].replaceAll(i18n.global.t("DAY_TITLE"), "")
                clicked_date(moment(year + "-" + month + "-" + day).toDate())
            }))
        })
    }

    function add_months(date: Date, diff: number) {
        const added_date = new Date(date)
        added_date.setMonth(added_date.getMonth() + diff)
        return added_date
    }

    // ── Init calls ──
    update_events()
    nextTick(() => {
        set_handler_on_calendar_date_texts()
    })

    // ── Return ──
    return {
        // Template refs
        kyou_counter_calendar,

        // State
        date,
        slider_model,
        events,
        time,

        // Business logic
        add_months,
        on_wheel,
    }
}
