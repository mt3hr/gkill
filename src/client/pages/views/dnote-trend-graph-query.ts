import type DnoteAgregateTarget from "../../classes/dnote/dnote-agregate-target"
import AgregateCountKyou from "../../classes/dnote/dnote-agregate-target/agregate-count-kyou"
import type DnotePredicate from "../../classes/dnote/dnote-predicate"
import AndPredicate from "../../classes/dnote/dnote-predicate/and-predicate"
import type { DnoteTrendChartType, DnoteTrendGranularity } from "../../classes/dnote/dnote-trend/dnote-trend-types"

export default class DnoteTrendGraphQuery {
    id: string = ""
    title: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    aggregate_target: DnoteAgregateTarget = new AgregateCountKyou()
    granularity: DnoteTrendGranularity = 'day'
    chart_type: DnoteTrendChartType = 'line'
}
