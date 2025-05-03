import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class TagEqualPredicate implements DnotePredicate {
    private tag: string
    constructor(tag: string) {
        this.tag = tag
    }
    static from_json(json: any): DnotePredicate {
        const tag = json.tag as string
        return new TagEqualPredicate(tag)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        for (let i = 0; i < loaded_kyou.attached_tags.length; i++) {
            const tag = loaded_kyou.attached_tags[i]
            if (tag.tag === this.tag) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "TagEqualPredicate",
            tag: this.tag,
        }
    }
}