import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class OrPredicate implements DnotePredicate {
    private predicates: Array<DnotePredicate>
    constructor(predicates: Array<DnotePredicate>) {
        this.predicates = predicates
    }
    static from_json(json: any): DnotePredicate {
        const children = json.predicates.map((j: any) => PredicateDictonary.get(j.type).from_json(j));
        return new OrPredicate(children);
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        for (const predicate of this.predicates) {
            if ((await predicate.is_match(loaded_kyou))) {
                return true
            }
        }
        return false
    }
    to_json(): any {
        return {
            type: "OrPredicate",
            predicates: this.predicates.map(p => (p as unknown as DnotePredicate).to_json())
        }
    }
}