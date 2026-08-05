import type DnoteAggregateTarget from "../../classes/dnote/dnote-aggregate-target"
import AggregateCountKyou from "../../classes/dnote/dnote-aggregate-target/aggregate-count-kyou"
import type DnotePredicate from "../../classes/dnote/dnote-predicate"
import AndPredicate from "../../classes/dnote/dnote-predicate/and-predicate"
import type { DnoteTrendChartType, DnoteTrendGranularity } from "../../classes/dnote/dnote-trend/dnote-trend-types"

export default class DnoteTrendGraphQuery {
    id: string = ""
    title: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    aggregate_target: DnoteAggregateTarget = new AggregateCountKyou()
    granularity: DnoteTrendGranularity = 'day'
    chart_type: DnoteTrendChartType = 'line'
}
