import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import AverageInfo from "./average-info";

export default class AggregateAverageKCNumValue implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateAverageKCNumValue()
    }
    async append_aggregate_element_value(typed_average_info_kc_num_value: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const cloned_typed_average_info_kc_num_value = typed_average_info_kc_num_value === null ? new AverageInfo() : (typed_average_info_kc_num_value as AverageInfo).clone()
        cloned_typed_average_info_kc_num_value.total_value = cloned_typed_average_info_kc_num_value.total_value === null ? 0 : cloned_typed_average_info_kc_num_value.total_value as number

        let num_value = 0
        if (kyou.typed_kc) {
            num_value += kyou.typed_kc.num_value

            cloned_typed_average_info_kc_num_value.total_value += num_value
            cloned_typed_average_info_kc_num_value.total_count++
        }
        return cloned_typed_average_info_kc_num_value
    }
    async result_to_string(typed_average_info_kc_num_value: unknown): Promise<string> {
        const cloned_typed_average_info_kc_num_value = typed_average_info_kc_num_value === null ? new AverageInfo() : (typed_average_info_kc_num_value as AverageInfo).clone()
        cloned_typed_average_info_kc_num_value.total_value = cloned_typed_average_info_kc_num_value.total_value === null ? 0 : cloned_typed_average_info_kc_num_value.total_value as number
        return (cloned_typed_average_info_kc_num_value.total_count === 0 ? 0 : (cloned_typed_average_info_kc_num_value.total_value / cloned_typed_average_info_kc_num_value.total_count)).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateAverageKCNumValue",
        }
    }
}