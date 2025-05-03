import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import moment from "moment";
import { i18n } from "@/i18n";
import AggregateTargetDictionary from "../serialize/dnote-aggregate-target-dictionary";

export default class AggregateSumTimeisTime implements DnoteAggregateTarget{
    static from_json(_json: any): DnoteAggregateTarget {
        return new AggregateSumTimeisTime()
    }
    async append_aggregate_element_value(aggregated_value_unix_time_milli_second: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_aggregated_value_unix_time_milli_second = aggregated_value_unix_time_milli_second === null ? 0 : aggregated_value_unix_time_milli_second as number

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
        }
        return typed_aggregated_value_unix_time_milli_second + duration_milli_second
    }
    async result_to_string(duration_milli_second: any | null): Promise<string> {
        if (duration_milli_second === 0) {
            return ""
        }
        let diff_str = ""
        const offset_in_locale_milli_second = new Date().getTimezoneOffset().valueOf() * 60000
        const diff = duration_milli_second
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
            type: "AggregateSumTimeisTime",
        }
    }
}