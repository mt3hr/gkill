import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import AgregateTargetDictionary from "../serialize/dnote-aggregate-target-dictionary";

export default class AgregateCountKyou implements DnoteAgregateTarget {
    static from_json(_json: any): DnoteAgregateTarget {
        return new AgregateCountKyou()
    }
    async append_agregate_element_value(kyou_count: any | null, _kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        return (kyou_count === null ? 0 : (kyou_count as number)) + 1
    }
    async result_to_string(kyou_count: any | null): Promise<string> {
        return (kyou_count === null ? 0 : (kyou_count as number)).toString()
    }
    to_json(): any {
        return {
            type: "AgregateCountKyou",
        }
    }
}

// 循環参照解決のためにここで定義
export function build_dnote_aggretgate_target_from_json(json: any): DnoteAgregateTarget {
    const ctor = AgregateTargetDictionary.get(json.type)
    if (!ctor) throw new Error(`Unknown predicate type: ${json.type}`)
    return ctor(json.value)
}