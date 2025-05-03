import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class MiTitleContainsPredicate implements DnotePredicate {
    private mi_title_contains_target: string
    constructor(mi_title_contains_target: string) {
        this.mi_title_contains_target = mi_title_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const mi_title_contains_target = json.mi_title_contains_target as string
        return new MiTitleContainsPredicate(mi_title_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const mi_title = loaded_kyou.typed_mi?.title
        if (mi_title) {
            if (mi_title?.includes(this.mi_title_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "MiTitleContainsPredicate",
            mi_title_contains_target: this.mi_title_contains_target,
        }
    }
}