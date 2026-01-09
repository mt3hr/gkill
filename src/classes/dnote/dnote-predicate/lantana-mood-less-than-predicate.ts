import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class LantanaMoodLessThanPredicate implements DnotePredicate {
    private lantana_mood_less_than_target: number
    constructor(lantana_mood_less_than_target: number) {
        this.lantana_mood_less_than_target = lantana_mood_less_than_target
    }
    static from_json(json: any): DnotePredicate {
        const lantana_mood_less_than_target = json.value as number
        return new LantanaMoodLessThanPredicate(lantana_mood_less_than_target)
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        const lantana_mood = loaded_kyou.typed_lantana?.mood
        if (lantana_mood) {
            if (lantana_mood.valueOf() <= this.lantana_mood_less_than_target) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "LantanaMoodLessThanPredicate",
            value: this.lantana_mood_less_than_target,
        }
    }
}