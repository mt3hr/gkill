import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import { i18n } from "@/i18n";
import AverageInfo from "./average-info";
import AggregateTargetDictionary from "../serialize/dnote-aggregate-target-dictionary";

export default class AggregateAverageLantanaMood implements DnoteAggregateTarget {
    static from_json(_json: any): DnoteAggregateTarget {
        return new AggregateAverageLantanaMood()
    }
    async append_aggregate_element_value(typed_average_info_lantana_mood: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_lantana_mood = typed_average_info_lantana_mood === null ? new AverageInfo() : (typed_average_info_lantana_mood as AverageInfo).clone()
        cloned_typed_average_info_lantana_mood.total_value = cloned_typed_average_info_lantana_mood.total_value === null ? 0 : cloned_typed_average_info_lantana_mood.total_value as number

        let mood = 0
        if (kyou.typed_lantana) {
            mood += kyou.typed_lantana.mood.valueOf()

            cloned_typed_average_info_lantana_mood.total_value += mood
            cloned_typed_average_info_lantana_mood.total_count++
        }
        return cloned_typed_average_info_lantana_mood
    }
    async result_to_string(typed_average_info_lantana_mood: any | null): Promise<string> {
        const cloned_typed_average_info_lantana_mood = typed_average_info_lantana_mood === null ? new AverageInfo() : (typed_average_info_lantana_mood as AverageInfo).clone()
        cloned_typed_average_info_lantana_mood.total_value = cloned_typed_average_info_lantana_mood.total_value === null ? 0 : cloned_typed_average_info_lantana_mood.total_value as number
        return (cloned_typed_average_info_lantana_mood.total_count === 0 ? 0 : (cloned_typed_average_info_lantana_mood.total_value / cloned_typed_average_info_lantana_mood.total_count)).toString()
    }
    to_json(): any {
        return {
            type: "AggregateAverageLantanaMood",
        }
    }
}