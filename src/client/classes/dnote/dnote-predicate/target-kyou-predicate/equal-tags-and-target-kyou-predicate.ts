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
        // Kyouは複数のタグを持つ。双方が同じタグ集合を持つ（互いのタグがすべて相手にも憑いている）場合のみTrue
        const loaded_tag_names = loaded_kyou.attached_tags.map(tag => tag.tag)
        const target_tag_names = target_kyou.attached_tags.map(tag => tag.tag)

        const all_loaded_in_target = loaded_tag_names.every(tag => target_tag_names.includes(tag))
        const all_target_in_loaded = target_tag_names.every(tag => loaded_tag_names.includes(tag))
        return all_loaded_in_target && all_target_in_loaded
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "EqualTagsAndTargetKyouPredicate",
        }
    }
}
