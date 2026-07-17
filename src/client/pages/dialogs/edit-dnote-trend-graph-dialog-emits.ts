import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"
import type DnoteTrendGraphQuery from "../views/dnote-trend-graph-query"

export default interface EditDnoteTrendGraphDialogEmits {
    (e: 'requested_update_dnote_trend_graph', dnote_trend_graph_query: DnoteTrendGraphQuery): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
