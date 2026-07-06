import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../../dnote-predicate";

export default class EqualIdTargetKyouPredicate implements DnotePredicate {
    constructor() { }
    static from_json(_: Record<string, unknown>): DnotePredicate {
        return new EqualIdTargetKyouPredicate()
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (!target_kyou) {
            return false
        }
        return loaded_kyou.id === target_kyou.id
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "EqualIdTargetKyouPredicate",
        }
    }
}
