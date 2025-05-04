import type DnoteAgregateTarget from "../../classes/dnote/dnote-agregate-target"
import AgregateCountKyou from "../../classes/dnote/dnote-agregate-target/agregate-count-kyou"
import type DnoteKeyGetter from "../../classes/dnote/dnote-key-getter"
import TitleGetter from "../../classes/dnote/dnote-key-getter/title-getter"
import type DnotePredicate from "../../classes/dnote/dnote-predicate"
import AndPredicate from "../../classes/dnote/dnote-predicate/and-predicate"

export default class DnoteListQuery {
    id: string = ""
    title: string = ""
    prefix: string = ""
    suffix: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    key_getter: DnoteKeyGetter = new TitleGetter()
    aggregate_target: DnoteAgregateTarget = new AgregateCountKyou()
}