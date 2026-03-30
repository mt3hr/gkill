import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class LantanaMoodEqualPredicate implements DnotePredicate {
    private lantana_mood_equal_target: number
    constructor(lantana_mood_equal_target: number) {
        this.lantana_mood_equal_target = lantana_mood_equal_target
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static from_json(json: any): DnotePredicate {
        const lantana_mood_equal_target = json.value as number
        return new LantanaMoodEqualPredicate(lantana_mood_equal_target)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const lantana_mood = loaded_kyou.typed_lantana?.mood
        if (lantana_mood) {
            if (lantana_mood.valueOf() === this.lantana_mood_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "LantanaMoodEqualPredicate",
            value: this.lantana_mood_equal_target,
        }
    }
}