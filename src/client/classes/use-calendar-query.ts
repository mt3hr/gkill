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
    // チェックボックスのUI状態。クエリ上は calendar_start_date / calendar_end_date の
    // null 判定が担うため、ローカルrefに分離してprops片方向同期する
    const use_calendar: Ref<boolean> = ref(props.find_kyou_query
        ? (props.find_kyou_query.calendar_start_date !== null || props.find_kyou_query.calendar_end_date !== null)
        : false)
    // 同一query_id内の null 着信（チェックオフ）ではローカルの日付選択を保持し、
    // query_id が変わったときだけ着信値で再構築するための同期済みquery_id
    const last_synced_query_id: Ref<string | null> = ref(null)

    watch(() => props.application_config, async () => {
        emits('inited')
    })

    watch(() => props.find_kyou_query, () => {
        if (props.find_kyou_query) {
            query.value = props.find_kyou_query.clone()
        } else {
            query.value = new FindKyouQuery()
        }

        const query_id_changed = last_synced_query_id.value !== query.value.query_id
        last_synced_query_id.value = query.value.query_id
        const has_dates = query.value.calendar_start_date !== null || query.value.calendar_end_date !== null

        // チェック状態を動かすのは「別の列に切り替わったとき」と「日付が載って着信したとき」だけ。
        // 同一列でのnull着信でここを触ると、同じ日付の2回クリック(Vuetifyのレンジ解除で
        // 選択が空になる)のたびにチェックが勝手に外れ、ピッカーごと消えてしまう
        if (query_id_changed || has_dates) {
            use_calendar.value = has_dates
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
        }
        // 同一query_idで両方null着信（チェックオフ／ピッカーでの選択解除）のときは
        // ローカルの日付選択もチェック状態も触らない（チェックの即時トグルで値が復活する）

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

    // VDatePicker は選択された日付の配列を返す。中身の型までは保証されないので unknown で受ける
    function clicked_date(recved_dates: Array<unknown>): void {
        if (!recved_dates || recved_dates.length === 0) {
            if (dates.value.length === 0) {
                // props同期の書き戻し(空のまま)はユーザー操作ではないのでemitしない
                return
            }
            dates.value = []
            emits('request_update_dates', null, null)
            return
        }
        // Vuetify4のrange modeは[start, end]の2要素のみを受け付ける。
        // 中間日付を全て含む配列が来ても先頭と末尾のみ残す。
        // VDatePicker は Date を返すが、型としては保証されないのでここで揃える
        const to_date = (value: unknown): Date => value instanceof Date ? value : moment(value as string).toDate()
        const first = to_date(recved_dates[0])
        const last = to_date(recved_dates[recved_dates.length - 1])
        const new_dates = recved_dates.length === 1 ? [first] : [first, last]
        // VDatePickerはprops同期で受け取ったmodel値を正規化して書き戻すことがある。
        // 日付が変わらない書き戻しはユーザー操作ではないのでemitしない
        // (get_start_date/get_end_dateは日granularityなので比較も日で足りる)
        const is_same_dates = new_dates.length === dates.value.length
            && new_dates.every((date, i) => moment(date).isSame(moment(dates.value[i]), 'day'))
        if (is_same_dates) {
            return
        }
        dates.value = new_dates
        emits('request_update_dates', moment(first).toDate(), moment(last).add(1, 'day').add(-1, 'millisecond').toDate())
    }

    function onWheel(e: WheelEvent) {
        if (0 < e.deltaY) {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(3) > span.v-btn__content > i").forEach((el) => { (el as HTMLElement).click() })
        } else {
            document.querySelectorAll("div.v-sheet.v-picker.v-date-picker.v-date-picker--month > div.v-picker__body > div.v-date-picker-controls > div.v-date-picker-controls__month > button:nth-child(1) > span.v-btn__content > i").forEach((el) => { (el as HTMLElement).click() })
        }
    }

    function clicked_clear_calendar_button(): void {
        // クリアは選択そのものを捨てる。既定期間が設定されていれば直後に既定の範囲が
        // 着信して選び直されるが、未設定(rykv_default_period === -1)だと返ってくるのが
        // null/nullで再構築の条件に入らないため、ここで落とさないと古い選択が残って
        // 次の編集で復活してしまう
        dates.value = []
        emits('request_clear_calendar_query')
    }

    function clicked_use_calendar_checkbox(): void {
        emits('request_update_use_calendar_query', use_calendar.value)
    }

    function get_use_calendar(): boolean {
        return use_calendar.value
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
        use_calendar,
        calendar_year,
        calendar_month,
        dates,
        clicked_date,
        onWheel,
        clicked_clear_calendar_button,
        clicked_use_calendar_checkbox,
        get_use_calendar,
        get_start_date,
        get_end_date,
    }
}
