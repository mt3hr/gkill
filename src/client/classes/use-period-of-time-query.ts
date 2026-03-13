import { type Ref, ref, watch, nextTick } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { PeriodOfTimeQueryEmits } from '@/pages/views/period-of-time-query-emits'
import type { PeriodOfTimeQueryProps } from '@/pages/views/period-of-time-query-props'
import moment from 'moment'
import { WeekOfDays } from '@/classes/api/find_query/week-of-days'

export function usePeriodOfTimeQuery(options: {
    props: PeriodOfTimeQueryProps,
    emits: PeriodOfTimeQueryEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const use_period_of_time: Ref<boolean> = ref(false)
    const show_period_of_time_start_time_menu: Ref<boolean> = ref(false)
    const show_period_of_time_end_time_menu: Ref<boolean> = ref(false)
    const period_of_time_start_time_string: Ref<string> = ref("")
    const period_of_time_end_time_string: Ref<string> = ref("")
    const week_of_days = ref<number[]>([])

    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
    const skip_emits_this_tick = ref(false)

    // ── Watchers: props -> local state ──
    watch(() => props.application_config, () => {
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        cloned_application_config.value = props.application_config.clone()
    })

    watch(() => props.find_kyou_query.use_period_of_time, (new_value: boolean, old_value: boolean) => {
        if (new_value === old_value) {
            return
        }
        skip_emits_this_tick.value = true
        nextTick(() => skip_emits_this_tick.value = false)
        use_period_of_time.value = props.find_kyou_query.use_period_of_time
    })

    watch(
        () => props.find_kyou_query.period_of_time_start_time_second,
        (newSec, oldSec) => {
            if (newSec === oldSec) return

            skip_emits_this_tick.value = true
            nextTick(() => (skip_emits_this_tick.value = false))

            period_of_time_start_time_string.value =
                newSec == null ? "" : moment.unix(newSec).format("HH:mm")
        }
    )

    watch(
        () => props.find_kyou_query.period_of_time_end_time_second,
        (newSec, oldSec) => {
            if (newSec === oldSec) return

            skip_emits_this_tick.value = true
            nextTick(() => (skip_emits_this_tick.value = false))

            period_of_time_end_time_string.value =
                newSec == null ? "" : moment.unix(newSec).format("HH:mm")
        }
    )

    watch(() => props.find_kyou_query.period_of_time_week_of_days, () => {
        week_of_days.value.splice(0)
        week_of_days.value.push(...props.find_kyou_query.period_of_time_week_of_days)
    })

    // ── Watchers: local state -> emits ──
    watch(() => use_period_of_time.value, () => {
        if (skip_emits_this_tick.value) {
            return
        }
        emits('request_update_use_period_of_time', use_period_of_time.value)
    })

    watch(() => period_of_time_start_time_string.value, () => {
        if (skip_emits_this_tick.value) {
            return
        }
        emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
    })

    watch(() => period_of_time_end_time_string.value, () => {
        if (skip_emits_this_tick.value) {
            return
        }
        emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
    })

    watch(() => week_of_days.value, () => {
        if (skip_emits_this_tick.value) {
            return
        }
        emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
    })

    // ── Business logic ──
    function get_use_period_of_time(): boolean {
        return use_period_of_time.value
    }

    function get_period_of_time_start_time_second(): number | null {
        if (period_of_time_start_time_string.value === "") return null
        const [h, m] = period_of_time_start_time_string.value.split(":").map(Number)
        return moment().startOf("day").hour(h).minute(m).second(0).unix()
    }

    function get_period_of_time_end_time_second(): number | null {
        if (period_of_time_end_time_string.value === "") return null
        const [h, m] = period_of_time_end_time_string.value.split(":").map(Number)
        return moment().startOf("day").hour(h).minute(m).second(0).unix()
    }

    function get_period_of_time_week_of_days(): Array<number> {
        return week_of_days.value.concat()
    }

    function to_week_of_days_label(num: WeekOfDays): string {
        switch (num) {
            case WeekOfDays.SunDay:
                return "SUNDAY_TITLE"
            case WeekOfDays.MonDay:
                return "MONDAY_TITLE"
            case WeekOfDays.TuesDay:
                return "TUESDAY_TITLE"
            case WeekOfDays.WednesDay:
                return "WEDNESDAY_TITLE"
            case WeekOfDays.ThrusDay:
                return "THURSDAY_TITLE"
            case WeekOfDays.FriDay:
                return "FRIDAY_TITLE"
            case WeekOfDays.SaturDay:
                return "SATURDAY_TITLE"
        }
        return ""
    }

    // ── Return ──
    return {
        // State
        use_period_of_time,
        show_period_of_time_start_time_menu,
        show_period_of_time_end_time_menu,
        period_of_time_start_time_string,
        period_of_time_end_time_string,
        week_of_days,

        // Exposed methods
        get_use_period_of_time,
        get_period_of_time_start_time_second,
        get_period_of_time_end_time_second,
        get_period_of_time_week_of_days,

        // Template helpers
        to_week_of_days_label,
    }
}
