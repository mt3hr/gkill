import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class NlogTitleEqualPredicate implements DnotePredicate {
    private nlog_title_equal_target: string
    constructor(nlog_title_equal_target: string) {
        this.nlog_title_equal_target = nlog_title_equal_target
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static from_json(json: any): DnotePredicate {
        const nlog_title_equal_target = json.value as string
        return new NlogTitleEqualPredicate(nlog_title_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const nlog_title = loaded_kyou.typed_nlog?.title
        if (nlog_title) {
            if (nlog_title === this.nlog_title_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "NlogTitleEqualPredicate",
            value: this.nlog_title_equal_target,
        }
    }
}