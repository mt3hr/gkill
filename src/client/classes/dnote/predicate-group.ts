import type PredicateCard from "./predicate-card";

export default interface PredicateGroup {
    logic: 'AND' | 'OR' | 'NOT',
    predicates: Array<PredicateCard | PredicateGroup>
}