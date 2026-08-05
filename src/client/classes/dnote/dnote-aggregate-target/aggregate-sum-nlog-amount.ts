import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";

export default class AggregateSumNlogAmount implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateSumNlogAmount()
    }
    async append_aggregate_element_value(aggregated_value_nlog_amount: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_aggregated_value_nlog_amount = aggregated_value_nlog_amount === null ? 0 : aggregated_value_nlog_amount as number
        let amount = 0
        if (kyou.typed_nlog) {
            amount += kyou.typed_nlog.amount
        }
        return typed_aggregated_value_nlog_amount + amount
    }
    async result_to_string(nlog_amount: unknown): Promise<string> {
        return ((nlog_amount === null ? 0 : nlog_amount) as number).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateSumNlogAmount",
        }
    }
}