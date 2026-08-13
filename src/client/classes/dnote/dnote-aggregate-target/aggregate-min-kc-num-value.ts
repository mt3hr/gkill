import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import format_aggregated_number from "./format-aggregated-number";

export default class AggregateMinKCNumValue implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateMinKCNumValue()
    }
    async append_aggregate_element_value(aggregated_value_kc_num_value: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_aggregated_value_max_kc_num_value = aggregated_value_kc_num_value === null ? 0 : aggregated_value_kc_num_value as number
        let max_num_value = 0
        if (kyou.typed_kc) {
            if (typed_aggregated_value_max_kc_num_value > kyou.typed_kc.num_value) {
                max_num_value = kyou.typed_kc.num_value
            } else {
                max_num_value = typed_aggregated_value_max_kc_num_value
            }
        }
        return max_num_value
    }
    async result_to_string(kc_num_value: unknown): Promise<string> {
        return format_aggregated_number((kc_num_value === null ? 0 : kc_num_value) as number)
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateMinKCNumValue",
        }
    }
}