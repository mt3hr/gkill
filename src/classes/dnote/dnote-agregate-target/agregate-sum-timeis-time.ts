import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import { i18n } from "@/i18n";
import { format_duration } from "@/classes/format-date-time";

export default class AgregateSumTimeIsTime implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateSumTimeIsTime()
    }
    async append_agregate_element_value(agregated_value_unix_time_milli_second: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_agregated_value_unix_time_milli_second = agregated_value_unix_time_milli_second === null ? 0 : agregated_value_unix_time_milli_second as number

        let duration_milli_second = 0
        if (kyou.typed_timeis) {
            let start_time_trimed = kyou.typed_timeis.start_time
            if (find_kyou_query.use_calendar && find_kyou_query.calendar_start_date) {
                start_time_trimed = start_time_trimed.getTime() <= find_kyou_query.calendar_start_date.getTime() ? find_kyou_query.calendar_start_date : start_time_trimed
            }

            let end_time_trimed = kyou.typed_timeis.end_time ? kyou.typed_timeis.end_time : new Date(Date.now())
            if (find_kyou_query.use_calendar && find_kyou_query.calendar_end_date) {
                end_time_trimed = end_time_trimed.getTime() >= find_kyou_query.calendar_end_date.getTime() ? find_kyou_query.calendar_end_date : end_time_trimed
            }

            if ((start_time_trimed.getTime() < end_time_trimed.getTime())) {
                duration_milli_second = Math.abs(end_time_trimed.getTime() - start_time_trimed.getTime())
            } else {
                duration_milli_second = 0
            }
        }
        return typed_agregated_value_unix_time_milli_second + duration_milli_second
    }
    async result_to_string(duration_milli_second: any | null): Promise<string> {
        if (duration_milli_second === 0) {
            return ""
        }
        const diff = duration_milli_second
        return format_duration(diff)
    }
    to_json(): any {
        return {
            type: "AgregateSumTimeIsTime",
        }
    }
}