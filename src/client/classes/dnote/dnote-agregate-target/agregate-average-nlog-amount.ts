import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import AverageInfo from "./average-info";

export default class AgregateAverageNlogAmount implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateAverageNlogAmount()
    }
    async append_agregate_element_value(typed_average_info_nlog_amount: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
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
    async result_to_string(typed_average_info_nlog_amount: any | null): Promise<string> {
        const cloned_typed_average_info_nlog_amount = typed_average_info_nlog_amount === null ? new AverageInfo() : (typed_average_info_nlog_amount as AverageInfo).clone()
        cloned_typed_average_info_nlog_amount.total_value = cloned_typed_average_info_nlog_amount.total_value === null ? 0 : cloned_typed_average_info_nlog_amount.total_value as number
        return (cloned_typed_average_info_nlog_amount.total_count === 0 ? 0 : (cloned_typed_average_info_nlog_amount.total_value / cloned_typed_average_info_nlog_amount.total_count)).toString()
    }
    to_json(): any {
        return {
            type: "AgregateAverageNlogAmount",
        }
    }
}