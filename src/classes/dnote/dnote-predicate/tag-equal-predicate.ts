import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class TagPredicate implements DnotePredicate {
    private tag: string
    constructor(tag: string) {
        this.tag = tag
    }
    static from_json(json: any): DnotePredicate {
        const tag = json.tag as string
        return new TagPredicate(tag)
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
    to_json(): any {
        return {
            type: "TagPredicate",
            tag: this.tag,
        }
    }
}