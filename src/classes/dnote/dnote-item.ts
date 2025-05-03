import type DnoteAggregateTarget from "./dnote-aggregate-target"
import AggregateCountKyou from "./dnote-aggregate-target/aggregate-count-kyou"
import type DnotePredicate from "./dnote-predicate"
import AndPredicate from "./dnote-predicate/and-predicate"

export default class DnoteItem {
    id: string = ""
    title: string = ""
    prefix: string = ""
    suffix: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    aggregate_target: DnoteAggregateTarget = new AggregateCountKyou()
}