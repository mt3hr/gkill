import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class NotPredicate implements DnotePredicate {
    private predicates: Array<DnotePredicate> = []
    constructor(predicates: Array<DnotePredicate>) {
        this.predicates = predicates
    }
    static from_json(json: any): DnotePredicate {
        let children = new Array<DnotePredicate>()
        if (json.predicates) {
            children = json.predicates.map((j: any) => PredicateDictonary.get(j.type).from_json(j));
        }
        return new NotPredicate(children);
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (!this.predicates || this.predicates.length === 0) {
            return true
        }
        for (const predicate of this.predicates) {
            if ((await predicate.is_match(loaded_kyou, target_kyou))) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            logic: 'NOT',
            type: "NotPredicate",
            predicates: ((this.predicates && this.predicates.length !== 0) ? this.predicates.map(p => (p as unknown as DnotePredicate).predicate_struct_to_json()) : [])
        }
    }
}