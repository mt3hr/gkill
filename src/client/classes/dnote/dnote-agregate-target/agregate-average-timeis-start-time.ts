import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import TimeOfDayAverageInfo, { milli_second_of_day } from "./time-of-day-average-info";
import { format_time_of_day } from "@/classes/format-date-time";

export default class AgregateAverageTimeIsStartTime implements DnoteAgregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAgregateTarget {
        return new AgregateAverageTimeIsStartTime()
    }
    async append_agregate_element_value(average_value_timeis: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const cloned_typed_average_info_timeis = average_value_timeis === null ? new TimeOfDayAverageInfo() : (average_value_timeis as TimeOfDayAverageInfo).clone()

        if (kyou.typed_timeis) {
            cloned_typed_average_info_timeis.append(milli_second_of_day(kyou.typed_timeis.start_time))
        }

        return cloned_typed_average_info_timeis
    }
    async result_to_string(average_value_timeis: unknown): Promise<string> {
        if (average_value_timeis === null) {
            return ""
        }
        return format_time_of_day((average_value_timeis as TimeOfDayAverageInfo).average_milli_second_of_day())
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AgregateAverageTimeIsStartTime",
        }
    }
}
