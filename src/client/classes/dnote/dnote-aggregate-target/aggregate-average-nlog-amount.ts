import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import AverageInfo from "./average-info";
import format_aggregated_number from "./format-aggregated-number";

export default class AggregateAverageNlogAmount implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateAverageNlogAmount()
    }
    async append_aggregate_element_value(typed_average_info_nlog_amount: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const cloned_typed_average_info_nlog_amount = typed_average_info_nlog_amount === null ? new AverageInfo() : (typed_average_info_nlog_amount as AverageInfo).clone()
        cloned_typed_average_info_nlog_amount.total_value = cloned_typed_average_info_nlog_amount.total_value === null ? 0 : cloned_typed_average_info_nlog_amount.total_value as number

        let amount = 0
        if (kyou.typed_nlog) {
            amount += kyou.typed_nlog.amount

            cloned_typed_average_info_nlog_amount.total_value += amount
            cloned_typed_average_info_nlog_amount.total_count++
        }
        return cloned_typed_average_info_nlog_amount
    }
    async result_to_string(typed_average_info_nlog_amount: unknown): Promise<string> {
        const cloned_typed_average_info_nlog_amount = typed_average_info_nlog_amount === null ? new AverageInfo() : (typed_average_info_nlog_amount as AverageInfo).clone()
        cloned_typed_average_info_nlog_amount.total_value = cloned_typed_average_info_nlog_amount.total_value === null ? 0 : cloned_typed_average_info_nlog_amount.total_value as number
        return format_aggregated_number(cloned_typed_average_info_nlog_amount.total_count === 0 ? 0 : (cloned_typed_average_info_nlog_amount.total_value / cloned_typed_average_info_nlog_amount.total_count))
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateAverageNlogAmount",
        }
    }
}