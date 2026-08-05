import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";

export default class AggregateCountKyou implements DnoteAggregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAggregateTarget {
        return new AggregateCountKyou()
    }
    async append_aggregate_element_value(kyou_count: unknown, _kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        return (kyou_count === null ? 0 : (kyou_count as number)) + 1
    }
    async result_to_string(kyou_count: unknown): Promise<string> {
        return (kyou_count === null ? 0 : (kyou_count as number)).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AggregateCountKyou",
        }
    }
}