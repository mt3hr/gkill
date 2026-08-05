import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";

export default class AggregateSumLantanaMood implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateSumLantanaMood()
    }
    async append_aggregate_element_value(aggregated_value_lantana_mood: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const typed_aggregated_value_lantana_mood = aggregated_value_lantana_mood === null ? 0 : aggregated_value_lantana_mood as number
        let mood = 0
        if (kyou.typed_lantana) {
            mood += kyou.typed_lantana.mood.valueOf()
        }
        return typed_aggregated_value_lantana_mood + mood
    }
    async result_to_string(lantana_mood: unknown): Promise<string> {
        return ((lantana_mood === null ? 0 : lantana_mood) as number).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateSumLantanaMood",
        }
    }
}