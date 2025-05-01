import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import { i18n } from "@/i18n";

export default class AggregateSumNlogAmount implements DnoteAggregateTarget {
    from_json(_json: any): DnoteAggregateTarget {
        return new AggregateSumNlogAmount()
    }
    async append_aggregate_element_value(aggregated_value_nlog_amount: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_aggregated_value_nlog_amount = aggregated_value_nlog_amount === null ? 0 : aggregated_value_nlog_amount as number
        let amount = 0
        if (kyou.typed_nlog) {
            amount += kyou.typed_nlog.amount
        }
        return typed_aggregated_value_nlog_amount + amount
    }
    async result_to_string(nlog_amount: any | null): Promise<string> {
        return (nlog_amount === null ? 0 : nlog_amount) + i18n.global.t("YEN_TITLE")
    }
    to_json(): any {
        return {
            type: "AggregateSumNlogAmount",
        }
    }
}