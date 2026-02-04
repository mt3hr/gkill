import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class KCTitleEqualPredicate implements DnotePredicate {
    private kc_title_equal_target: string
    constructor(kc_title_equal_target: string) {
        this.kc_title_equal_target = kc_title_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const kc_title_equal_target = json.value as string
        return new KCTitleEqualPredicate(kc_title_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const kc_title = loaded_kyou.typed_kc?.title
        if (kc_title) {
            if (kc_title === this.kc_title_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "KCTitleEqualPredicate",
            value: this.kc_title_equal_target,
        }
    }
}