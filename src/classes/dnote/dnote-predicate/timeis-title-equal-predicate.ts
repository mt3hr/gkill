import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class TimeIsTitleEqualPredicate implements DnotePredicate {
    private timeis_title_equal_target: string
    constructor(timeis_title_equal_target: string) {
        this.timeis_title_equal_target = timeis_title_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const timeis_title_equal_target = json.value as string
        return new TimeIsTitleEqualPredicate(timeis_title_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const timeis_title = loaded_kyou.typed_timeis?.title
        if (timeis_title) {
            if (timeis_title === this.timeis_title_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "TimeIsTitleEqualPredicate",
            value: this.timeis_title_equal_target,
        }
    }
}