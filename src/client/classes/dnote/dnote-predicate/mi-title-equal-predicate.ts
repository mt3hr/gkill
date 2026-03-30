import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class MiTitleEqualPredicate implements DnotePredicate {
    private mi_title_equal_target: string
    constructor(mi_title_equal_target: string) {
        this.mi_title_equal_target = mi_title_equal_target
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static from_json(json: any): DnotePredicate {
        const mi_title_equal_target = json.value as string
        return new MiTitleEqualPredicate(mi_title_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const mi_title = loaded_kyou.typed_mi?.title
        if (mi_title) {
            if (mi_title === this.mi_title_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "MiTitleEqualPredicate",
            value: this.mi_title_equal_target,
        }
    }
}