import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class KmemoContentEqualPredicate implements DnotePredicate {
    private kmemo_content_equal_target: string
    constructor(kmemo_content_equal_target: string) {
        this.kmemo_content_equal_target = kmemo_content_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const kmemo_content_equal_target = json.kmemo_content_equal_target as string
        return new KmemoContentEqualPredicate(kmemo_content_equal_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const kmemo_content = loaded_kyou.typed_kmemo?.content
        if (kmemo_content) {
            if (kmemo_content === this.kmemo_content_equal_target) {
                return true
            }
        }
        return false
    }
    to_json(): any {
        return {
            type: "KmemoContentEqualPredicate",
            kmemo_content_equal_target: this.kmemo_content_equal_target,
        }
    }
}