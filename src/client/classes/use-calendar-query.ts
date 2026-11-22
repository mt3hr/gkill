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
    // チェックボックスのUI状態。クエリ上は calendar_start_time / calendar_end_time の
    // null 判定が担うため、ローカルrefに分離してprops片方向同期する
    const use_calendar: Ref<boolean> = ref(props.find_kyou_query
        ? (props.find_kyou_query.calendar_start_time !== null || props.find_kyou_query.calendar_end_time !== null)
        : false)
    // 同一query_id内の null 着信（チェックオフ）ではローカルの日付選択を保持し、
    // query_id が変わったときだけ着信値で再構築するための同期済みquery_id
    const last_synced_query_id: Ref<string | null> = ref(null)
    // 開始・終了の時刻。"HH:mm" か null（= 未指定）。
    // null のときは従来どおり日の境界（00:00:00.000 / 23:59:59.999）になる。
    // 空で「未指定」を表すのが唯一の手段なので、入力欄には必ず clearable を付けること
    const start_time_string: Ref<string | null> = ref(null)
    const end_time_string: Ref<string | null> = ref(null)
    const show_start_time_menu: Ref<boolean> = ref(false)
    const show_end_time_menu: Ref<boolean> = ref(false)
    // 時刻欄の watch から出る emit を、props 着信の逆同期中だけ止める。
    // 止めないと「着信 → 時刻欄を書き換え → emit → 再検索」で往復が終わらない
    const is_syncing_from_props: Ref<boolean> = ref(false)

    // "HH:mm" を時・分へ。読めない値は未指定として扱う
    function parse_time_string(value: string | null): { hour: number, minute: number } | null {
        if (!value) {
            return null
        }
        const matched = /^(\d{1,2}):(\d{2})$/.exec(value)
        if (!matched) {
            return null
        }
        const hour = Number(matched[1])
        const minute = Number(matched[2])
        if (!Number.isFinite(hour) || !Number.isFinite(minute)) {
            return null
        }
        if (hour < 0 || 23 < hour || minute < 0 || 59 < minute) {
            return null
        }
        return { hour, minute }
    }

    // 日付 + 開始時刻。時刻が未指定ならその日の 00:00:00.000（改名前と同じ既定）
    function to_start_moment(date: Date): moment.Moment {
        const day = moment(moment(date).format("YYYY-MM-DD"))
        const parsed = parse_time_string(start_time_string.value)
        if (!parsed) {
            return day
        }
        return day.hour(parsed.hour).minute(parsed.minute).second(0).millisecond(0)
    }

    // 日付 + 終了時刻。時刻が未指定ならその日の 23:59:59.999（改名前と同じ既定）。
    // 指定されていれば「その分の末尾まで」（VTimePicker に秒の UI が無いので
    // 18:30 は 18:30:59.999 と読む。18:30:00.000 にすると、ちょうど 18:30 の記録1件しか入らない）
    function to_end_moment(date: Date): moment.Moment {
        const day = moment(moment(date).format("YYYY-MM-DD"))
        const parsed = parse_time_string(end_time_string.value)
        if (!parsed) {
            return day.add(1, 'days').add(-1, 'milliseconds')
        }
        return day.hour(parsed.hour).minute(parsed.minute).second(0).millisecond(0)
            .add(1, 'minutes').add(-1, 'milliseconds')
    }

    // 着信した日時から時刻欄を復元する。日の境界ちょうどは「未指定」なので空のまま
    // （埋めてしまうと、既定期間で開いただけの列に 00:00 / 23:59 が表示される）
    function start_time_string_from(date: Date | null): string | null {
        if (!date) {
            return null
        }
        const value = moment(date)
        if (value.hour() === 0 && value.minute() === 0 && value.second() === 0 && value.millisecond() === 0) {
            return null
        }
        return value.format('HH:mm')
    }

    function end_time_string_from(date: Date | null): string | null {
        if (!date) {
            return null
        }
        const value = moment(date)
        if (value.hour() === 23 && value.minute() === 59 && value.second() === 59 && value.millisecond() === 999) {
            return null
        }
        return value.format('HH:mm')
    }

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
        const has_dates = query.value.calendar_start_time !== null || query.value.calendar_end_time !== null

        // チェック状態を動かすのは「別の列に切り替わったとき」と「日付が載って着信したとき」だけ。
        // 同一列でのnull着信でここを触ると、同じ日付の2回クリック(Vuetifyのレンジ解除で
        // 選択が空になる)のたびにチェックが勝手に外れ、ピッカーごと消えてしまう
        if (query_id_changed || has_dates) {
            use_calendar.value = has_dates
            const start_date = moment(query.value.calendar_start_time)
            const date_list = Array<Date>()
            if (query.value.calendar_start_time && query.value.calendar_end_time) {
                // Vuetify4のrange modeは[start, end]の2要素のみを受け付ける。
                // 中間日付を全て詰めるとレンジではなく個別選択として解釈されてしまう。
                date_list.push(start_date.toDate())
                const end_day = moment(query.value.calendar_end_time).startOf('day')
                // startとendが同日の場合は1要素のみにする。
                // 2要素にするとVDatePickerが「レンジ完了状態」と判断し、
                // 次のクリックで新しいレンジが始まってしまうため。
                if (!end_day.isSame(start_date, 'day')) {
                    date_list.push(end_day.toDate())
                }
            } else {
                if (query.value.calendar_start_time) {
                    date_list.push(start_date.toDate())
                }
                if (query.value.calendar_end_time) {
                    date_list.push(moment(query.value.calendar_end_time).startOf('day').toDate())
                }
            }
            dates.value = date_list
            is_syncing_from_props.value = true
            start_time_string.value = start_time_string_from(query.value.calendar_start_time)
            end_time_string.value = end_time_string_from(query.value.calendar_end_time)
            nextTick(() => { is_syncing_from_props.value = false })
        }
        // 同一query_idで両方null着信（チェックオフ／ピッカーでの選択解除）のときは
        // ローカルの日付選択もチェック状態も触らない（チェックの即時トグルで値が復活する）

        if (!props.inited) {
            nextTick(() => {
                if (props.find_kyou_query.calendar_end_time) {
                    const calendar_end_time = moment(props.find_kyou_query.calendar_end_time);
                    calendar_year.value = calendar_end_time.toDate().getFullYear()
                    calendar_month.value = calendar_end_time.toDate().getMonth()
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
        // (get_start_time/get_end_timeは日granularityなので比較も日で足りる)
        const is_same_dates = new_dates.length === dates.value.length
            && new_dates.every((date, i) => moment(date).isSame(moment(dates.value[i]), 'day'))
        if (is_same_dates) {
            return
        }
        dates.value = new_dates
        emits('request_update_dates', to_start_moment(first).toDate(), to_end_moment(last).toDate())
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
        start_time_string.value = null
        end_time_string.value = null
        emits('request_clear_calendar_query')
    }

    function clicked_use_calendar_checkbox(): void {
        emits('request_update_use_calendar_query', use_calendar.value)
    }

    function get_use_calendar(): boolean {
        return use_calendar.value
    }

    function get_start_time(): Date | null {
        if (dates.value.length >= 1) {
            return to_start_moment(dates.value[0]).toDate()
        }
        return null
    }

    function get_end_time(): Date | null {
        if (dates.value.length >= 1) {
            return to_end_moment(dates.value[dates.value.length - 1]).toDate()
        }
        return null
    }

    // 時刻だけ動かしたときも検索をやり直す。
    // 日付が1つも選ばれていなければ時刻に意味が無いので黙っている
    watch([start_time_string, end_time_string], () => {
        if (is_syncing_from_props.value) {
            return
        }
        if (dates.value.length === 0) {
            return
        }
        emits('request_update_dates', get_start_time(), get_end_time())
    })

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
        start_time_string,
        end_time_string,
        show_start_time_menu,
        show_end_time_menu,
        get_use_calendar,
        get_start_time,
        get_end_time,
    }
}
