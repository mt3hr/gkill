import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class LantanaMoodEqualPredicate implements DnotePredicate {
    private lantana_mood_equal_target: number
    constructor(lantana_mood_equal_target: number) {
        this.lantana_mood_equal_target = lantana_mood_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const lantana_mood_equal_target = json.lantana_mood_equal_target as number
        return new LantanaMoodEqualPredicate(lantana_mood_equal_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const lantana_mood = loaded_kyou.typed_lantana?.mood
        if (lantana_mood) {
            if (lantana_mood.valueOf() === this.lantana_mood_equal_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "LantanaMoodEqualPredicate",
            lantana_mood_equal_target: this.lantana_mood_equal_target,
        }
    }
}