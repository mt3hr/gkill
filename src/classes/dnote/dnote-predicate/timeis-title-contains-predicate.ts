import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class TimeIsTitleContainsPredicate implements DnotePredicate {
    private timeis_title_contains_target: string
    constructor(timeis_title_contains_target: string) {
        this.timeis_title_contains_target = timeis_title_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const timeis_title_contains_target = json.timeis_title_contains_target as string
        return new TimeIsTitleContainsPredicate(timeis_title_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const timeis_title = loaded_kyou.typed_timeis?.title
        if (timeis_title) {
            if (timeis_title?.includes(this.timeis_title_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "TimeIsTitleContainsPredicate",
            timeis_title_contains_target: this.timeis_title_contains_target,
        }
    }
}