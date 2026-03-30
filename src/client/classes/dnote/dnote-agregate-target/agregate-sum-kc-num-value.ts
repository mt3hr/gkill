import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";

export default class AgregateSumKCNumValue implements DnoteAgregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAgregateTarget {
        return new AgregateSumKCNumValue()
    }
    async append_agregate_element_value(agregated_value_kc_num_value: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_agregated_value_kc_num_value = agregated_value_kc_num_value === null ? 0 : agregated_value_kc_num_value as number
        let num_value = 0
        if (kyou.typed_kc) {
            num_value += kyou.typed_kc.num_value
        }
        return typed_agregated_value_kc_num_value + num_value
    }
    async result_to_string(kc_num_value: unknown): Promise<string> {
        return ((kc_num_value === null ? 0 : kc_num_value) as number).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AgregateSumKCNumValue",
        }
    }
}