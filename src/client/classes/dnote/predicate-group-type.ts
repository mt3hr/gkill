import type Predicate from "./predicate"

export default interface PredicateGroupType {
    logic: 'AND' | 'OR' | 'NOT'
    predicates: Array<Predicate | PredicateGroupType>
}

