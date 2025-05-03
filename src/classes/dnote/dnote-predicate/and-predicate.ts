import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class AndPredicate implements DnotePredicate {
    private predicates: Array<DnotePredicate>
    constructor(predicates: Array<DnotePredicate>) {
        this.predicates = predicates
    }
    static from_json(json: any): DnotePredicate {
        const children = json.predicates.map((j: any) => PredicateDictonary.get(j.type).from_json(j));
        return new AndPredicate(children);
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        for (const predicate of this.predicates) {
            if (!(await predicate.is_match(loaded_kyou))) {
                return false
            }
        }
        return true
    }
    predicate_struct_to_json(): any {
        return {
            type: "AndPredicate",
            predicates: this.predicates.map(p => (p as unknown as DnotePredicate).predicate_struct_to_json())
        }
    }
}
