import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import AverageInfo from "./average-info";
import { format_duration } from "@/classes/format-date-time";

export default class AgregateAverageTimeIsEndTime implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateAverageTimeIsEndTime()
    }
    async append_agregate_element_value(average_value_timeis: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_timeis = average_value_timeis === null ? new AverageInfo() : (average_value_timeis as AverageInfo).clone()
        cloned_typed_average_info_timeis.total_value = cloned_typed_average_info_timeis.total_value === null ? 0 : cloned_typed_average_info_timeis.total_value as number

        if (kyou.typed_timeis && kyou.typed_timeis.end_time) {
            const end_time = new Date(`1970-01-01T${kyou.typed_timeis.end_time.getHours().toString().padStart(2, '0')}:${kyou.typed_timeis.end_time.getMinutes().toString().padStart(2, '0')}:${kyou.typed_timeis.end_time.getSeconds().toString().padStart(2, '0')}`).getTime()
            cloned_typed_average_info_timeis.total_value += end_time
            cloned_typed_average_info_timeis.total_count++
        }

        return cloned_typed_average_info_timeis
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
            type: "AgregateAverageTimeIsEndTime",
        }
    }
}