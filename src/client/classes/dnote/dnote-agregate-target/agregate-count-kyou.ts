import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";

export default class AgregateCountKyou implements DnoteAgregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAgregateTarget {
        return new AgregateCountKyou()
    }
    async append_agregate_element_value(kyou_count: unknown, _kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        return (kyou_count === null ? 0 : (kyou_count as number)) + 1
    }
    async result_to_string(kyou_count: unknown): Promise<string> {
        return (kyou_count === null ? 0 : (kyou_count as number)).toString()
    }
    to_json(): Record<string, unknown> {
        return {
            type: "AgregateCountKyou",
        }
    }
}