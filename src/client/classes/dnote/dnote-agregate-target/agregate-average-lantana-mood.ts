import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import AverageInfo from "./average-info";

export default class AgregateAverageLantanaMood implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateAverageLantanaMood()
    }
    async append_agregate_element_value(typed_average_info_lantana_mood: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
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
            type: "AgregateAverageLantanaMood",
        }
    }
}