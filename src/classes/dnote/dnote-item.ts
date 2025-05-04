import type DnoteAgregateTarget from "./dnote-agregate-target"
import AgregateCountKyou from "./dnote-agregate-target/agregate-count-kyou"
import type DnotePredicate from "./dnote-predicate"
import AndPredicate from "./dnote-predicate/and-predicate"

export default class DnoteItem {
    id: string = ""
    title: string = ""
    prefix: string = ""
    suffix: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    agregate_target: DnoteAgregateTarget = new AgregateCountKyou()
}