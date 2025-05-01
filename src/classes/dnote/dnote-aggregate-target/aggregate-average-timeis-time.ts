import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import moment from "moment";
import { i18n } from "@/i18n";
import AverageInfo from "./average-info";

export default class AggregateAverageTimeisTime implements DnoteAggregateTarget {
    from_json(_json: any): DnoteAggregateTarget {
        return new AggregateAverageTimeisTime()
    }
    async append_aggregate_element_value(average_value_timeis: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_timeis = average_value_timeis === null ? new AverageInfo() : (average_value_timeis as AverageInfo).clone()
        cloned_typed_average_info_timeis.total_value = cloned_typed_average_info_timeis.total_value === null ? 0 : cloned_typed_average_info_timeis.total_value as number

        let duration_milli_second = 0
        if (kyou.typed_timeis) {
            let start_time_trimed = kyou.typed_timeis!.start_time
            if (find_kyou_query.calendar_start_date) {
                start_time_trimed = start_time_trimed.getTime() <= find_kyou_query.calendar_start_date.getTime() ? find_kyou_query.calendar_start_date : start_time_trimed
            }

            let end_time_trimed = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
            if (find_kyou_query.calendar_end_date) {
                end_time_trimed = end_time_trimed.getTime() >= find_kyou_query.calendar_end_date.getTime() ? find_kyou_query.calendar_end_date : end_time_trimed
            }

            if ((start_time_trimed.getTime() < end_time_trimed.getTime())) {
                duration_milli_second = Math.abs(moment.duration(moment(start_time_trimed).diff(moment(end_time_trimed))).asMilliseconds())
            } else {
                duration_milli_second = 0
            }

            cloned_typed_average_info_timeis.total_value += duration_milli_second
            cloned_typed_average_info_timeis.total_count++
        }
        return cloned_typed_average_info_timeis
    }
    async result_to_string(average_value_timeis: any | null): Promise<string> {
        const average_value = (average_value_timeis === null || (average_value_timeis as AverageInfo).total_count === 0) ? 0 : (average_value_timeis as AverageInfo).total_value / (average_value_timeis as AverageInfo).total_count
        if (average_value === 0) {
            return ""
        }
        let diff_str = ""
        const offset_in_locale_milli_second = new Date().getTimezoneOffset().valueOf() * 60000
        const diff = average_value
        const diff_date = moment(diff + offset_in_locale_milli_second).toDate()
        if (diff_date.getFullYear() - 1970 !== 0) {
            if (diff_str !== "") {
                diff_str += " "
            }
            diff_str += diff_date.getFullYear() - 1970 + i18n.global.t("YEAR_SUFFIX")
        }
        if (diff_date.getMonth() !== 0) {
            if (diff_str !== "") {
                diff_str += " "
            }
            diff_str += (diff_date.getMonth() + 1) + i18n.global.t("MONTH_SUFFIX")
        }
        if ((diff_date.getDate() - 1) !== 0) {
            if (diff_str !== "") {
                diff_str += " "
            }
            diff_str += (diff_date.getDate() - 1) + i18n.global.t("DAY_SUFFIX")
        }
        if (diff_date.getHours() !== 0) {
            if (diff_str !== "") {
                diff_str += " "
            }
            diff_str += (diff_date.getHours()) + i18n.global.t("HOUR_SUFFIX")
        }
        if (diff_date.getMinutes() !== 0) {
            if (diff_str !== "") {
                diff_str += " "
            }
            diff_str += diff_date.getMinutes() + i18n.global.t("MINUTE_SUFFIX")
        }
        if (diff_str === "") {
            diff_str += diff_date.getSeconds() + i18n.global.t("SECOND_SUFFIX")
        }
        return diff_str
    }
    to_json(): any {
        return {
            type: "AggregateAverageTimeisTime",
        }
    }
}