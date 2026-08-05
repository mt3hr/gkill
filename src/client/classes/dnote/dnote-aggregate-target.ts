import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";

export default interface DnoteAggregateTarget {
    append_aggregate_element_value(aggregated_value: unknown, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<unknown>
    result_to_string(aggregated_value: unknown): Promise<string>
    to_json(): Record<string, unknown>
}