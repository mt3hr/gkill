import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class KmemoContentEqualPredicate implements DnotePredicate {
    private kmemo_content_equal_target: string
    constructor(kmemo_content_equal_target: string) {
        this.kmemo_content_equal_target = kmemo_content_equal_target
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static from_json(json: any): DnotePredicate {
        const kmemo_content_equal_target = json.value as string
        return new KmemoContentEqualPredicate(kmemo_content_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const kmemo_content = loaded_kyou.typed_kmemo?.content
        if (kmemo_content) {
            if (kmemo_content === this.kmemo_content_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "KmemoContentEqualPredicate",
            value: this.kmemo_content_equal_target,
        }
    }
}