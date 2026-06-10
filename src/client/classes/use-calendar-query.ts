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
        const date_list = Array<Date>()
        if (query.value.calendar_start_date && query.value.calendar_end_date) {
            // Vuetify4のrange modeは[start, end]の2要素のみを受け付ける。
            // 中間日付を全て詰めるとレンジではなく個別選択として解釈されてしまう。
            date_list.push(start_date.toDate())
            const end_day = moment(query.value.calendar_end_date).startOf('day')
            // startとendが同日の場合は1要素のみにする。
            // 2要素にするとVDatePickerが「レンジ完了状態」と判断し、
            // 次のクリックで新しいレンジが始まってしまうため。
            if (!end_day.isSame(start_date, 'day')) {
                date_list.push(end_day.toDate())
            }
        } else {
            if (query.value.calendar_start_date) {
                date_list.push(start_date.toDate())
            }
            if (query.value.calendar_end_date) {
                date_list.push(moment(query.value.calendar_end_date).startOf('day').toDate())
            }
        }
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

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function clicked_date(recved_dates: any[]): void {
        if (!recved_dates || recved_dates.length === 0) {
            dates.value = []
            emits('request_update_dates', null, null)
            return
        }
        // Vuetify4のrange modeは[start, end]の2要素のみを受け付ける。
        // 中間日付を全て含む配列が来ても先頭と末尾のみ残す。
        const first = recved_dates[0]
        const last = recved_dates[recved_dates.length - 1]
        dates.value = recved_dates.length === 1 ? [first] : [first, last]
        emits('request_update_dates', moment(first).toDate(), moment(last).add(1, 'day').add(-1, 'millisecond').toDate())
    }

    function on_wheel(e: WheelEvent) {
        if (0 < e.deltaY) {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(3) > span.v-btn__content > i").forEach((el) => { (el as HTMLElement).click() })
        } else {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(1) > span.v-btn__content > i").forEach((el) => { (el as HTMLElement).click() })
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
