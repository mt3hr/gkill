import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class KCTitleContainsPredicate implements DnotePredicate {
    private kc_title_contains_target: string
    constructor(kc_title_contains_target: string) {
        this.kc_title_contains_target = kc_title_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const kc_title_contains_target = json.value as string
        return new KCTitleContainsPredicate(kc_title_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const kc_title = loaded_kyou.typed_kc?.title
        if (kc_title) {
            if (kc_title?.includes(this.kc_title_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "KCTitleContainsPredicate",
            value: this.kc_title_contains_target,
        }
    }
}