'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type DnoteTrendGraphQuery from "./dnote-trend-graph-query"

export interface ConfirmDeleteDnoteTrendGraphViewProps extends GkillPropsBase {
    dnote_trend_graph_query: DnoteTrendGraphQuery
}
