import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../../dnote-predicate";

export default class EqualTagsTargetKyouPredicate implements DnotePredicate {
    private and: boolean
    constructor(and: boolean) {
        this.and = and
    }

    static from_json(json: any): DnotePredicate {
        const and = json.and as boolean
        return new EqualTagsTargetKyouPredicate(and)
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

        let match_or = false
        let match_and = true
        for (let i = 0; i < loaded_tags.length; i++) {
            const loaded_tag = loaded_tags[i]
            for (let j = 0; j < target_tags.length; j++) {
                const target_tag = target_tags[j]
                if (loaded_tag.tag === target_tag.tag) {
                    match_or = true
                } else {
                    match_and = false
                    if (this.and) {
                        return false
                    }
                }
            }
        }

        if (this.and && match_and) {
            return true
        }
        if (!this.and && match_or) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "EqualTagsTargetKyouPredicate",
            value: this.and,
        }
    }
}