import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class NlogAmountGreaterThanPredicate implements DnotePredicate {
    private nlog_amount: number
    constructor(nlog_amount: number) {
        this.nlog_amount = nlog_amount
    }
    static from_json(json: any): DnotePredicate {
        const nlog_amount = json.value as number
        return new NlogAmountGreaterThanPredicate(nlog_amount)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const nlog_amount = loaded_kyou.typed_nlog?.amount
        if (nlog_amount) {
            if (nlog_amount >= this.nlog_amount) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "NlogAmountGreaterThanPredicate",
            value: this.nlog_amount,
        }
    }
}