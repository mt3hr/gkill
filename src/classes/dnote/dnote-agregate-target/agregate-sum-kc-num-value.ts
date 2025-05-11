import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";

export default class AgregateSumKCNumValue implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateSumKCNumValue()
    }
    async append_agregate_element_value(agregated_value_kc_num_value: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_agregated_value_kc_num_value = agregated_value_kc_num_value === null ? 0 : agregated_value_kc_num_value as number
        let num_value = 0
        if (kyou.typed_kc) {
            num_value += kyou.typed_kc.num_value
        }
        return typed_agregated_value_kc_num_value + num_value
    }
    async result_to_string(kc_num_value: any | null): Promise<string> {
        return (kc_num_value === null ? 0 : kc_num_value).toString()
    }
    to_json(): any {
        return {
            type: "AgregateSumKCNumValue",
        }
    }
}