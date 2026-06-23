import { i18n } from '@/i18n'
import { computed, ref, watch, nextTick, type Ref } from 'vue'
import type { MiKyouCountCalendarProps } from '@/pages/views/mi-kyou-count-calendar-props'
import type { MiKyouCountCalendarEmits } from '@/pages/views/mi-kyou-count-calendar-emits'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { Kyou } from '@/classes/datas/kyou'
import moment from 'moment'
import type { ComponentRef } from '@/classes/component-ref'

export function useMiKyouCountCalendar(options: {
    props: MiKyouCountCalendarProps,
    emits: MiKyouCountCalendarEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_counter_calendar = ref<ComponentRef | null>(null)

    // ── State refs ──
    const date = ref(new Date(Date.now()))
    const events: Ref<Array<Record<string, unknown>>> = ref(new Array<Record<string, unknown>>())

    // ── Watchers ──
    watch(() => date.value, () => {
        nextTick(() => {
            set_handler_on_calendar_date_texts()
        })
    })

    watch(() => props.kyous, () => {
        update_events()
    })

    watch(() => props.mi_sort_type, () => {
        update_events()
    })

    // ── Business logic ──
    function get_kyou_date(kyou: Kyou): Date | null {
        switch (props.mi_sort_type) {
            case MiSortType.create_time:
                return kyou.data_type === "mi_create" ? kyou.related_time : null
            case MiSortType.estimate_start_time:
                return kyou.data_type === "mi_start" ? kyou.related_time : null
            case MiSortType.estimate_end_time:
                return kyou.data_type === "mi_end" ? kyou.related_time : null
            case MiSortType.limit_time:
                return kyou.data_type === "mi_limit" ? kyou.related_time : null
            default:
                return kyou.related_time
        }
    }

    function update_events(): void {
        events.value.splice(0)
        if (!props.kyous) {
            return
        }
        const date_event_map: Map<string, number> = new Map<string, number>()
        for (let i = 0; i < props.kyous.length; i++) {
            const kyou = props.kyous[i]
            const target_date = get_kyou_date(kyou)
            if (!target_date) {
                continue
            }
            const date_str = moment(target_date).format("yyyy-MM-DD")
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

    function clicked_date(clicked: Date): void {
        emits('requested_focus_time', moment(moment(clicked).format("yyyy-MM-DD") + " 00:00:00").toDate())
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

    // ── Computed ──
    const calendar_year_month = computed(() => {
        return date.value.getFullYear().toString() + "/" + ("0" + (date.value.getMonth() + 1).toString()).slice(-2)
    })

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
        events,

        // Computed
        calendar_year_month,

        // Business logic
        add_months,
        on_wheel,
    }
}
