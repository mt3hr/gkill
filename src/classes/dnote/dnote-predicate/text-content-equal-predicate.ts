import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class TextContentEqualPredicate implements DnotePredicate {
    private text_content_equal_target: string
    constructor(text_content_equal_target: string) {
        this.text_content_equal_target = text_content_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const text_content_equal_target = json.value as string
        return new TextContentEqualPredicate(text_content_equal_target)
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (loaded_kyou.attached_texts) {
            for (let i = 0; i < loaded_kyou.attached_texts.length; i++) {
                const text_content = loaded_kyou.attached_texts[i].text
                if (text_content) {
                    if (text_content == this.text_content_equal_target) {
                        return true
                    }
                }
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "TextContentEqualPredicate",
            value: this.text_content_equal_target,
        }
    }
}