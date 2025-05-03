import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";

export default interface DnoteAggregateTarget {
    append_aggregate_element_value(aggregated_value: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any>
    result_to_string(aggregated_value: any | null): Promise<string>
    to_json(): any
}