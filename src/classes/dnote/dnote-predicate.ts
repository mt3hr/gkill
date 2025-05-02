import type { Kyou } from "../datas/kyou";
import AndPredicate from "./dnote-predicate/and-predicate";
import NotPredicate from "./dnote-predicate/not-predicate";
import OrPredicate from "./dnote-predicate/or-predicate";
import PredicateDictonary from "./serialize/dnote-predicate-dictionary";

export default interface DnotePredicate {
    is_match(loaded_kyou: Kyou): Promise<boolean>
    to_json(): any
}

export function build_dnote_predicate_from_json(json: any): DnotePredicate {
    if ('logic' in json && Array.isArray(json.predicates)) {
        const children = json.predicates.map(build_dnote_predicate_from_json)
        if (json.logic === 'AND') return new AndPredicate(children)
        if (json.logic === 'OR') return new OrPredicate(children)
        if (json.logic === 'NOT') return new NotPredicate(children)
        throw new Error(`Unknown logic type: ${json.logic}`)
    }

    const ctor = PredicateDictonary.get(json.type)
    if (!ctor) throw new Error(`Unknown predicate type: ${json.type}`)
    return ctor(json.value)
}