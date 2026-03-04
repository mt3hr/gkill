import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";

export default class AgregateSumNlogAmount implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateSumNlogAmount()
    }
    async append_agregate_element_value(agregated_value_nlog_amount: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_agregated_value_nlog_amount = agregated_value_nlog_amount === null ? 0 : agregated_value_nlog_amount as number
        let amount = 0
        if (kyou.typed_nlog) {
            amount += kyou.typed_nlog.amount
        }
        return typed_agregated_value_nlog_amount + amount
    }
    async result_to_string(nlog_amount: any | null): Promise<string> {
        return (nlog_amount === null ? 0 : nlog_amount).toString()
    }
    to_json(): any {
        return {
            type: "AgregateSumNlogAmount",
        }
    }
}