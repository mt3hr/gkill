import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class KmemoContentContainsPredicate implements DnotePredicate {
    private kmemo_content_contains_target: string
    constructor(kmemo_content_contains_target: string) {
        this.kmemo_content_contains_target = kmemo_content_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const kmemo_content_contains_target = json.value as string
        return new KmemoContentContainsPredicate(kmemo_content_contains_target)
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        const kmemo_content = loaded_kyou.typed_kmemo?.content
        if (kmemo_content) {
            if (kmemo_content?.includes(this.kmemo_content_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "KmemoContentContainsPredicate",
            value: this.kmemo_content_contains_target,
        }
    }
}