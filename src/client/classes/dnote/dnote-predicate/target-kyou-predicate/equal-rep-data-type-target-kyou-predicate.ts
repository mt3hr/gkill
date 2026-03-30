import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../../dnote-predicate";

export default class EqualDataTypeTargetKyouPredicate implements DnotePredicate {
    constructor() { }
    static from_json(_: Record<string, unknown>): DnotePredicate {
        return new EqualDataTypeTargetKyouPredicate()
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (!target_kyou) {
            return false
        }
        const target_kyou_data_type = target_kyou.data_type.split("_")[0]
        const loaded_kyou_data_type = loaded_kyou.data_type.split("_")[0]
        if (target_kyou_data_type.startsWith(loaded_kyou_data_type)) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "EqualDataTypeTargetKyouPredicate",
        }
    }
}