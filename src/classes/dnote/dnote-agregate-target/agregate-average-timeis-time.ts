import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import { i18n } from "@/i18n";
import AverageInfo from "./average-info";
import { format_duration } from "@/classes/format-date-time";

export default class AgregateAverageTimeisTime implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateAverageTimeisTime()
    }
    async append_agregate_element_value(average_value_timeis: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_timeis = average_value_timeis === null ? new AverageInfo() : (average_value_timeis as AverageInfo).clone()
        cloned_typed_average_info_timeis.total_value = cloned_typed_average_info_timeis.total_value === null ? 0 : cloned_typed_average_info_timeis.total_value as number

        let duration_milli_second = 0
        if (kyou.typed_timeis) {
            let start_time_trimed = kyou.typed_timeis!.start_time
            if (find_kyou_query.use_calendar && find_kyou_query.calendar_start_date) {
                start_time_trimed = start_time_trimed.getTime() <= find_kyou_query.calendar_start_date.getTime() ? find_kyou_query.calendar_start_date : start_time_trimed
            }

            let end_time_trimed = kyou.typed_timeis?.end_time ? kyou.typed_timeis!.end_time : new Date(Date.now())
            if (find_kyou_query.use_calendar && find_kyou_query.calendar_end_date) {
                end_time_trimed = end_time_trimed.getTime() >= find_kyou_query.calendar_end_date.getTime() ? find_kyou_query.calendar_end_date : end_time_trimed
            }

            if ((start_time_trimed.getTime() < end_time_trimed.getTime())) {
                duration_milli_second = Math.abs(end_time_trimed.getTime() - start_time_trimed.getTime())
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
        const diff = average_value
        return format_duration(diff)
    }
    to_json(): any {
        return {
            type: "AgregateAverageTimeisTime",
        }
    }
}