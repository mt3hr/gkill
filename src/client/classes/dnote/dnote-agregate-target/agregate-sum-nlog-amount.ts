import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";

export default class AgregateSumNlogAmount implements DnoteAgregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAgregateTarget {
        return new AgregateSumNlogAmount()
    }
    async append_agregate_element_value(agregated_value_nlog_amount: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_agregated_value_nlog_amount = agregated_value_nlog_amount === null ? 0 : agregated_value_nlog_amount as number
        let amount = 0
        if (kyou.typed_nlog) {
            amount += kyou.typed_nlog.amount
        }
        return typed_agregated_value_nlog_amount + amount
    }
    async result_to_string(nlog_amount: unknown): Promise<string> {
        return ((nlog_amount === null ? 0 : nlog_amount) as number).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AgregateSumNlogAmount",
        }
    }
}