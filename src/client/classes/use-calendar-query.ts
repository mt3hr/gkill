'use strict'

import { ref, watch, nextTick, type Ref } from 'vue'
import moment from 'moment'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { CalendarQueryProps } from '@/pages/views/calendar-query-props'
import type { CalendarQueryEmits } from '@/pages/views/calendar-query-emits'

export function useCalendarQuery(options: {
    props: CalendarQueryProps
    emits: CalendarQueryEmits
}) {
    const { props, emits } = options

    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const now = moment().toDate()
    const calendar_year = ref(now.getFullYear())
    const calendar_month = ref(now.getMonth())
    const dates: Ref<Array<Date>> = ref([])

    watch(() => props.application_config, async () => {
        emits('inited')
    })

    watch(() => props.find_kyou_query, () => {
        if (props.find_kyou_query) {
            query.value = props.find_kyou_query.clone()
        } else {
            query.value = new FindKyouQuery()
        }

        const start_date = moment(query.value.calendar_start_date)
        const end_date = moment(query.value.calendar_end_date)
        const date_list = Array<Date>()
        if (query.value.calendar_start_date && query.value.calendar_end_date) {
            for (let date = start_date; date.unix() <= end_date.unix(); date = date.add(1, 'day')) {
                date_list.push(date.toDate())
            }
        } else {
            if (query.value.calendar_start_date) {
                date_list.push(start_date.toDate())
            }
            if (query.value.calendar_end_date) {
                date_list.push(start_date.toDate())
            }
        }
        dates.value = []
        dates.value = date_list

        if (!props.inited) {
            nextTick(() => {
                if (props.find_kyou_query.calendar_end_date) {
                    const calendar_end_date = moment(props.find_kyou_query.calendar_end_date);
                    calendar_year.value = calendar_end_date.toDate().getFullYear()
                    calendar_month.value = calendar_end_date.toDate().getMonth()
                }
            })
        }
    })

    function clicked_date(recved_dates: any): void {
        dates.value = recved_dates
        if (dates.value) {
            emits('request_update_dates', moment(dates.value[0]).toDate(), moment(dates.value[dates.value.length - 1]).add(1, 'day').add(-1, 'millisecond').toDate())
        } else {
            emits('request_update_dates', null, null)
        }
    }

    function on_wheel(e: any) {
        if (0 < e.deltaY) {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(3) > span.v-btn__content > i").forEach((el) => { (el as any).click() })
        } else {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(1) > span.v-btn__content > i").forEach((el) => { (el as any).click() })
        }
    }

    function clicked_clear_calendar_button(): void {
        emits('request_clear_calendar_query')
    }

    function clicked_use_calendar_checkbox(): void {
        emits('request_update_use_calendar_query', query.value.use_calendar)
    }

    function get_use_calendar(): boolean {
        return query.value.use_calendar
    }

    function get_start_date(): Date | null {
        if (dates.value.length >= 1) {
            return moment(moment(dates.value[0]).format("YYYY-MM-DD")).toDate()
        }
        return null
    }

    function get_end_date(): Date | null {
        if (dates.value.length >= 1) {
            return moment(moment(dates.value[dates.value.length - 1]).format("YYYY-MM-DD")).add(1, 'days').add(-1, 'milliseconds').toDate()
        }
        return null
    }

    return {
        query,
        calendar_year,
        calendar_month,
        dates,
        clicked_date,
        on_wheel,
        clicked_clear_calendar_button,
        clicked_use_calendar_checkbox,
        get_use_calendar,
        get_start_date,
        get_end_date,
    }
}
