import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import AggregateTargetDictionary from "./serialize/dnote-aggregate-target-dictionary";

export default interface DnoteAggregateTarget {
    append_aggregate_element_value(aggregated_value: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any>
    result_to_string(aggregated_value: any | null): Promise<string>
    to_json(): any
}
export function build_dnote_aggregate_target_from_json(json: any): DnoteAggregateTarget {
    const ctor = AggregateTargetDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown predicate type: ${json.type}`)
    return ctor(json.value)
}