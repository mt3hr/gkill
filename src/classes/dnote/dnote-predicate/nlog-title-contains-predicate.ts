import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class NlogTitleContainsPredicate implements DnotePredicate {
    private nlog_title_contains_target: string
    constructor(nlog_title_contains_target: string) {
        this.nlog_title_contains_target = nlog_title_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const nlog_title_contains_target = json.nlog_title_contains_target as string
        return new NlogTitleContainsPredicate(nlog_title_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const nlog_title = loaded_kyou.typed_nlog?.title
        if (nlog_title) {
            if (nlog_title?.includes(this.nlog_title_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "NlogTitleContainsPredicate",
            nlog_title_contains_target: this.nlog_title_contains_target,
        }
    }
}