import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import moment from "moment";
import { i18n } from "@/i18n";
import AverageInfo from "./average-info";

export default class AgregateAverageTimeIsEndTime implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateAverageTimeIsEndTime()
    }
    async append_agregate_element_value(average_value_timeis: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_timeis = average_value_timeis === null ? new AverageInfo() : (average_value_timeis as AverageInfo).clone()
        cloned_typed_average_info_timeis.total_value = cloned_typed_average_info_timeis.total_value === null ? 0 : cloned_typed_average_info_timeis.total_value as number

        if (kyou.typed_timeis) {
            const end_time = moment((moment(0).format("YYYY-MM-DD ")) + moment(kyou.typed_timeis.end_time).format("HH:mm:ss")).toDate().getTime()
            cloned_typed_average_info_timeis.total_value += end_time
            cloned_typed_average_info_timeis.total_count++
        }

        return cloned_typed_average_info_timeis
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
            type: "AgregateAverageTimeIsEndTime",
        }
    }
}