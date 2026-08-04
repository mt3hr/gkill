import { i18n } from '@/i18n'
import { computed, onUnmounted, ref, watch, nextTick, type Ref } from 'vue'
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

    // 日付セルのクリックハンドラをまとめて外すためのもの（use-kyou-count-calendar.ts と同じ）
    let calendar_listener_abort: AbortController | null = null

    // ── State refs ──
    const date = ref(new Date(Date.now()))
    const events: Ref<Array<Record<string, unknown>>> = ref(new Array<Record<string, unknown>>())

    // ── Watchers ──
    watch(() => date.value, () => {
        nextTick(() => {
            set_handler_on_calendar_date_texts()
        })
    })

    // 参照の入れ替えだけでなく件数の増減でも更新する
    // （削除は splice で行われるため、参照だけ見ていると件数バッジが更新されない）
    watch([() => props.kyous, () => props.kyous.length], () => {
        update_events()
    })

    watch(() => props.mi_sort_type, () => {
        update_events()
    })

    // ── Business logic ──
    function get_kyou_date(kyou: Kyou): Date | null {
        // MiReKyouはmirekyou_*、Miはmi_*で来るのでどちらも同じ射影として扱う
        const is_projection = (suffix: string): boolean =>
            kyou.data_type === "mi_" + suffix || kyou.data_type === "mirekyou_" + suffix

        switch (props.mi_sort_type) {
            case MiSortType.create_time:
                return is_projection("create") ? kyou.related_time : null
            case MiSortType.estimate_start_time:
                return is_projection("start") ? kyou.related_time : null
            case MiSortType.estimate_end_time:
                return is_projection("end") ? kyou.related_time : null
            case MiSortType.limit_time:
                return is_projection("limit") ? kyou.related_time : null
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

    // 張り直すたびに前回ぶんを外す。理由は use-kyou-count-calendar.ts のコメント参照。
    function set_handler_on_calendar_date_texts(): void {
        calendar_listener_abort?.abort()
        calendar_listener_abort = new AbortController()
        const signal = calendar_listener_abort.signal

        // document 全体だと同じページの別カレンダーにも付いてしまうので自分の配下に限定する
        const root: ParentNode = kyou_counter_calendar.value?.$el ?? document
        const calendar_date_text_selector = ".v-calendar-weekly__day"
        root.querySelectorAll(calendar_date_text_selector).forEach((element) => {
            element.addEventListener('click', (() => {
                if (!element.textContent || element.textContent.trim() === "") {
                    return
                }
                const year = date.value.getFullYear().toString()
                const month = (date.value.getMonth() + 1).toString()
                const day = (element as HTMLElement).innerText.toString().split("\n")[0].split(" ").slice(-1)[0].replaceAll(i18n.global.t("DAY_TITLE"), "")
                clicked_date(moment(year + "-" + month + "-" + day).toDate())
            }), { signal })
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

    // ── Lifecycle ──
    onUnmounted(() => {
        calendar_listener_abort?.abort()
        calendar_listener_abort = null
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
