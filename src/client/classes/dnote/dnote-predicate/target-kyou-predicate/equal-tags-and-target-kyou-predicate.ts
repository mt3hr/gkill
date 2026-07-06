import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../../dnote-predicate";

export default class EqualTagsAndTargetKyouPredicate implements DnotePredicate {
    constructor() { }
    static from_json(_: Record<string, unknown>): DnotePredicate {
        return new EqualTagsAndTargetKyouPredicate()
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (!target_kyou) {
            return false
        }
        const loaded_tags = loaded_kyou.attached_tags
        const target_tags = target_kyou?.attached_tags

        if (target_tags.length === 0 && loaded_tags.length === 0) {
            return true
        }

        let match_and = true
        for (let i = 0; i < loaded_tags.length; i++) {
            const loaded_tag = loaded_tags[i]
            for (let j = 0; j < target_tags.length; j++) {
                const target_tag = target_tags[j]
                if (loaded_tag.tag !== target_tag.tag) {
                    match_and = false
                    return false
                }
            }
        }

        return match_and
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "EqualTagsAndTargetKyouPredicate",
        }
    }
}
