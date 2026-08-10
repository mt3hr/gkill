import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import { format_duration } from "@/classes/format-date-time";

export default class AggregateSumTimeIsTime implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateSumTimeIsTime()
    }
    async append_aggregate_element_value(aggregated_value_unix_time_milli_second: unknown, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_aggregated_value_unix_time_milli_second = aggregated_value_unix_time_milli_second === null ? 0 : aggregated_value_unix_time_milli_second as number

        let duration_milli_second = 0
        if (kyou.typed_timeis) {
            let start_time_trimed = kyou.typed_timeis.start_time
            if (find_kyou_query.calendar_start_date) {
                start_time_trimed = start_time_trimed.getTime() <= find_kyou_query.calendar_start_date.getTime() ? find_kyou_query.calendar_start_date : start_time_trimed
            }

            let end_time_trimed = kyou.typed_timeis.end_time ? kyou.typed_timeis.end_time : new Date(Date.now())
            if (find_kyou_query.calendar_end_date) {
                end_time_trimed = end_time_trimed.getTime() >= find_kyou_query.calendar_end_date.getTime() ? find_kyou_query.calendar_end_date : end_time_trimed
            }

            if ((start_time_trimed.getTime() < end_time_trimed.getTime())) {
                duration_milli_second = Math.abs(end_time_trimed.getTime() - start_time_trimed.getTime())
            } else {
                duration_milli_second = 0
            }
        }
        return typed_aggregated_value_unix_time_milli_second + duration_milli_second
    }
    async result_to_string(duration_milli_second: unknown): Promise<string> {
        if (duration_milli_second === 0) {
            return ""
        }
        const diff = duration_milli_second as number
        return format_duration(diff)
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateSumTimeIsTime",
        }
    }
}