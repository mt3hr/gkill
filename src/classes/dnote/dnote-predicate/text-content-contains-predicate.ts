import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class TextContentContainsPredicate implements DnotePredicate {
    private text_content_contains_target: string
    constructor(text_content_contains_target: string) {
        this.text_content_contains_target = text_content_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const text_content_contains_target = json.value as string
        return new TextContentContainsPredicate(text_content_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        if (loaded_kyou.attached_texts) {
            for (let i = 0; i < loaded_kyou.attached_texts.length; i++) {
                const text_content = loaded_kyou.attached_texts[i].text
                if (text_content) {
                    if (text_content?.includes(this.text_content_contains_target)) {
                        return true
                    }
                }
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "TextContentContainsPredicate",
            value: this.text_content_contains_target,
        }
    }
}