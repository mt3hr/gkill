import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class NotPredicate implements DnotePredicate {
    private predicate: DnotePredicate
    constructor(predicate: DnotePredicate) {
        this.predicate = predicate
    }
    static from_json(json: any): DnotePredicate {
        const predicate = PredicateDictonary.get(json.predicate.type).from_json(json)
        return new NotPredicate(predicate)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        return await !(this.predicate.is_match(loaded_kyou))
    }
    predicate_struct_to_json(): any {
        return {
            type: "NotPredicate",
            predicate: (this.predicate as unknown as DnotePredicate).predicate_struct_to_json()
        }
    }
}